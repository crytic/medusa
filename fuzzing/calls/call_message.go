package calls

import (
	"math/big"

	"github.com/crytic/medusa/chain"
	"github.com/crytic/medusa/logging"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	coreTypes "github.com/ethereum/go-ethereum/core/types"
	"golang.org/x/exp/slices"
)

// The following directives will be picked up by the `go generate` command to generate JSON marshaling code from
// templates defined below. They should be preserved for re-use in case we change our structures.
//go:generate go get github.com/fjl/gencodec
//go:generate go run github.com/fjl/gencodec -type CallMessage -field-override callMessageMarshaling -out gen_call_message_json.go

// CallMessage implements and extends Ethereum's coreTypes.Message, used to apply EVM/state updates.
type CallMessage struct {
	// From represents a core.Message's from parameter (sender), indicating who sent a transaction/message to the
	// Ethereum core to apply a state update.
	From common.Address `json:"from"`

	// To represents the receiving address for a given core.Message.
	To *common.Address `json:"to"`

	// Nonce represents the core.Message sender's nonce
	Nonce uint64 `json:"nonce"`

	// Value represents ETH value to be sent to the receiver of the message.
	Value *big.Int `json:"value"`

	// GasLimit represents the maximum amount of gas the sender is willing to spend to cover the cost of executing the
	// message or transaction.
	GasLimit uint64 `json:"gasLimit"`

	// GasPrice represents the price which the sender is willing to pay for each unit of gas used during execution
	// of the message.
	GasPrice *big.Int `json:"gasPrice"`

	// GasFeeCap represents the maximum fee to enforce for gas related costs (related to 1559 transaction executed).
	// The use of nil here indicates that the gas price oracle should be relied on instead.
	GasFeeCap *big.Int `json:"gasFeeCap"`

	// GasTipCap represents the fee cap to use for 1559 transaction. The use of nil here indicates that the gas price
	// oracle should be relied on instead.
	GasTipCap *big.Int `json:"gasTipCap"`

	// Data represents the underlying message data to be sent to the receiver. If the receiver is a smart contract,
	// this will likely house your call parameters and other serialized data. If MsgDataAbiValues is non-nil, this
	// value is not used.
	Data []byte `json:"data,omitempty"`

	// DataAbiValues represents the underlying message data to be sent to the receiver. If the receiver is a smart
	// contract, this will likely house your call parameters and other serialized data. This overrides Data if it is
	// set, allowing Data to be sourced from method ABI input arguments instead.
	DataAbiValues *CallMessageDataAbiValues `json:"dataAbiValues,omitempty"`

	// AccessList represents a core.Message's AccessList parameter which represents the storage slots and contracts
	// that will be accessed during the execution of this message.
	AccessList coreTypes.AccessList

	// SkipAccountChecks represents a core.Message's SkipAccountChecks. If it is set to true, then the message nonce
	// is not checked against the account nonce in state and will not verify if the sender is an EOA.
	SkipAccountChecks bool
}

// callMessageMarshaling is a structure that overrides field types during JSON marshaling. It allows CallMessage to
// have its custom marshaling methods auto-generated and will handle type conversions for serialization purposes.
// For example, this enables serialization of big.Int but specifying a different field type to control serialization.
type callMessageMarshaling struct {
	Value     *hexutil.Big
	GasPrice  *hexutil.Big
	GasFeeCap *hexutil.Big
	GasTipCap *hexutil.Big
	Data      hexutil.Bytes
}

// NewCallMessage instantiates a new call message from a given set of parameters, with call data set from bytes.
func NewCallMessage(from common.Address, to *common.Address, nonce uint64, value *big.Int, gasLimit uint64, gasPrice, gasFeeCap, gasTipCap *big.Int, data []byte) *CallMessage {
	// Construct and return a new message from our given parameters.
	return &CallMessage{
		From:              from,
		To:                to,
		Nonce:             nonce,
		Value:             value,
		GasLimit:          gasLimit,
		GasPrice:          gasPrice,
		GasFeeCap:         gasFeeCap,
		GasTipCap:         gasTipCap,
		Data:              data,
		DataAbiValues:     nil,
		AccessList:        nil,
		SkipAccountChecks: false,
	}
}

