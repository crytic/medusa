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
	MsgFrom      common.Address  `json:"from"`
	MsgTo        *common.Address `json:"to"`
	MsgNonce     uint64          `json:"nonce"`
	MsgValue     *big.Int        `json:"gas"`
	MsgGas       uint64          `json:"value"`
	MsgGasPrice  *big.Int        `json:"gas_price"`
	MsgGasFeeCap *big.Int        `json:"gas_fee_cap"`
	MsgGasTipCap *big.Int        `json:"gas_tip_cap"`
	MsgData      []byte          `json:"data"`
}

// callMessageMarshaling is a structure that overrides field types during JSON marshaling. It allows CallMessage to
// have its custom marshaling methods auto-generated and will handle type conversions for serialization purposes.
// For example, this enables serialization of big.Int but specifying a different field type to serialize it as.
type callMessageMarshaling struct {
	MsgValue     *hexutil.Big
	MsgGasPrice  *hexutil.Big
	MsgGasFeeCap *hexutil.Big
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
