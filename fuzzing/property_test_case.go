package fuzzing

import (
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi"
	fuzzerTypes "github.com/trailofbits/medusa/fuzzing/types"
	"strings"
)

// PropertyTestCase describes a test being run by a PropertyTestCaseProvider.
type PropertyTestCase struct {
	status         string
	targetContract *fuzzerTypes.Contract
	targetMethod   abi.Method
	callSequence   fuzzerTypes.CallSequence
}

// Status describes the TestCaseStatus enum option used to define the current state of the test.
func (t *PropertyTestCase) Status() string {
	return t.status
}

// CallSequence describes the types.CallSequence of calls sent to the EVM which resulted in this TestCase.
func (t *PropertyTestCase) CallSequence() fuzzerTypes.CallSequence {
	return t.callSequence
}

// Name describes the name of the test case.
func (t *PropertyTestCase) Name() string {
	return fmt.Sprintf("Property Test: %s.%s", t.targetContract.Name(), t.targetMethod.Sig)
}

// Message obtains a text-based printable message which describes the test result.
func (t *PropertyTestCase) Message() string {
	// If the test did not fail, we have no message.
	if t.Status() != TestCaseStatusFailed {
		return ""
	}

	// Construct an array of strings specifying each step of our call sequence.
	txMethodNames := make([]string, len(t.CallSequence()))
	for i := 0; i < len(txMethodNames); i++ {
		// Obtain our tx and decode our method from this.
		callSequenceElement := t.CallSequence()[i]
		method, err := callSequenceElement.Method()
		if err != nil || method == nil {
			panic("failed to evaluate failed test method from call sequence data")
		}

		// Next decode our arguments (we jump four bytes to skip the function selector)
		args, err := method.Inputs.Unpack(callSequenceElement.Call().Data()[4:])
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
		senderStr := callSequenceElement.Call().From()

		txMethodNames[i] = fmt.Sprintf(
			"[%d] %s(%s) (sender=%s, gas=%d, gasprice=%s, value=%s)",
			i+1,
			method.Name,
			string(b),
			senderStr,
			callSequenceElement.Call().Gas(),
			callSequenceElement.Call().GasPrice().String(),
			callSequenceElement.Call().Value().String(),
		)
	}

	// Create our final message and return it.
	message := fmt.Sprintf(
		"%s.%s failed after the following call sequence:\n%s",
		t.targetContract.Name(),
		t.targetMethod.Sig,
		strings.Join(txMethodNames, "\n"),
	)
	return message
}

// ID obtains a unique identifier for a test result. If the same test fails, this ID should match for both
// PropertyTestResult instances (even if the CallSequence differs or has not been shrunk).
func (t *PropertyTestCase) ID() string {
	return strings.Replace(fmt.Sprintf("PROPTEST-%s-%s", t.targetContract.Name(), t.targetMethod.Sig), "_", "-", -1)
}
