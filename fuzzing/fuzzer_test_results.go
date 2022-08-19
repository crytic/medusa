package fuzzing

import (
	"encoding/json"
	"fmt"
	coreTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/trailofbits/medusa/compilation/types"
	fuzzingTypes "github.com/trailofbits/medusa/types"
	"strings"
	"sync"
)

// FuzzerResults describes results from a Fuzzer execution.
type FuzzerResults struct {
	// failedTests describes a list of property tests which were violated by the fuzzer.
	failedTests []FuzzerResultFailedTest
	// failedTestsLock provides thread-synchronization when accessing failedTests, to prevent a race condition.
	failedTestsLock sync.Mutex
}

// NewFuzzerResults returns a new FuzzerResults struct to track results of a Fuzzer run.
func NewFuzzerResults() *FuzzerResults {
	results := &FuzzerResults{
		failedTests: make([]FuzzerResultFailedTest, 0),
	}
	return results
}

// GetFailedTests returns information about any failed property tests we encountered.
func (r *FuzzerResults) GetFailedTests() []FuzzerResultFailedTest {
	return r.failedTests
}

// addFailedTest adds a new FuzzerResultFailedTest to our list of failed property tests.
func (r *FuzzerResults) addFailedTest(result *FuzzerResultFailedTest) {
	// Add our fuzzer result
	r.failedTestsLock.Lock()
	r.failedTests = append(r.failedTests, *result)
	r.failedTestsLock.Unlock()
}

// FuzzerResultFailedTest describes failed tests generated by a transaction sequence in a Fuzzer run.
type FuzzerResultFailedTest struct {
	// TxSequence represents the transaction sequence used to trigger the failed tests.
	TxSequence []FuzzerResultFailedTestTx
	// FailedTests represents the property tests which were violated after applying the TxSequence.
	FailedTests []fuzzingTypes.DeployedMethod
}

// NewFuzzerResultFailedTest returns a new FuzzerResultFailedTest struct which describes a property test which failed
// during Fuzzer execution.
func NewFuzzerResultFailedTest(txSequence []FuzzerResultFailedTestTx, failedTests []fuzzingTypes.DeployedMethod) *FuzzerResultFailedTest {
	result := &FuzzerResultFailedTest{
		TxSequence:  txSequence,
		FailedTests: failedTests,
	}
	return result
}

// String provides a string representation of a failed test.
func (ft *FuzzerResultFailedTest) String() string {
	// Construct a message with all property test names.
	violatedNames := make([]string, len(ft.FailedTests))
	for i := 0; i < len(ft.FailedTests); i++ {
		violatedNames[i] = ft.FailedTests[i].Method.Sig
	}

	// Next we'll want the tx call information.
	txMethodNames := make([]string, len(ft.TxSequence))
	for i := 0; i < len(txMethodNames); i++ {
		// Obtain our tx and decode our method from this.
		failedTestTx := ft.TxSequence[i]
		method, err := failedTestTx.Contract.Abi.MethodById(failedTestTx.Tx.Data)
		if err != nil || method == nil {
			panic("failed to evaluate failed test method from transaction data")
		}

		// Next decode our arguments
		args, err := method.Inputs.Unpack(failedTestTx.Tx.Data[4:])
		if err != nil {
			panic("failed to unpack method args from transaction data")
		}

		// Serialize our args to a JSON string and set it as our tx method name for this index.
		// TODO: Byte arrays are encoded as base64 strings, so this should be represented another way in the future:
		//  Reference: https://stackoverflow.com/questions/14177862/how-to-marshal-a-byte-uint8-array-as-json-array-in-go
		b, err := json.Marshal(args)
		if err != nil {
			b = []byte("<error resolving args>")
		}

		// Obtain our sender for this transaction
		var senderStr string
		sender, err := coreTypes.Sender(coreTypes.HomesteadSigner{}, coreTypes.NewTx(failedTestTx.Tx))
		if err == nil {
			senderStr = sender.String()
		} else {
			senderStr = "<unresolved>"
		}

		txMethodNames[i] = fmt.Sprintf(
			"[%d] %s(%s) (sender=%s, gas=%d, gasprice=%s, value=%s)",
			i+1,
			method.Name,
			string(b),
			senderStr,
			failedTestTx.Tx.Gas,
			failedTestTx.Tx.GasPrice.String(),
			failedTestTx.Tx.Value.String(),
		)
	}

	// Create our final message and return it.
	msg := fmt.Sprintf(
		"Failed property tests: %s\nTransaction Sequence:\n%s",
		strings.Join(violatedNames, ", "),
		strings.Join(txMethodNames, "\n"),
	)
	return msg
}

// FuzzerResultFailedTestTx describes a single transaction in a transaction sequence causing failed property tests in a
// Fuzzer run.
type FuzzerResultFailedTestTx struct {
	// Contract describes the contract which was targeted by a transaction.
	Contract *types.CompiledContract

	// Tx represents the underlying transaction.
	Tx *coreTypes.LegacyTx
}

// NewFuzzerResultFailedTestTx returns a new FuzzerResultFailedTestTx struct to track a tx in a tx sequence leading to
// a failed test.
func NewFuzzerResultFailedTestTx(contract *types.CompiledContract, tx *coreTypes.LegacyTx) *FuzzerResultFailedTestTx {
	failedTx := &FuzzerResultFailedTestTx{
		Contract: contract,
		Tx:       tx,
	}
	return failedTx
}
