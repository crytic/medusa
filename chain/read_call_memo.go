package chain

import (
	"bytes"
	"math/big"
	"slices"

	"github.com/crytic/medusa-geth/common"
	"github.com/crytic/medusa-geth/core"
	gethState "github.com/crytic/medusa-geth/core/state"
	"github.com/crytic/medusa-geth/core/tracing"
	"github.com/crytic/medusa-geth/core/vm"
	"github.com/crytic/medusa-geth/params"
	"github.com/holiman/uint256"
)

// allowReadCallMemo limits reuse to a native in-memory chain with no observer beyond the built-in cheat
// tracer. A collector rejects any actual use of a cheat/precompile,
// a write, a context-dependent/unknown opcode, or an unsuccessful child call.
// ApplyMessage still validates and prepares every transaction. With access
// lists and authorizations excluded, initial gas warmth is determined by the
// sender, destination, coinbase, rules and active precompile warming list.
// Delegated code cannot validate against the recorded executed bytes.
func (t *TestChain) allowReadCallMemo(msg *core.Message, extra []*TestChainTracer) bool {
	if msg.To == nil || len(extra) != 0 || len(msg.AccessList) != 0 || len(msg.SetCodeAuthorizations) != 0 || len(msg.BlobHashes) != 0 || len(msg.Data) > 4096 {
		return false
	}
	// Native state can also collect witnesses independently of fork rules. Such
	// collectors observe execution and must retain the ordinary path.
	native, ok := t.state.(*gethState.StateDB)
	if !ok || native.Witness() != nil || native.AccessEvents() != nil {
		return false
	}
	return t.onlyBuiltinCallTracer()
}

// readCallKey captures every supported input and the initial gas-warmth context.
type readCallKey struct {
	caller, address, coinbase common.Address
	value                     uint256.Int
	gas                       uint64
	input                     string
	readOnly                  bool
	rules                     params.Rules // ChainID is nil: CHAINID is deliberately unsupported.
}

// codeRead owns the complete code bytes used at an address.
type codeRead struct {
	address common.Address
	code    []byte
}

// slotRead records an exact storage dependency.
type slotRead struct {
	address     common.Address
	slot, value common.Hash
}

// readCallEntry contains an interpreter result and its complete dependencies.
type readCallEntry struct {
	code               []codeRead
	slots              []slotRead
	output, returnData []byte
	gas                uint64
	children           int
}

// readCallMemo entries own their code and return buffers and belong to one TestChain. A hit
// compares every previously read slot and code body exactly, not by a digest.
// The opcode whitelist ensures those reads plus the key determine the entire
// path and its gas. All transient effects of successful STATICCALLs are inside
// CallContract's outer revert; only revision numbering needs explicit replay.
type readCallMemo struct {
	entries         map[readCallKey]*readCallEntry
	warmPrecompiles []common.Address
}

// newReadCallMemo creates a bounded, chain-local cache with independently owned entries.
func newReadCallMemo() *readCallMemo {
	return &readCallMemo{entries: make(map[readCallKey]*readCallEntry)}
}

// readCallRecorder proves one execution eligible while forwarding normal hooks.
type readCallRecorder struct {
	chain      *TestChain
	memo       *readCallMemo
	hooks      tracing.Hooks
	base       *tracing.Hooks
	evm        *vm.EVM
	key        readCallKey
	old, entry *readCallEntry
	frames     []bool
	calls      int
	bad, hit   bool
	codeBytes  int
}

// newReadCallRecorder wraps the built-in tracer for a single CallContract.
func newReadCallRecorder(chain *TestChain, memo *readCallMemo, base *tracing.Hooks) *readCallRecorder {
	r := &readCallRecorder{chain: chain, memo: memo, base: base, hooks: *base}
	r.hooks.OnEnter = r.onEnter
	r.hooks.OnExit = r.onExit
	r.hooks.OnOpcode = r.captureOpcode
	// The interpreter ignores OnOpcode when OnOpcodeForScope is set, so wrap the
	// base selection as well. Every opcode must still reach the dependency proof.
	if base.OnOpcodeForScope != nil {
		r.hooks.OnOpcodeForScope = r.captureOpcodeForScope
	}
	return r
}

// precompile checks the current native and extension dispatch tables.
func (r *readCallRecorder) precompile(address common.Address) bool {
	return r.evm.IsPrecompile(address)
}

