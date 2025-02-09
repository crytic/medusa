package reverts

import (
	"encoding/json"
	"errors"
	"os"

	"github.com/crytic/medusa/compilation/abiutils"
	"github.com/crytic/medusa/logging"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/vm"
)

// RevertMetrics is used to track the number of times calls to various contracts and functions revert and why.
type RevertMetrics struct {
	// ContractRevertMetrics holds the revert metrics for each contract in the fuzzing campaign.
	ContractRevertMetrics map[string]*ContractRevertMetrics `json:"contractRevertMetrics"`
}

// ContractRevertMetrics is used to track the number of times calls to various functions in a contract revert and why.
type ContractRevertMetrics struct {
	// Name is the name of the contract.
	Name string `json:"name"`
	// FunctionRevertMetrics holds the revert metrics for each function in the contract.
	FunctionRevertMetrics map[string]*FunctionRevertMetrics `json:"functionRevertMetrics"`
}

// FunctionRevertMetrics is used to track the number of times a function reverted and why
type FunctionRevertMetrics struct {
	// Name is the name of the function.
	Name string `json:"name"`
	// TotalCalls is the total number of calls to the function.
	TotalCalls uint `json:"totalCalls"`
	// TotalReverts is the total number of times the function reverted.
	TotalReverts uint `json:"totalReverts"`
	// RevertReasonCounts holds the number of times a revert reason occurred for the function.
	RevertReasonCounts map[string]uint `json:"revertReasonCounts"`
}

// RevertMetricsUpdate is used to update the RevertMetrics struct.
// The fuzzer workers will send these updates via the metrics channel.
type RevertMetricsUpdate struct {
	ContractName    string
	FunctionName    string
	ExecutionResult *core.ExecutionResult
}

func NewRevertMetrics() *RevertMetrics {
	return &RevertMetrics{
		ContractRevertMetrics: make(map[string]*ContractRevertMetrics),
	}
}

// Update updates the RevertMetrics based on the execution result of a call sequence element.
func (m *RevertMetrics) Update(update *RevertMetricsUpdate) {
	// Guard clause for nil updates or nil execution results
	if update == nil || update.ExecutionResult == nil {
		return
	}

	// Capture the contract, function, and execution result
	contractName := update.ContractName
	functionName := update.FunctionName
	executionResult := update.ExecutionResult

	// Retrieve or create the contract revert metrics
	contractRevertMetrics := m.ContractRevertMetrics[contractName]
	if contractRevertMetrics == nil {
		contractRevertMetrics = &ContractRevertMetrics{
			Name:                  contractName,
			FunctionRevertMetrics: make(map[string]*FunctionRevertMetrics),
		}
		m.ContractRevertMetrics[contractName] = contractRevertMetrics
	}

	// Retrieve or create the function revert metrics
	functionRevertMetrics := contractRevertMetrics.FunctionRevertMetrics[functionName]
	if functionRevertMetrics == nil {
		functionRevertMetrics = &FunctionRevertMetrics{
			Name:               functionName,
			TotalCalls:         0,
			TotalReverts:       0,
			RevertReasonCounts: make(map[string]uint),
		}
		contractRevertMetrics.FunctionRevertMetrics[functionName] = functionRevertMetrics
	}

	// Increment the total calls for this contract/function combination
	functionRevertMetrics.TotalCalls++

	// Exit early if the execution result is not a revert or the error is not an EVM revert error
	if executionResult.Err == nil || (executionResult.Err != nil && !errors.Is(executionResult.Err, vm.ErrExecutionReverted)) {
		return
	}

	// Increment the reverted calls for this contract/function combination
	functionRevertMetrics.TotalReverts++

	// Try to capture if we hit an EVM panic
	panicCode := abiutils.GetSolidityPanicCode(executionResult.Err, executionResult.ReturnData, true)
	if panicCode != nil {
		revertReason := abiutils.GetPanicReason(panicCode.Uint64())
		functionRevertMetrics.RevertReasonCounts[revertReason]++
	}

	// Otherwise, use the selector of the error return data as the revert reason
	// TODO: Make sure this is what we want to use after some testing.
	if len(executionResult.ReturnData) > 4 {
		revertReason := string(executionResult.ReturnData[:4])
		functionRevertMetrics.RevertReasonCounts[revertReason]++
	}
}

// NewRevertMetricsFromPath will load a RevertMetrics object using the provided file path.
func NewRevertMetricsFromPath(path string) (*RevertMetrics, error) {
	// Guard clause for empty path
	if path == "" {
		return nil, errors.New("empty path was provided")
	}

	// Read the file
	b, err := os.ReadFile(path)
	if err != nil {
		// Don't throw an error if the file does not exist.
		if errors.Is(err, os.ErrNotExist) {
			// Return an empty object if the file does not exist.
			logging.GlobalLogger.Info("No previous revert metrics found at: ", path)
			return &RevertMetrics{}, nil
		} else {
			return nil, err
		}
	}

	// Deserialize the artifact
	var metrics *RevertMetrics
	err = json.Unmarshal(b, &metrics)
	if err != nil {
		return nil, err
	}

	return metrics, nil
}
