package calls

import (
	"fmt"
	"github.com/crytic/medusa/chain"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	coreTypes "github.com/ethereum/go-ethereum/core/types"
	"golang.org/x/exp/slices"
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

	// MsgDataAbiValues represents the underlying message data to be sent to the receiver. If the receiver is a smart
	// contract, this will likely house your call parameters and other serialized data. This overrides MsgData if it is
	// set, allowing Data to be sourced from method ABI input arguments instead.
	MsgDataAbiValues *CallMessageDataAbiValues `json:"dataAbiValues,omitempty"`
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

// Clone creates a copy of the given message and its underlying components, or an error if one occurs.
func (m *CallMessage) Clone() (*CallMessage, error) {
	// Clone our underlying ABI values data if we have any.
	clonedAbiValues, err := m.MsgDataAbiValues.Clone()
	if err != nil {
		return nil, err
	}

	// Create a message with the same data copied over.
	clone := &CallMessage{
		MsgFrom:          m.MsgFrom,
		MsgTo:            m.MsgTo, // this value should be read-only, so we re-use it rather than cloning.
		MsgNonce:         m.MsgNonce,
		MsgValue:         new(big.Int).Set(m.MsgValue),
		MsgGas:           m.MsgGas,
		MsgGasPrice:      new(big.Int).Set(m.MsgGasPrice),
		MsgGasFeeCap:     new(big.Int).Set(m.MsgGasFeeCap),
		MsgGasTipCap:     new(big.Int).Set(m.MsgGasTipCap),
		MsgData:          slices.Clone(m.MsgData),
		MsgDataAbiValues: clonedAbiValues,
	}
	return clone, nil
}