// NewCallMessageWithAbiValueData instantiates a new call message from a given set of parameters, with call data set
// from method ABI specified inputs.
func NewCallMessageWithAbiValueData(from common.Address, to *common.Address, nonce uint64, value *big.Int, gasLimit uint64, gasPrice, gasFeeCap, gasTipCap *big.Int, abiData *CallMessageDataAbiValues) *CallMessage {
	// Pack the ABI value data
	var data []byte
	var err error
	if abiData != nil {
		data, err = abiData.Pack()
		if err != nil {
			logging.GlobalLogger.Panic("Failed to pack call message ABI values", err)
		}
	}

	// Construct and return a new message from our given parameters.
	return &CallMessage{
		From:              from,
		To:                to,
		Nonce:             nonce,
		Value:             value,
		GasLimit:          gasLimit,
		GasPrice:          gasPrice,
		GasFeeCap:         gasFeeCap,
		GasTipCap:         gasTipCap,
		Data:              data,
		DataAbiValues:     abiData,
		AccessList:        nil,
		SkipAccountChecks: false,
	}
}

// WithDataAbiValues resets the call message's data and ABI values, ensuring the values are in sync and
// reusing the other existing fields.
func (m *CallMessage) WithDataAbiValues(abiData *CallMessageDataAbiValues) {
	if abiData == nil {
		logging.GlobalLogger.Panic("Method ABI and data should always be defined")
	}

	// Pack the ABI value data
	var data []byte
	var err error
	data, err = abiData.Pack()
	if err != nil {
		logging.GlobalLogger.Panic("Failed to pack call message ABI values", err)
	}
	// Set our data and ABI values
	m.DataAbiValues = abiData
	m.Data = data
}

// FillFromTestChainProperties populates gas limit, price, nonce, and other fields automatically based on the worker's
// underlying test chain properties if they are not yet set.
func (m *CallMessage) FillFromTestChainProperties(chain *chain.TestChain) {
	// Set our nonce for this
	m.Nonce = chain.State().GetNonce(m.From)

	// If a gas limit was not provided, allow the entire block gas limit to be used for this message.
	if m.GasLimit == 0 {
		m.GasLimit = chain.BlockGasLimit
	}

	// If a gas price was not provided, we use 1 as a default.
	if m.GasPrice == nil {
		m.GasPrice = big.NewInt(1)
	}

	// Setting fee and tip cap to zero alongside the NoBaseFee for the vm.Config will bypass base fee validation.
	// TODO: Set this appropriately for newer transaction types.
	m.GasFeeCap = big.NewInt(0)
	m.GasTipCap = big.NewInt(0)
}

// Clone creates a copy of the given message and its underlying components, or an error if one occurs.
func (m *CallMessage) Clone() (*CallMessage, error) {
	// Clone our underlying ABI values data if we have any.
	clonedAbiValues, err := m.DataAbiValues.Clone()
	if err != nil {
		return nil, err
	}

	// Create a message with the same data copied over.
	clone := &CallMessage{
		From:              m.From,
		To:                m.To, // this value should be read-only, so we re-use it rather than cloning.
		Nonce:             m.Nonce,
		Value:             new(big.Int).Set(m.Value),
		GasLimit:          m.GasLimit,
		GasPrice:          new(big.Int).Set(m.GasPrice),
		GasFeeCap:         new(big.Int).Set(m.GasFeeCap),
		GasTipCap:         new(big.Int).Set(m.GasTipCap),
		Data:              slices.Clone(m.Data),
		DataAbiValues:     clonedAbiValues,
		AccessList:        m.AccessList,
		SkipAccountChecks: m.SkipAccountChecks,
	}
	return clone, nil
}

func (m *CallMessage) ToCoreMessage() *core.Message {
	return &core.Message{
		To:                m.To,
		From:              m.From,
		Nonce:             m.Nonce,
		Value:             new(big.Int).Set(m.Value),
		GasLimit:          m.GasLimit,
		GasPrice:          new(big.Int).Set(m.GasPrice),
		GasFeeCap:         new(big.Int).Set(m.GasFeeCap),
		GasTipCap:         new(big.Int).Set(m.GasTipCap),
		Data:              slices.Clone(m.Data),
		AccessList:        m.AccessList,
		SkipAccountChecks: m.SkipAccountChecks,
	}
}
