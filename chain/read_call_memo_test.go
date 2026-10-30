package chain

import (
	"math/big"
	"testing"

	"github.com/crytic/medusa-geth/common"
	"github.com/crytic/medusa-geth/core"
	gethState "github.com/crytic/medusa-geth/core/state"
	"github.com/crytic/medusa-geth/core/stateless"
	"github.com/crytic/medusa-geth/core/tracing"
	gethTypes "github.com/crytic/medusa-geth/core/types"
	"github.com/crytic/medusa-geth/core/vm"
	"github.com/crytic/medusa-geth/eth/tracers"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

// memoRoot and memoChild are ordinary, non-precompiled test contracts.
var memoRoot = common.HexToAddress("0x1234")
var memoChild = common.HexToAddress("0x1235")

// memoGetter returns slot zero; memoNested returns the child getter result.
var memoGetter = []byte{0x60, 0, 0x54, 0x60, 0, 0x52, 0x60, 32, 0x60, 0, 0xf3}
var memoNested = []byte{0x60, 32, 0x60, 0, 0x60, 0, 0x60, 0, 0x61, 0x12, 0x35, 0x5a, 0xfa, 0x50, 0x60, 32, 0x60, 0, 0xf3}

// memoTestPair creates equivalent chains, with an observer disabling reuse on the reference.
func memoTestPair(t *testing.T, code []byte) (*TestChain, *TestChain, *core.Message) {
	t.Helper()
	a, senders := createChain(t)
	b, _ := createChain(t)
	t.Cleanup(a.Close)
	t.Cleanup(b.Close)
	b.AddTracer(&TestChainTracer{Tracer: &tracers.Tracer{Hooks: &tracing.Hooks{}}}, false, true)
	for _, c := range []*TestChain{a, b} {
		c.state.SetCode(memoRoot, code)
		c.state.SetCode(memoChild, memoGetter)
		c.state.SetState(memoRoot, common.Hash{}, common.Hash{31: 7})
		c.state.SetState(memoChild, common.Hash{}, common.Hash{31: 9})
	}
	msg := &core.Message{From: senders[0], To: &memoRoot, GasLimit: 1000000, Value: new(big.Int), GasPrice: new(big.Int), GasFeeCap: new(big.Int), GasTipCap: new(big.Int)}
	return a, b, msg
}

// compareMemoCall compares results, gas and externally accessible state after each call.
func compareMemoCall(t *testing.T, a, b *TestChain, msg *core.Message) *core.ExecutionResult {
	t.Helper()
	got, err := a.CallContract(msg, nil)
	require.NoError(t, err)
	want, err := b.CallContract(msg, nil)
	require.NoError(t, err)
	require.Equal(t, want, got)
	// Validate state and snapshot-ID effects as well as the return value/gas.
	for _, addr := range []common.Address{memoRoot, memoChild, msg.From, a.Head().Header.Coinbase} {
		require.Equal(t, b.state.GetBalance(addr), a.state.GetBalance(addr))
		require.Equal(t, b.state.GetNonce(addr), a.state.GetNonce(addr))
		require.Equal(t, b.state.GetState(addr, common.Hash{}), a.state.GetState(addr, common.Hash{}))
		require.Equal(t, b.state.AddressInAccessList(addr), a.state.AddressInAccessList(addr))
		bw, bs := b.state.SlotInAccessList(addr, common.Hash{})
		aw, as := a.state.SlotInAccessList(addr, common.Hash{})
		require.Equal(t, bw, aw)
		require.Equal(t, bs, as)
	}
	ai, bi := a.state.Snapshot(), b.state.Snapshot()
	require.Equal(t, bi, ai)
	a.state.RevertToSnapshot(ai)
	b.state.RevertToSnapshot(bi)
	return got
}

// TestReadCallMemoDependencies changes read and unread dependencies at both call depths.
func TestReadCallMemoDependencies(t *testing.T) {
	for _, nested := range []bool{false, true} {
		t.Run(map[bool]string{false: "root", true: "nested"}[nested], func(t *testing.T) {
			code := memoGetter
			target := memoRoot
			if nested {
				code = memoNested
				target = memoChild
			}
			a, b, msg := memoTestPair(t, code)
			compareMemoCall(t, a, b, msg)
			require.NotEmpty(t, a.callMemo.entries)
			compareMemoCall(t, a, b, msg)
			for _, c := range []*TestChain{a, b} {
				c.state.SetState(target, common.Hash{31: 1}, common.Hash{31: 12})
			}
			compareMemoCall(t, a, b, msg)
			for _, c := range []*TestChain{a, b} {
				c.state.SetState(target, common.Hash{}, common.Hash{31: 15})
			}
			result := compareMemoCall(t, a, b, msg)
			require.EqualValues(t, 15, result.ReturnData[31])
			result.ReturnData[31] = 99
			result = compareMemoCall(t, a, b, msg)
			require.EqualValues(t, 15, result.ReturnData[31])
			// Changing gas or code must not reuse an old result.
			msg.GasLimit += 1000
			compareMemoCall(t, a, b, msg)
			for _, c := range []*TestChain{a, b} {
				c.state.SetCode(target, []byte{0x60, 11, 0x60, 0, 0x52, 0x60, 32, 0x60, 0, 0xf3})
			}
			result = compareMemoCall(t, a, b, msg)
			require.EqualValues(t, 11, result.ReturnData[31])
		})
	}
}

// TestReadCallMemoFallsBack ensures unsupported or unsuccessful executions are not retained.
func TestReadCallMemoFallsBack(t *testing.T) {
	cases := map[string][]byte{
		"empty code":      nil,
		"timestamp":       {0x42, 0x60, 0, 0x52, 0x60, 32, 0x60, 0, 0xf3},
		"balance":         {0x47, 0x60, 0, 0x52, 0x60, 32, 0x60, 0, 0xf3},
		"writes":          {0x60, 1, 0x60, 0, 0x55, 0},
		"revert":          {0x60, 0, 0x60, 0, 0xfd},
		"invalid":         {0xfe},
		"chain ID":        {0x46, 0x60, 0, 0x52, 0x60, 32, 0x60, 0, 0xf3},
		"transient read":  {0x60, 0, 0x5c, 0x60, 0, 0x52, 0x60, 32, 0x60, 0, 0xf3},
		"transient write": {0x60, 1, 0x60, 0, 0x5d, 0},
		"log":             {0x60, 0, 0x60, 0, 0xa0, 0},
	}
	for name, code := range cases {
		t.Run(name, func(t *testing.T) {
			a, b, msg := memoTestPair(t, code)
			compareMemoCall(t, a, b, msg)
			for _, c := range []*TestChain{a, b} {
				c.Head().Header.Time += 100
				c.state.SetBalance(memoRoot, uint256.NewInt(123), tracing.BalanceChangeUnspecified)
			}
			compareMemoCall(t, a, b, msg)
			require.Empty(t, a.callMemo.entries)
		})
	}
}

// TestReadCallMemoObservers verifies that permanent and per-call observers see every opcode.
func TestReadCallMemoObservers(t *testing.T) {
	a, b, msg := memoTestPair(t, memoGetter)
	compareMemoCall(t, a, b, msg)
	compareMemoCall(t, a, b, msg)
	opcodes := 0
	tracer := &TestChainTracer{Tracer: &tracers.Tracer{Hooks: &tracing.Hooks{OnOpcode: func(uint64, byte, uint64, uint64, tracing.OpContext, []byte, int, error) { opcodes++ }}}}
	_, err := a.CallContract(msg, nil, tracer)
	require.NoError(t, err)
	require.Positive(t, opcodes)
	opcodes = 0
	a.AddTracer(tracer, false, true)
	_, err = a.CallContract(msg, nil)
	require.NoError(t, err)
	require.Positive(t, opcodes)
}

// memoTestPrecompile counts executions so tests can prove that precompiles are never skipped.
type memoTestPrecompile struct{ calls int }

func (*memoTestPrecompile) RequiredGas([]byte) uint64 { return 13 }
func (p *memoTestPrecompile) Run([]byte) ([]byte, error) {
	p.calls++
	out := make([]byte, 32)
	out[31] = 27
	return out, nil
}

// TestReadCallMemoNewPrecompile invalidates an entry when an extension replaces child code.
func TestReadCallMemoNewPrecompile(t *testing.T) {
	a, b, msg := memoTestPair(t, memoNested)
	compareMemoCall(t, a, b, msg)
	compareMemoCall(t, a, b, msg)
	pa, pb := &memoTestPrecompile{}, &memoTestPrecompile{}
	a.vmConfigExtensions.AdditionalPrecompiles[memoChild] = pa
	b.vmConfigExtensions.AdditionalPrecompiles[memoChild] = pb
	for range 2 {
		result := compareMemoCall(t, a, b, msg)
		require.EqualValues(t, 27, result.ReturnData[31])
	}
	require.Equal(t, 2, pa.calls)
	require.Equal(t, 2, pb.calls)
}

// TestReadCallMemoNativePrecompileTable checks changes to the exported native dispatch table.
func TestReadCallMemoNativePrecompileTable(t *testing.T) {
	a, b, msg := memoTestPair(t, memoNested)
	compareMemoCall(t, a, b, msg)
	compareMemoCall(t, a, b, msg)
	// The exported native table can also be extended; ActivePrecompiles is a
	// separate list used for gas warming and does not reflect such additions.
	_, present := vm.PrecompiledContractsPrague[memoChild]
	require.False(t, present)
	p := &memoTestPrecompile{}
	vm.PrecompiledContractsPrague[memoChild] = p
	t.Cleanup(func() { delete(vm.PrecompiledContractsPrague, memoChild) })
	for range 2 {
		result := compareMemoCall(t, a, b, msg)
		require.EqualValues(t, 27, result.ReturnData[31])
	}
	require.Equal(t, 4, p.calls)
}

// TestReadCallMemoPrecompileWarmingList checks gas changes from the separate warming list.
func TestReadCallMemoPrecompileWarmingList(t *testing.T) {
	a, b, msg := memoTestPair(t, memoNested)
	compareMemoCall(t, a, b, msg)
	cold := compareMemoCall(t, a, b, msg)
	old := vm.PrecompiledAddressesPrague
	vm.PrecompiledAddressesPrague = append(append([]common.Address(nil), old...), memoChild)
	t.Cleanup(func() { vm.PrecompiledAddressesPrague = old })
	warm := compareMemoCall(t, a, b, msg)
	require.Less(t, warm.UsedGas, cold.UsedGas)
	compareMemoCall(t, a, b, msg)
}

// TestReadCallMemoWarmnessAndInput varies warmth, input, value and available gas.
func TestReadCallMemoWarmnessAndInput(t *testing.T) {
	a, b, msg := memoTestPair(t, memoNested)
	compareMemoCall(t, a, b, msg)
	cold := compareMemoCall(t, a, b, msg)
	// Coinbase is warm after Shanghai. Changing it to the child changes gas,
	// even though the getter never executes COINBASE.
	for _, c := range []*TestChain{a, b} {
		c.Head().Header.Coinbase = memoChild
	}
	warm := compareMemoCall(t, a, b, msg)
	require.Less(t, warm.UsedGas, cold.UsedGas)
	compareMemoCall(t, a, b, msg)
	msg.AccessList = gethTypes.AccessList{{Address: memoChild, StorageKeys: []common.Hash{{}}}}
	compareMemoCall(t, a, b, msg)
	msg.AccessList = nil
	msg.Data = []byte{1, 2, 3}
	compareMemoCall(t, a, b, msg)
	msg.Value.SetUint64(1)
	compareMemoCall(t, a, b, msg)
	msg.GasLimit = 22000
	compareMemoCall(t, a, b, msg)
}

// TestReadCallMemoExternalCodeAndChildFailures covers EXTCODESIZE and non-memoizable children.
func TestReadCallMemoExternalCodeAndChildFailures(t *testing.T) {
	t.Run("external code", func(t *testing.T) {
		code := []byte{0x61, 0x12, 0x35, 0x3b, 0x60, 0, 0x52, 0x60, 32, 0x60, 0, 0xf3}
		a, b, msg := memoTestPair(t, code)
		compareMemoCall(t, a, b, msg)
		compareMemoCall(t, a, b, msg)
		for _, c := range []*TestChain{a, b} {
			c.state.SetCode(memoChild, nil)
		}
		result := compareMemoCall(t, a, b, msg)
		require.Equal(t, make([]byte, 32), result.ReturnData)
	})
	for name, code := range map[string][]byte{"empty": nil, "revert": {0x60, 0, 0x60, 0, 0xfd}, "invalid": {0xfe}} {
		t.Run(name, func(t *testing.T) {
			a, b, msg := memoTestPair(t, memoNested)
			for _, c := range []*TestChain{a, b} {
				c.state.SetCode(memoChild, code)
			}
			for range 2 {
				compareMemoCall(t, a, b, msg)
			}
			require.Empty(t, a.callMemo.entries)
		})
	}
}

// TestReadCallMemoLookup exercises the actual hit flag without production-only
// counters. It also checks that storage, code, context and gas changes miss.
func TestReadCallMemoLookup(t *testing.T) {
	a, b, msg := memoTestPair(t, memoNested)
	want := compareMemoCall(t, a, b, msg)
	require.Len(t, a.callMemo.entries, 1)
	var key readCallKey
	var entry *readCallEntry
	for key, entry = range a.callMemo.entries {
	}
	lookup := func(key readCallKey) bool {
		snapshot := a.state.Snapshot()
		defer a.state.RevertToSnapshot(snapshot)
		context := newTestChainBlockContext(a, a.Head().Header)
		context.Coinbase = key.coinbase
		evm := vm.NewEVM(context, a.state, a.chainConfig, vm.Config{ConfigExtensions: a.vmConfigExtensions})
		contract := vm.NewContract(key.caller, key.address, &key.value, key.gas, nil)
		recorder := newReadCallRecorder(a, a.callMemo, &tracing.Hooks{})
		output, data, gas, hit := recorder.Lookup(evm, contract, []byte(key.input), key.readOnly, key.rules)
		if hit {
			require.Equal(t, want.ReturnData, output)
			require.Equal(t, entry.returnData, data)
			require.Equal(t, entry.gas, gas)
			// Every returned buffer belongs to this lookup, never to the entry.
			if len(output) != 0 {
				output[0] ^= 0xff
				require.NotEqual(t, output, entry.output)
			}
			if len(data) != 0 {
				data[0] ^= 0xff
				require.NotEqual(t, data, entry.returnData)
			}
		}
		return hit
	}
	require.True(t, lookup(key))
	require.True(t, lookup(key))
	for name, mutate := range map[string]func(*readCallKey){
		"caller":    func(k *readCallKey) { k.caller[0]++ },
		"recipient": func(k *readCallKey) { k.address[0]++ },
		"coinbase":  func(k *readCallKey) { k.coinbase[0]++ },
		"value":     func(k *readCallKey) { k.value.SetUint64(1) },
		"gas":       func(k *readCallKey) { k.gas-- },
		"input":     func(k *readCallKey) { k.input = "changed" },
		"read only": func(k *readCallKey) { k.readOnly = !k.readOnly },
		"fork":      func(k *readCallKey) { k.rules.IsIstanbul = !k.rules.IsIstanbul },
	} {
		t.Run(name, func(t *testing.T) {
			changed := key
			mutate(&changed)
			require.False(t, lookup(changed))
		})
	}
	a.state.SetState(memoChild, common.Hash{31: 1}, common.Hash{31: 99})
	require.True(t, lookup(key), "unread storage is not a dependency")
	a.state.SetState(memoChild, common.Hash{}, common.Hash{31: 10})
	require.False(t, lookup(key))
	a.state.SetState(memoChild, common.Hash{}, common.Hash{31: 9})
	require.True(t, lookup(key))
	a.state.SetCode(memoChild, []byte{0})
	require.False(t, lookup(key))
}

// TestReadCallMemoDelegation changes an ordinary root into a delegated account
// and back, including a changed delegation target. Gas and result must agree.
func TestReadCallMemoDelegation(t *testing.T) {
	a, b, msg := memoTestPair(t, memoGetter)
	compareMemoCall(t, a, b, msg)
	compareMemoCall(t, a, b, msg)
	for _, c := range []*TestChain{a, b} {
		c.state.SetCode(memoRoot, gethTypes.AddressToDelegation(memoChild))
	}
	for range 2 {
		compareMemoCall(t, a, b, msg)
	}
	for _, c := range []*TestChain{a, b} {
		c.state.SetCode(memoChild, []byte{0x60, 11, 0x60, 0, 0x52, 0x60, 32, 0x60, 0, 0xf3})
	}
	for range 2 {
		result := compareMemoCall(t, a, b, msg)
		require.EqualValues(t, 11, result.ReturnData[31])
	}
	for _, c := range []*TestChain{a, b} {
		c.state.SetCode(memoRoot, memoGetter)
	}
	compareMemoCall(t, a, b, msg)
	compareMemoCall(t, a, b, msg)
}

// TestReadCallMemoExplicitState keeps caller-provided state on the ordinary path,
// including the chain's own state when it is supplied explicitly.
func TestReadCallMemoExplicitState(t *testing.T) {
	a, b, msg := memoTestPair(t, memoGetter)
	want, err := b.CallContract(msg, nil)
	require.NoError(t, err)
	for range 2 {
		got, err := a.CallContract(msg, a.state)
		require.NoError(t, err)
		require.Equal(t, want, got)
		require.Nil(t, a.callMemo)
	}
}

// TestReadCallMemoEmptyReturn preserves the distinction between nil and empty
// return buffers and checks repeated reuse of successful zero-output frames.
func TestReadCallMemoEmptyReturn(t *testing.T) {
	for name, code := range map[string][]byte{"stop": {0x00}, "return": {0x60, 0, 0x60, 0, 0xf3}} {
		t.Run(name, func(t *testing.T) {
			a, b, msg := memoTestPair(t, code)
			for range 3 {
				compareMemoCall(t, a, b, msg)
			}
			require.Len(t, a.callMemo.entries, 1)
		})
	}
}

// TestReadCallMemoLateObserver invalidates a previously cached child when it is
// replaced by an extension that registers an observer during execution.
func TestReadCallMemoLateObserver(t *testing.T) {
	a, _, msg := memoTestPair(t, memoNested)
	for range 2 {
		_, err := a.CallContract(msg, nil)
		require.NoError(t, err)
	}
	require.NotEmpty(t, a.callMemo.entries)
	var receipt *gethTypes.Receipt
	opcodes := 0
	observer := &TestChainTracer{Tracer: &tracers.Tracer{Hooks: &tracing.Hooks{
		OnOpcode: func(uint64, byte, uint64, uint64, tracing.OpContext, []byte, int, error) { opcodes++ },
		OnTxEnd: func(r *gethTypes.Receipt, err error) {
			require.NoError(t, err)
			receipt = r
		},
	}}}
	a.vmConfigExtensions.AdditionalPrecompiles[memoChild] = &memoObserverPrecompile{run: func() {
		a.AddTracer(observer, false, true)
	}}
	result, err := a.CallContract(msg, nil)
	require.NoError(t, err)
	require.Positive(t, opcodes)
	require.NotNil(t, receipt)
	require.Equal(t, result.UsedGas, receipt.GasUsed)
	require.Equal(t, make([]byte, 32), result.ReturnData)
}

// TestReadCallMemoWitness keeps explicit witness collection on the ordinary
// path, even when the active fork itself does not require witness accounting.
func TestReadCallMemoWitness(t *testing.T) {
	a, b, msg := memoTestPair(t, memoNested)
	witnesses := make([]*stateless.Witness, 0, 2)
	for _, chain := range []*TestChain{a, b} {
		witness, err := stateless.NewWitness(chain.Head().Header, nil)
		require.NoError(t, err)
		native := chain.state.(*gethState.StateDB)
		native.StartPrefetcher("read-call-memo-test", witness)
		t.Cleanup(native.StopPrefetcher)
		witnesses = append(witnesses, witness)
	}
	for range 2 {
		compareMemoCall(t, a, b, msg)
		require.Nil(t, a.callMemo)
	}
	require.Equal(t, witnesses[1].Codes, witnesses[0].Codes)
	require.Contains(t, witnesses[0].Codes, string(memoNested))
	require.Contains(t, witnesses[0].Codes, string(memoGetter))
}

// memoObserverPrecompile invokes an extension callback while a call is executing.
type memoObserverPrecompile struct{ run func() }

func (*memoObserverPrecompile) RequiredGas([]byte) uint64 { return 13 }
func (p *memoObserverPrecompile) Run([]byte) ([]byte, error) {
	p.run()
	return make([]byte, 32), nil
}

// TestReadCallMemoNilMessage preserves the existing invalid-input panic and the
// snapshot revision allocated before it, rather than failing during eligibility.
func TestReadCallMemoNilMessage(t *testing.T) {
	chain, _ := createChain(t)
	t.Cleanup(chain.Close)
	before := chain.state.Snapshot()
	chain.state.RevertToSnapshot(before)
	require.Panics(t, func() { _, _ = chain.CallContract(nil, nil) })
	after := chain.state.Snapshot()
	require.Equal(t, before+2, after)
	chain.state.RevertToSnapshot(after)
}