// Lookup validates all dependencies after ordinary transaction preparation.
func (r *readCallRecorder) Lookup(evm *vm.EVM, c *vm.Contract, input []byte, readOnly bool, rules params.Rules) ([]byte, []byte, uint64, bool) {
	r.evm = evm
	rules.ChainID = nil
	r.key = readCallKey{c.Caller(), c.Address(), evm.Context.Coinbase, *c.Value(), c.Gas, string(input), readOnly, rules}
	// Prepare warms this exported list before execution. Its contents can be
	// changed independently of both the fork rules and the dispatch table.
	warm := vm.ActivePrecompiles(rules)
	if !slices.Equal(r.memo.warmPrecompiles, warm) {
		clear(r.memo.entries)
		r.memo.warmPrecompiles = slices.Clone(warm)
	}
	// Witness/access-event state and modified instruction tables need a separate
	// proof; always execute them normally.
	if rules.IsEIP4762 || rules.IsVerkle || evm.Config.EnablePreimageRecording || len(evm.Config.ExtraEips) != 0 || evm.Cancelled() {
		r.bad = true
	}
	r.old = r.memo.entries[r.key]
	if !r.bad && r.old != nil && r.chain.state.Error() == nil {
		valid := true
		for _, read := range r.old.code {
			if r.precompile(read.address) || !bytes.Equal(evm.StateDB.GetCode(read.address), read.code) {
				valid = false
				break
			}
		}
		if valid {
			for _, read := range r.old.slots {
				if evm.StateDB.GetState(read.address, read.slot) != read.value {
					valid = false
					break
				}
			}
		}
		if valid && r.chain.state.Error() == nil {
			// STATICCALL takes a revision even for a read-only child. Preserve the
			// revision sequence; the surrounding CallContract snapshot reverts them.
			for i := 0; i < r.old.children; i++ {
				evm.StateDB.Snapshot()
			}
			r.hit = true
			return bytes.Clone(r.old.output), bytes.Clone(r.old.returnData), r.old.gas, true
		}
	}
	r.entry = &readCallEntry{}
	return nil, nil, 0, false
}

// Store retains only successful, bounded executions with fully recorded frames.
func (r *readCallRecorder) Store(output, returnData []byte, gas uint64, err error) {
	if r.bad || r.hit || err != nil || r.entry == nil || r.chain.state.Error() != nil || len(output) > 65536 || len(returnData) > 65536 {
		return
	}
	if r.calls < 1 || len(r.frames) == 0 || !r.frames[0] {
		return
	}
	r.entry.output = bytes.Clone(output)
	r.entry.returnData = bytes.Clone(returnData)
	r.entry.gas = gas
	r.entry.children = r.calls - 1
	if len(r.memo.entries) >= 64 {
		clear(r.memo.entries)
	}
	r.memo.entries[r.key] = r.entry
}

// onEnter admits only STATICCALL descendants and bounds the recorded call tree.
func (r *readCallRecorder) onEnter(depth int, typ byte, from, to common.Address, input []byte, gas uint64, value *big.Int) {
	if r.base.OnEnter != nil {
		r.base.OnEnter(depth, typ, from, to, input, gas, value)
	}
	r.calls++
	if depth != 0 && vm.OpCode(typ) != vm.STATICCALL || r.calls > 64 {
		r.bad = true
	}
	r.frames = append(r.frames, false)
	// Precompile detection uses current rules in captureOpcode or onExit, after Lookup
	// initialized them. Empty/precompiled children have no scope and are rejected.
}

// onExit rejects failed, empty or precompiled child frames.
func (r *readCallRecorder) onExit(depth int, output []byte, gas uint64, err error, reverted bool) {
	if r.base.OnExit != nil {
		r.base.OnExit(depth, output, gas, err, reverted)
	}
	if !r.hit && (err != nil || reverted || len(r.frames) == 0 || !r.frames[len(r.frames)-1]) {
		r.bad = true
	}
	if len(r.frames) > 0 {
		r.frames = r.frames[:len(r.frames)-1]
	}
}

