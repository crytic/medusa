package reverts

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/crytic/medusa-geth/accounts/abi"
	"github.com/crytic/medusa-geth/core"
	"github.com/crytic/medusa-geth/core/vm"
	"github.com/crytic/medusa/compilation/abiutils"
	"github.com/crytic/medusa/logging"
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
	// Pct is the percent of times a call to this function reverted
	Pct float64 `json:"pct"`
	// PrevPct is the percentage of total calls that reverted in the previous campaign.
	PrevPct float64 `json:"prevPct"`
	// RevertReasonMetrics holds the revert reason metrics for the function.
	RevertReasonMetrics map[string]*RevertReasonMetrics `json:"revertReasonMetrics"`
}

// RevertReasonMetrics is used to track the number of times a revert reason occurred for a function.
type RevertReasonMetrics struct {
	// Reason is the revert reason.
	Reason string `json:"reason"`
	// Count is the number of times the revert reason occurred.
	Count uint `json:"count"`
	// Pct is the percentage of total calls to that resulted in the revert reason.
	Pct float64 `json:"pct"`
	// PrevPct is the percentage of total calls to that resulted in the revert reason in the previous campaign.
	PrevPct float64 `json:"prevPct"`
}

// RevertMetricsUpdate is used to update the RevertMetrics struct.
// The fuzzer workers will send these updates via the metrics channel.
type RevertMetricsUpdate struct {
	// ContractName is the name of the contract which was called
	ContractName string
	// FunctionName is the name of the function which was called
	FunctionName string
	// ExecutionResult is the result of the execution
	ExecutionResult *core.ExecutionResult
}

// NewRevertMetrics will create a new RevertMetrics object.
func NewRevertMetrics() *RevertMetrics {
	return &RevertMetrics{
		ContractRevertMetrics: make(map[string]*ContractRevertMetrics),
	}
}

// NewRevertMetricsFromPath will load a RevertMetrics object using the provided file path.
func NewRevertMetricsFromPath(path string) (*RevertMetrics, error) {
	// Guard clause for empty path
	if path == "" {
		return nil, errors.New("empty path was provided")
	}

	// Get the file path
	filePath := filepath.Join(path, "revert_report.json")

	// Read the file
	b, err := os.ReadFile(filePath)
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

// Update updates the RevertMetrics based on the execution result of a call sequence element.
// A mapping of error IDs to their custom error names is also provided to aid with decoding the revert reason.
func (m *RevertMetrics) Update(update *RevertMetricsUpdate, errorIDs map[string]abi.Error) {
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
			Name:                functionName,
			RevertReasonMetrics: make(map[string]*RevertReasonMetrics),
		}
		contractRevertMetrics.FunctionRevertMetrics[functionName] = functionRevertMetrics
	}

	// Increment the total calls for this contract/function combination
	functionRevertMetrics.TotalCalls++

	// Exit early if the execution result is not a revert or the error is not an EVM revert error
	if executionResult.Err == nil || (executionResult.Err != nil && !errors.Is(executionResult.Err, vm.ErrExecutionReverted)) {
		return
	}

	// Now we know that the transaction reverted
	functionRevertMetrics.TotalReverts++

	revertReason := "unknown"
	// First, figure out whether the revert reason is a panic
	panicCode := abiutils.GetSolidityPanicCode(executionResult.Err, executionResult.ReturnData, false)
	if panicCode != nil {
		revertReason = abiutils.GetPanicReason(panicCode.Uint64())
	}

	// Next, check to see if the error is a custom, user-defined error
	if revertReason == "unknown" && len(executionResult.ReturnData) >= 4 {
		if err, ok := errorIDs[hex.EncodeToString(executionResult.ReturnData[:4])]; ok {
			revertReason = err.Name
		}
	}

	// Finally, we will try to decode an error string (e.g. `require("this is an error")`)
	if revertReason == "unknown" {
		revertReasonPtr := abiutils.GetSolidityRevertErrorString(executionResult.Err, executionResult.ReturnData)
		if revertReasonPtr != nil {
			revertReason = *revertReasonPtr
		}
	}

	revertReasonMetrics := functionRevertMetrics.RevertReasonMetrics[revertReason]
	if revertReasonMetrics == nil {
		revertReasonMetrics = &RevertReasonMetrics{
			Reason: revertReason,
		}
		functionRevertMetrics.RevertReasonMetrics[revertReason] = revertReasonMetrics
	}
	revertReasonMetrics.Count++
}

// Finalize finalizes the revert metrics by updating the percentages for each function and revert reason.
// Additionally, if an optional RevertMetrics object is provided, it is merged into the current RevertMetrics object.
func (m *RevertMetrics) Finalize(other *RevertMetrics) {
	// Iterate over the contract revert metrics in the current object
	for contractName, contractRevertMetrics := range m.ContractRevertMetrics {
		var otherContractRevertMetrics *ContractRevertMetrics
		if other != nil {
			otherContractRevertMetrics = other.ContractRevertMetrics[contractName]
		}
		for functionName, functionRevertMetrics := range contractRevertMetrics.FunctionRevertMetrics {
			// Update the percentage
			functionRevertMetrics.Pct = float64(functionRevertMetrics.TotalReverts) / float64(functionRevertMetrics.TotalCalls)

			// Update the previous percentage if the function existed in the previous campaign
			var otherFunctionRevertMetrics *FunctionRevertMetrics
			if otherContractRevertMetrics != nil {
				if otherFunctionRevertMetrics = otherContractRevertMetrics.FunctionRevertMetrics[functionName]; otherFunctionRevertMetrics != nil {
					functionRevertMetrics.PrevPct = otherFunctionRevertMetrics.Pct
				}
			}
			for revertReason, revertReasonMetrics := range functionRevertMetrics.RevertReasonMetrics {
				// Update the percentage
				revertReasonMetrics.Pct = float64(revertReasonMetrics.Count) / float64(functionRevertMetrics.TotalCalls)

				// Update the previous percentage if the revert reason exists in the other revert metrics artifact
				if otherFunctionRevertMetrics == nil {
					continue
				}
				if otherRevertReasonMetrics := otherFunctionRevertMetrics.RevertReasonMetrics[revertReason]; otherRevertReasonMetrics != nil {
					revertReasonMetrics.PrevPct = otherRevertReasonMetrics.Pct
				}
			}
		}
	}
}
