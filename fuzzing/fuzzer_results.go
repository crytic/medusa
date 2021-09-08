package fuzzing

import (
	"encoding/json"
	"fmt"
	coreTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/trailofbits/medusa/compilation/types"
	"strings"
	"sync"
)

type FuzzerResults struct {
	failedTests []FuzzerResultFailedTest
	failedTestsLock sync.Mutex
}

type FuzzerResultFailedTest struct {
	TxSequence []FuzzerResultFailedTestTx
	FailedTests []deployedMethod
}

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
		Tx: tx,
	}
	return failedTx
}


// NewFuzzerResults returns a new FuzzerResults struct to track results of a Fuzzer run.
func NewFuzzerResults() *FuzzerResults {
	results := &FuzzerResults{
		failedTests: make([]FuzzerResultFailedTest, 0),
	}
	return results
}

// NewFuzzerResultFailedTest returns a new FuzzerResultFailedTest struct which describes a property test which failed
// during Fuzzer execution.
func NewFuzzerResultFailedTest(txSequence []FuzzerResultFailedTestTx, failedTests []deployedMethod) *FuzzerResultFailedTest {
	result := &FuzzerResultFailedTest{
		TxSequence: txSequence,
		FailedTests: failedTests,
	}
	return result
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

// String provides a string representation of a failed test.
func (ft *FuzzerResultFailedTest) String() string {
	// Construct a message with all property test names.
	violatedNames := make([]string, len(ft.FailedTests))
	for i := 0; i < len(ft.FailedTests); i++ {
		violatedNames[i] = ft.FailedTests[i].method.Sig
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
		b, err := json.Marshal(args)
		if err != nil {
			b = []byte("<error resolving args>")
		}
		txMethodNames[i] = fmt.Sprintf("[%d] %s(%s)", i + 1, method.Name, string(b))
	}

	// Create our final message and return it.
	msg := fmt.Sprintf(
		"Failed property tests: %s\nTransaction Sequence:\n%s",
		strings.Join(violatedNames, ", "),
		strings.Join(txMethodNames, "\n"),
	)
	return msg
}
