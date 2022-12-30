package types

import (
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	coreTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/trailofbits/medusa/chain"
	"github.com/trailofbits/medusa/fuzzing/valuegeneration"
	"math/big"
)

// The following directives will be picked up by the `go generate` command to generate JSON marshaling code from
// templates defined below. They should be preserved for re-use in case we change our structures.
//go:generate go get github.com/fjl/gencodec
//go:generate go run github.com/fjl/gencodec -type CallMessage -field-override callMessageMarshaling -out gen_call_message_json.go

// CallMessage implements Ethereum's coreTypes.Message, used to apply EVM/state updates.
type CallMessage struct {
	// MsgFrom represents a core.Message's from parameter (sender), indicating who sent a transaction/message to the
	// Ethereum core to apply a state update.
	MsgFrom common.Address `json:"from"`

	// MsgTo represents the receiving address for a given core.Message.
	MsgTo *common.Address `json:"to"`

	// MsgNonce represents the core.Message sender's nonce
	MsgNonce uint64 `json:"nonce"`

	// MsgValue represents ETH value to be sent to the receiver of the message.
	MsgValue *big.Int `json:"value"`

	// MsgGas represents the maximum amount of gas the sender is willing to spend to cover the cost of executing the
	// message or transaction.
	MsgGas uint64 `json:"gas"`

	// MsgGasPrice represents the price which the sender is willing to pay for each unit of gas used during execution
	// of the message.
	MsgGasPrice *big.Int `json:"gasPrice"`

	// MsgGasFeeCap represents the maximum fee to enforce for gas related costs (related to 1559 transaction executed).
	// The use of nil here indicates that the gas price oracle should be relied on instead.
	MsgGasFeeCap *big.Int `json:"gasFeeCap"`

	// MsgGasTipCap represents the fee cap to use for 1559 transaction. The use of nil here indicates that the gas price
	// oracle should be relied on instead.
	MsgGasTipCap *big.Int `json:"gasTipCap"`

	// MsgData represents the underlying message data to be sent to the receiver. If the receiver is a smart contract,
	// this will likely house your call parameters and other serialized data. If MsgDataAbiValues is non-nil, this
	// value is not used.
	MsgData []byte `json:"data,omitempty"`

	// MsgData represents the underlying message data to be sent to the receiver. If the receiver is a smart contract,
	// this will likely house your call parameters and other serialized data. This overrides MsgData if it is set,
	// allowing Data to be sourced from method ABI input arguments instead.
	MsgDataAbiValues *CallMessageDataAbiValues `json:"data_abi_values,omitempty"`
}

// callMessageMarshaling is a structure that overrides field types during JSON marshaling. It allows CallMessage to
// have its custom marshaling methods auto-generated and will handle type conversions for serialization purposes.
// For example, this enables serialization of big.Int but specifying a different field type to control serialization.
type callMessageMarshaling struct {
	MsgValue     *hexutil.Big
	MsgGasPrice  *hexutil.Big
	MsgGasFeeCap *hexutil.Big
	MsgGasTipCap *hexutil.Big
	MsgData      hexutil.Bytes
}

// NewCallMessage instantiates a new call message from a given set of parameters, with call data set from bytes.
func NewCallMessage(from common.Address, to *common.Address, nonce uint64, value *big.Int, gasLimit uint64, gasPrice, gasFeeCap, gasTipCap *big.Int, data []byte) *CallMessage {
	// Construct and return a new message from our given parameters.
	return &CallMessage{
		MsgFrom:          from,
		MsgTo:            to,
		MsgNonce:         nonce,
		MsgValue:         value,
		MsgGas:           gasLimit,
		MsgGasPrice:      gasPrice,
		MsgGasFeeCap:     gasFeeCap,
		MsgGasTipCap:     gasTipCap,
		MsgData:          data,
		MsgDataAbiValues: nil,
	}
}

// NewCallMessageWithAbiValueData instantiates a new call message from a given set of parameters, with call data set
// from method ABI specified inputs.
func NewCallMessageWithAbiValueData(from common.Address, to *common.Address, nonce uint64, value *big.Int, gasLimit uint64, gasPrice, gasFeeCap, gasTipCap *big.Int, data *CallMessageDataAbiValues) *CallMessage {
	// Construct and return a new message from our given parameters.
	return &CallMessage{
		MsgFrom:          from,
		MsgTo:            to,
		MsgNonce:         nonce,
		MsgValue:         value,
		MsgGas:           gasLimit,
		MsgGasPrice:      gasPrice,
		MsgGasFeeCap:     gasFeeCap,
		MsgGasTipCap:     gasTipCap,
		MsgData:          nil,
		MsgDataAbiValues: data,
	}
}

