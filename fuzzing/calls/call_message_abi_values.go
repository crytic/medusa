package calls

import (
	"encoding/json"
	"fmt"
	"github.com/crytic/medusa/fuzzing/valuegeneration"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

// CallMessageDataAbiValues describes a CallMessage Data field which is represented by ABI input argument values.
// This is represented at runtime by an abi.Method and its input values.
// Note: The data may be serialized. When deserializing, the Resolve method must be called to resolve the abi.Method
// and transform the encoded input data into compatible input values for the method.
type CallMessageDataAbiValues struct {
	// Method defines the ABI method definition used to pack input argument values.
	Method *abi.Method

	// InputValues represents the ABI packable input argument values to use alongside the Method to produce the call
	// data.
	InputValues []any

	// methodName stores the name of Method when decoding from JSON. The Method will be resolved using this internal
	// reference when Resolve is called.
	methodName string

	// encodedInputValues stores the raw encoded input values when decoding from JSON. The actual InputValues will be
	// decoded using this and the resolved Method once Resolve is called.
	encodedInputValues []any
}

// callMessageDataAbiValuesMarshal is used as an internal struct to represent JSON serialized data for
// CallMessageDataAbiValues.
type callMessageDataAbiValuesMarshal struct {
	MethodName         string `json:"methodName"`
	EncodedInputValues []any  `json:"inputValues"`
}

// Clone creates a copy of the given message data and its underlying components, or an error if one occurs.
func (m *CallMessageDataAbiValues) Clone() (*CallMessageDataAbiValues, error) {
	// Create a cloned struct
	clone := &CallMessageDataAbiValues{
		Method:             m.Method,
		InputValues:        nil, // set lower
		methodName:         m.methodName,
		encodedInputValues: m.encodedInputValues,
	}

	// If we have a method, clone our input values by packing/unpacking them.
	if m.Method != nil {
		data, err := m.Method.Inputs.Pack(m.InputValues...)
		if err != nil {
			return nil, err
		}

		clone.InputValues, err = m.Method.Inputs.Unpack(data)
		if err != nil {
			return nil, err
		}
	}

	return clone, nil
}

// Resolve takes a previously unmarshalled CallMessageDataAbiValues and resolves all internal data needed for it to be
// used at runtime by resolving the abi.Method it references from the provided contract ABI.
func (d *CallMessageDataAbiValues) Resolve(contractAbi abi.ABI) error {
	// Try to resolve the method from our contract ABI.
	if resolvedMethod, ok := contractAbi.Methods[d.methodName]; ok {
		d.Method = &resolvedMethod
	} else {
		return fmt.Errorf("could not resolve method '%v' from the given contract ABI", d.methodName)
	}

	// Now that we've resolved the method, decode our encoded input values.
	decodedArguments, err := valuegeneration.DecodeJSONArgumentsFromSlice(d.Method.Inputs, d.encodedInputValues, make(map[string]common.Address))
	if err != nil {
		return err
	}

	// If we've decoded arguments successfully, set them and clear our encoded arguments as they're no longer needed.
	d.InputValues = decodedArguments
	d.encodedInputValues = nil
	return nil
}

// Pack packs all the ABI argument InputValues into call data for the relevant Method it targets. If this was
// deserialized, Resolve must be called first to resolve necessary runtime data (such as the Method).
func (d *CallMessageDataAbiValues) Pack() ([]byte, error) {
	// We must have set an ABI method at runtime to serialize this.
	if d.Method == nil {
		return nil, fmt.Errorf("ABI call data packing failed, method definition was not set at runtime")
	}

	// If our ABI method was not set, we can't serialize our data.
	// If our method has a different amount of inputs than we have values, return an error.
	if len(d.Method.Inputs) != len(d.InputValues) {
		return nil, fmt.Errorf("ABI call data packing failed, method definition describes %d input arguments, but %d were provided", len(d.Method.Inputs), len(d.InputValues))
	}

	// Pack the input values
	argData, err := d.Method.Inputs.Pack(d.InputValues...)
	if err != nil {
		return nil, fmt.Errorf("ABI call data packing encountered error: %v", err)
	}

	// Prepend the method ID to the data and return it.
	callData := append(append([]byte{}, d.Method.ID...), argData...)
	return callData, nil
}

// MarshalJSON provides custom JSON marshalling for the struct.
// Returns the JSON marshalled data, or an error if one occurs.
func (d *CallMessageDataAbiValues) MarshalJSON() ([]byte, error) {
	// We must have set an ABI method at runtime to serialize this.
	if d.Method == nil {
		return nil, fmt.Errorf("ABI call data JSON marshaling failed, method definition was not set at runtime")
	}

	// If our ABI method was not set, we can't serialize our data.
	// If our method has a different amount of inputs than we have values, return an error.
	if len(d.Method.Inputs) != len(d.InputValues) {
		return nil, fmt.Errorf("ABI call data JSON marshaling failed, method definition describes %d input arguments, but %d were provided", len(d.Method.Inputs), len(d.InputValues))
	}

	// For every input we have, we serialize it.
	inputValuesEncoded, err := valuegeneration.EncodeJSONArgumentsToSlice(d.Method.Inputs, d.InputValues)
	if err != nil {
		return nil, err
	}

	// Now create our outer struct and marshal all the data and return it.
	marshalData := callMessageDataAbiValuesMarshal{
		MethodName:         d.Method.Name,
		EncodedInputValues: inputValuesEncoded,
	}
	return json.Marshal(marshalData)
}

// UnmarshalJSON provides custom JSON unmarshalling for the struct.
// Returns an error if one occurs.
func (d *CallMessageDataAbiValues) UnmarshalJSON(b []byte) error {
	// Decode our intermediate structure
	var marshalData callMessageDataAbiValuesMarshal
	err := json.Unmarshal(b, &marshalData)
	if err != nil {
		return err
	}

	// Set our data in our actual structure now
	d.methodName = marshalData.MethodName
	d.encodedInputValues = marshalData.EncodedInputValues
	return nil
}