// recordCode stores exact owned bytes, sharing only immutable prior copies.
func (r *readCallRecorder) recordCode(address common.Address, code []byte) {
	if r.bad || r.entry == nil {
		return
	}
	if r.precompile(address) {
		r.bad = true
		return
	}
	for _, read := range r.entry.code {
		if read.address == address {
			if !bytes.Equal(code, read.code) {
				r.bad = true
			}
			return
		}
	}
	if len(r.entry.code) >= 64 || r.codeBytes+len(code) > 131072 {
		r.bad = true
		return
	}
	var owned []byte
	if r.old != nil {
		for _, read := range r.old.code {
			if read.address == address && bytes.Equal(read.code, code) {
				owned = read.code
				break
			}
		}
	}
	if owned == nil {
		owned = bytes.Clone(code)
	}
	r.entry.code = append(r.entry.code, codeRead{address, owned})
	r.codeBytes += len(code)
}

// captureOpcode forwards the ordinary hook and records the code of each entered
// frame once. Empty frames have no opcode callback and cannot be memoized.
func (r *readCallRecorder) captureOpcode(pc uint64, op byte, gas, cost uint64, scope tracing.OpContext, data []byte, depth int, err error) {
	if r.base.OnOpcode != nil {
		r.base.OnOpcode(pc, op, gas, cost, scope, data, depth, err)
	}
	r.recordOpcode(op, scope, err)
}

// captureOpcodeForScope keeps the base tracer's per-frame hook selection while
// recording every opcode of the frame.
func (r *readCallRecorder) captureOpcodeForScope(frame tracing.OpContext) tracing.OpcodeHook {
	selected := r.base.OnOpcodeForScope(frame)
	return func(pc uint64, op byte, gas, cost uint64, scope tracing.OpContext, data []byte, depth int, err error) {
		if selected != nil {
			selected(pc, op, gas, cost, scope, data, depth, err)
		}
		r.recordOpcode(op, scope, err)
	}
}

// recordOpcode records the frame's code on its first opcode and checks the opcode.
func (r *readCallRecorder) recordOpcode(op byte, scope tracing.OpContext, err error) {
	if len(r.frames) > 0 && !r.frames[len(r.frames)-1] {
		r.frames[len(r.frames)-1] = true
		r.recordCode(scope.Address(), scope.ContractCode())
	}
	r.onOpcode(vm.OpCode(op), scope, err)
}

// onOpcode accepts only operations determined by the key and recorded reads.
// Unknown opcodes, writes, environmental reads and unbounded dependencies fail closed.
func (r *readCallRecorder) onOpcode(op vm.OpCode, scope tracing.OpContext, err error) {
	if r.bad || r.entry == nil {
		return
	}
	if err != nil {
		r.bad = true
		return
	}
	switch {
	case op >= vm.PUSH0 && op <= vm.SWAP16:
		return
	case op >= vm.ADD && op <= vm.SIGNEXTEND, op >= vm.LT && op <= vm.SAR:
		return
	}
	switch op {
	case vm.STOP, vm.KECCAK256, vm.ADDRESS, vm.CALLER, vm.CALLVALUE, vm.CALLDATALOAD, vm.CALLDATASIZE, vm.CALLDATACOPY, vm.CODESIZE, vm.CODECOPY, vm.RETURNDATASIZE, vm.RETURNDATACOPY, vm.POP, vm.MLOAD, vm.MSTORE, vm.MSTORE8, vm.JUMP, vm.JUMPI, vm.PC, vm.MSIZE, vm.GAS, vm.JUMPDEST, vm.MCOPY, vm.STATICCALL, vm.RETURN:
		return
	case vm.EXTCODESIZE:
		stack := scope.StackData()
		if len(stack) < 1 {
			r.bad = true
			return
		}
		address := common.Address(stack[len(stack)-1].Bytes20())
		r.recordCode(address, r.evm.StateDB.GetCode(address))
	case vm.SLOAD:
		stack := scope.StackData()
		if len(stack) < 1 {
			r.bad = true
			return
		}
		slot := common.Hash(stack[len(stack)-1].Bytes32())
		address := scope.Address()
		for _, read := range r.entry.slots {
			if read.address == address && read.slot == slot {
				return
			}
		}
		if len(r.entry.slots) >= 256 {
			r.bad = true
			return
		}
		r.entry.slots = append(r.entry.slots, slotRead{address, slot, r.evm.StateDB.GetState(address, slot)})
	default:
		r.bad = true
	}
}