// FillFromTestChainProperties populates gas limit, price, nonce, and other fields automatically based on the worker's
// underlying test chain properties if they are not yet set.
func (m *CallMessage) FillFromTestChainProperties(chain *chain.TestChain) {
	// Set our nonce for this
	m.MsgNonce = chain.State().GetNonce(m.MsgFrom)

	// If a gas limit was not provided, allow the entire block gas limit to be used for this message.
	if m.MsgGas == 0 {
		m.MsgGas = chain.BlockGasLimit
	}

	// If a gas price was not provided, we use 1 as a default.
	if m.MsgGasPrice == nil {
		m.MsgGasPrice = big.NewInt(1)
	}

	// Setting fee and tip cap to zero alongside the NoBaseFee for the vm.Config will bypass base fee validation.
	// TODO: Set this appropriately for newer transaction types.
	m.MsgGasFeeCap = big.NewInt(0)
	m.MsgGasTipCap = big.NewInt(0)
}

func (m *CallMessage) From() common.Address { return m.MsgFrom }
func (m *CallMessage) To() *common.Address  { return m.MsgTo }
func (m *CallMessage) GasPrice() *big.Int   { return m.MsgGasPrice }
func (m *CallMessage) GasFeeCap() *big.Int  { return m.MsgGasFeeCap }
func (m *CallMessage) GasTipCap() *big.Int  { return m.MsgGasTipCap }
func (m *CallMessage) Value() *big.Int      { return m.MsgValue }
func (m *CallMessage) Gas() uint64          { return m.MsgGas }
func (m *CallMessage) Nonce() uint64        { return m.MsgNonce }
func (m *CallMessage) Data() []byte {
	// If we have message data derived from ABI values, pack them and return the data.
	if m.MsgDataAbiValues != nil {
		data, err := m.MsgDataAbiValues.Pack()
		if err != nil {
			panic(fmt.Errorf("error while packing call message ABI values: %v", err))
		}
		return data
	}

	// Otherwise we return our message data set from bytes.
	return m.MsgData
}
func (m *CallMessage) AccessList() coreTypes.AccessList { return nil }
func (m *CallMessage) IsFake() bool                     { return true }

type CallMessageDataAbiValues struct {
	Method      *abi.Method
	MethodID    []byte
	InputValues []any
}

type callMessageDataAbiValuesMarshal struct {
	MethodID    hexutil.Bytes     `json:"methodId"`
	InputValues []json.RawMessage `json:"inputValues"`
}

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
	inputValues := make([]json.RawMessage, len(d.Method.Inputs))
	for i := 0; i < len(inputValues); i++ {
		serializedInput, err := json.Marshal(valuegeneration.AbiValueToMap(&d.Method.Inputs[i].Type, d.InputValues[i]))
		if err != nil {
			return nil, err
		}
		inputValues[i] = serializedInput
	}

	// Now create our outer struct and marshal all the data and return it.
	marshalData := callMessageDataAbiValuesMarshal{
		MethodID:    d.Method.ID,
		InputValues: inputValues,
	}
	return json.Marshal(marshalData)
}

func (d *CallMessageDataAbiValues) UnmarshalJSON(b []byte) error {
	// Decode our intermediate structure
	var marshalData callMessageDataAbiValuesMarshal
	err := json.Unmarshal(b, &marshalData)
	if err != nil {
		return err
	}

	// Verify our method ID
	if len(marshalData.MethodID) != 4 {
		return fmt.Errorf("ABI call data JSON unmarshaling failed, expected a 4 byte method ID, but a %d byte one was provided", len(marshalData.MethodID))
	}

	// Copy out all of our input arguments
	inputValues := make([]any, len(marshalData.InputValues))
	for i := 0; i < len(inputValues); i++ {
		var inputValue any
		err = json.Unmarshal(marshalData.InputValues[i], &inputValue)
		if err != nil {
			return err
		}
	}

	// Set our data in our actual structure now
	d.MethodID = marshalData.MethodID
	d.InputValues = inputValues
	return nil
}
