package types

import (
	"encoding/hex"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/trailofbits/medusa/compilation/types"
	"golang.org/x/crypto/sha3"
	"strings"
)

// DeployedMethod describes a method which is accessible through a contract actively deployed on a fuzzing.TestNode.
type DeployedMethod struct {
	// Address represents the Ethereum address where the deployed contract containing the method exists.
	Address common.Address

	// Contract describes the contract which was deployed and contains the target method.
	Contract types.CompiledContract

	// Method describes the method which is available through the deployed contract.
	Method abi.Method
}

// Hash calculated a unique hash string for the given contract and method, so it can be discerned as targeting different
// methods than another instance of the same contract and method deployed to another address. Returns the computed hash.
func (d *DeployedMethod) Hash() string {
	// Calculate a hash which is unique per contract/method
	srcData := strings.Join([]string{d.Contract.InitBytecode, d.Contract.RuntimeBytecode, d.Contract.SrcMapsInit, d.Contract.SrcMapsRuntime, d.Method.Sig}, ",")
	hash := sha3.NewLegacyKeccak256().Sum([]byte(srcData))
	return hex.EncodeToString(hash)
}
