package types

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	coreTypes "github.com/ethereum/go-ethereum/core/types"
	"math/big"
)

//go:generate go get github.com/fjl/gencodec
//go:generate go run github.com/fjl/gencodec -type CallMessage -field-override callMessageMarshaling -out gen_message_json.go

// CallMessage implements Ethereum's core.Message, used to apply EVM/state updates.
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
	MsgGasPrice *big.Int `json:"gas_price"`

	// MsgGasFeeCap represents the maximum fee to enforce for gas related costs (related to 1559 transaction executed).
	// The use of nil here indicates that the gas price oracle should be relied on instead.
	MsgGasFeeCap *big.Int `json:"gas_fee_cap"`

	// MsgGasTipCap represents the fee cap to use for 1559 transaction. The use of nil here indicates that the gas price
	// oracle should be relied on instead.
	MsgGasTipCap *big.Int `json:"gas_tip_cap"`

	// MsgData represents the underlying message data to be sent to the receiver. If the receiver is a smart contract,
	// this will likely house your call parameters and other serialized data.
	MsgData []byte `json:"data"`
}

// callMessageMarshaling is a structure that overrides field types during JSON marshaling. It allows CallMessage to
// have its custom marshaling methods auto-generated and will handle type conversions for serialization purposes.
// For example, this enables serialization of big.Int but specifying a different field type to serialize it as.
type callMessageMarshaling struct {
	// MsgValue represents the marshaling rules for CallMessage.MsgValue.
	MsgValue *hexutil.Big

	// MsgGasPrice represents the marshaling rules for CallMessage.MsgGasPrice.
	MsgGasPrice *hexutil.Big

	// MsgGasFeeCap represents the marshaling rules for CallMessage.MsgGasFeeCap.
	MsgGasFeeCap *hexutil.Big

	// MsgGasTipCap represents the marshaling rules for CallMessage.MsgGasTipCap.
	MsgGasTipCap *hexutil.Big
}

// NewCallMessage instantiates a new call message from a given set of parameters.
func NewCallMessage(from common.Address, to *common.Address, nonce uint64, value *big.Int, gasLimit uint64, gasPrice, gasFeeCap, gasTipCap *big.Int, data []byte) *CallMessage {
	// Construct and return a new message from our given parameters.
	return &CallMessage{
		MsgFrom:      from,
		MsgTo:        to,
		MsgNonce:     nonce,
		MsgValue:     value,
		MsgGas:       gasLimit,
		MsgGasPrice:  gasPrice,
		MsgGasFeeCap: gasFeeCap,
		MsgGasTipCap: gasTipCap,
		MsgData:      data,
	}
}

func (m *CallMessage) From() common.Address             { return m.MsgFrom }
func (m *CallMessage) Nonce() uint64                    { return m.MsgNonce }
func (m *CallMessage) IsFake() bool                     { return true }
func (m *CallMessage) To() *common.Address              { return m.MsgTo }
func (m *CallMessage) GasPrice() *big.Int               { return m.MsgGasPrice }
func (m *CallMessage) GasFeeCap() *big.Int              { return m.MsgGasFeeCap }
func (m *CallMessage) GasTipCap() *big.Int              { return m.MsgGasTipCap }
func (m *CallMessage) Gas() uint64                      { return m.MsgGas }
func (m *CallMessage) Value() *big.Int                  { return m.MsgValue }
func (m *CallMessage) Data() []byte                     { return m.MsgData }
func (m *CallMessage) AccessList() coreTypes.AccessList { return nil }

// ToEVMMessage returns a representation of the CallMessage that is compatible with EVM methods.
func (m *CallMessage) ToEVMMessage() coreTypes.Message {
	return coreTypes.NewMessage(m.From(), m.To(), m.Nonce(), m.Value(), m.Gas(), m.GasPrice(), m.GasFeeCap(), m.GasTipCap(), m.Data(), nil, true)
}
