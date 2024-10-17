package calls

import (
	"fmt"

	"math/big"

	"github.com/crytic/medusa/chain"
	"github.com/crytic/medusa/fuzzing/contracts"
	"github.com/crytic/medusa/fuzzing/executiontracer"
	"github.com/crytic/medusa/utils"
	"github.com/ethereum/go-ethereum/core"
)

// ExecuteCallSequenceFetchElementFunc describes a function that is called to obtain the next call sequence element to
// execute. It is given the current call index in the sequence.
// Returns the call sequence element to execute, or an error if one occurs. If the call sequence element is nil,
// it indicates the end of the sequence and execution breaks.
type ExecuteCallSequenceFetchElementFunc func(index int) (*CallSequenceElement, error)

// ExecuteCallSequenceExecutionCheckFunc describes a function that is called after each call is executed in a
// sequence. It is given the currently executed call sequence to this point.
// Returns a boolean indicating if the sequence execution should break, or an error if one occurs.
type ExecuteCallSequenceExecutionCheckFunc func(currentExecutedSequence CallSequence) (bool, error)

// ExecuteCallSequenceIteratively executes a CallSequence upon a provided chain iteratively. It ensures calls are
// included in blocks which adhere to the CallSequence properties (such as delays) as much as possible.
// A "fetch next call" function is provided to fetch the next element to execute.
// A "post element executed check" function is provided to check whether execution should stop after each element is
// executed.
// Returns the call sequence which was executed and an error if one occurs.
func ExecuteCallSequenceIteratively(chain *chain.TestChain, fetchElementFunc ExecuteCallSequenceFetchElementFunc, executionCheckFunc ExecuteCallSequenceExecutionCheckFunc, additionalTracers ...*chain.TestChainTracer) (CallSequence, error) {
	// If there is no fetch element function provided, throw an error
	if fetchElementFunc == nil {
		return nil, fmt.Errorf("could not execute call sequence on chain as the 'fetch element function' provided was nil")
	}

	// Create a call sequence to track all elements executed throughout this operation.
	var callSequenceExecuted CallSequence

	// Create a variable to track if the post-execution check operation requested we break execution.
	execCheckFuncRequestedBreak := false

	// Loop through each sequence element in our sequence we'll want to execute.
	for i := 0; true; i++ {
		// Call our "fetch next call" function and obtain our next call sequence element.
		callSequenceElement, err := fetchElementFunc(i)
		if err != nil {
			return callSequenceExecuted, err
		}

		// If we are at the end of our sequence, break out of our execution loop.
		if callSequenceElement == nil {
			break
		}

		// Process the call sequence element
		err = processCallSequenceElement(chain, callSequenceElement, &callSequenceExecuted, additionalTracers...)
		if err != nil {
			return callSequenceExecuted, err
		}

		// We added our call to the block as a transaction. Call our step function with the update and check
		// if it returned an error.
		if executionCheckFunc != nil {
			execCheckFuncRequestedBreak, err = executionCheckFunc(callSequenceExecuted)
			if err != nil {
				return callSequenceExecuted, err
			}

			// If post-execution check requested we break execution, break out of our "retry loop"
			if execCheckFuncRequestedBreak {
				break
			}
		}
	}

	return callSequenceExecuted, nil
}

// processCallSequenceElement handles the execution of a single call sequence element, including the setup hook.
func processCallSequenceElement(chain *chain.TestChain, callSequenceElement *CallSequenceElement, callSequenceExecuted *CallSequence, additionalTracers ...*chain.TestChainTracer) error {
	// We try to add the transaction with our call more than once. If the pending block is too full, we may hit a
	// block gas limit, which we handle by committing the pending block without this tx, and creating a new pending
	// block that is empty to try adding this tx there instead.
	// If we encounter an error on an empty block, we throw the error as there is nothing more we can do.

	// Process contract setup hook if present
	if callSequenceElement.Contract.SetupHook != nil {
		err := executeContractSetupHook(chain, callSequenceElement, callSequenceExecuted)
		if err != nil {
			return err
		}
	}

	// Process the main call sequence element
	return executeCall(chain, callSequenceElement, callSequenceExecuted, additionalTracers...)
}

// executeContractSetupHook processes the contract setup hook for the call sequence element.
func executeContractSetupHook(chain *chain.TestChain, callSequenceElement *CallSequenceElement, callSequenceExecuted *CallSequence) error {
	// Get our contract setup hook
	contractSetupHook := callSequenceElement.Contract.SetupHook

	// Create a call targeting our setup hook
	msg := NewCallMessageWithAbiValueData(contractSetupHook.DeployerAddress, callSequenceElement.Call.To, 0, big.NewInt(0), callSequenceElement.Call.GasLimit, nil, nil, nil, &CallMessageDataAbiValues{
		Method:      contractSetupHook.Method,
		InputValues: nil,
	})
	msg.FillFromTestChainProperties(chain)

	// Execute the call
	// If we have no pending block to add a tx containing our call to, we must create one.
	err := addTxToPendingBlock(chain, callSequenceElement.BlockNumberDelay, callSequenceElement.BlockTimestampDelay, msg.ToCoreMessage())
	if err != nil {
		return err
	}

	setupCallSequenceElement := NewCallSequenceElement(callSequenceElement.Contract, msg, callSequenceElement.BlockNumberDelay, callSequenceElement.BlockTimestampDelay)
	setupCallSequenceElement.ChainReference = &CallSequenceElementChainReference{
		Block:            chain.PendingBlock(),
		TransactionIndex: len(chain.PendingBlock().Messages) - 1,
	}

	// Register the call in our call sequence so it gets registered in coverage.
	*callSequenceExecuted = append(*callSequenceExecuted, setupCallSequenceElement)
	return nil
}

// executeCall processes the main call of the call sequence element.
func executeCall(chain *chain.TestChain, callSequenceElement *CallSequenceElement, callSequenceExecuted *CallSequence, additionalTracers ...*chain.TestChainTracer) error {
	// Update call sequence element call message if setup hook was executed
	if callSequenceElement.Contract.SetupHook != nil {
		callSequenceElement.Call.FillFromTestChainProperties(chain)
	}

	// Try to add our transaction to this block.
	err := addTxToPendingBlock(chain, callSequenceElement.BlockNumberDelay, callSequenceElement.BlockTimestampDelay, callSequenceElement.Call.ToCoreMessage(), additionalTracers...)
	if err != nil {
		return err
	}

	// Update our chain reference for this element.
	callSequenceElement.ChainReference = &CallSequenceElementChainReference{
		Block:            chain.PendingBlock(),
		TransactionIndex: len(chain.PendingBlock().Messages) - 1,
	}

	// Add to our executed call sequence
	*callSequenceExecuted = append(*callSequenceExecuted, callSequenceElement)
	return nil
}

// addTxToPendingBlock attempts to add a transaction to the pending block, handling block creation and retries as necessary.
func addTxToPendingBlock(chain *chain.TestChain, numberDelay, timeDelay uint64, txMessage *core.Message, additionalTracers ...*chain.TestChainTracer) error {
	for {
		// If we have a pending block, but we intend to delay this call from the last, we commit that block.
		if chain.PendingBlock() != nil && numberDelay > 0 {
			err := chain.PendingBlockCommit()
			if err != nil {
				return err
			}
		}

		// If we have no pending block to add a tx containing our call to, we must create one.
		if chain.PendingBlock() == nil {
			// The minimum step between blocks must be 1 in block number and timestamp, so we ensure this is the
			// case.
			if numberDelay == 0 {
				numberDelay = 1
			}
			if timeDelay == 0 {
				timeDelay = 1
			}

			// Each timestamp/block number must be unique as well, so we cannot jump more block numbers than time.
			if numberDelay > timeDelay {
				numberDelay = timeDelay
			}
			_, err := chain.PendingBlockCreateWithParameters(chain.Head().Header.Number.Uint64()+numberDelay, chain.Head().Header.Time+timeDelay, nil)
			if err != nil {
				return err
			}
		}

		// Try to add our transaction to this block.
		err := chain.PendingBlockAddTx(txMessage, additionalTracers...)
		if err != nil {
			// If we encountered a block gas limit error, this tx is too expensive to fit in this block.
			// If there are other transactions in the block, this makes sense. The block is "full".
			// In that case, we commit the pending block without this tx, and create a new pending block to add
			// our tx to, and iterate to try and add it again.
			// TODO: This should also check the condition that this is a block gas error specifically. For now, we
			//  simply assume it is and try processing in an empty block (if that fails, that error will be
			//  returned).
			if len(chain.PendingBlock().Messages) > 0 {
				err := chain.PendingBlockCommit()
				if err != nil {
					return err
				}
				continue
			}

			// If there are no transactions in our block, and we failed to add this one, return the error
			return err
		}

		// We didn't encounter an error, so we were successful in adding this transaction. Break out of this
		// inner "retry loop" and move onto processing the next element in the outer loop.
		break
	}

	return nil
}

// ExecuteCallSequence executes a provided CallSequence on the provided chain.
// It returns the slice of the call sequence which was tested, and an error if one occurred.
// If no error occurred, it can be expected that the returned call sequence contains all elements originally provided.
func ExecuteCallSequence(chain *chain.TestChain, callSequence CallSequence) (CallSequence, error) {
	// Execute our sequence with a simple fetch operation provided to obtain each element.
	fetchElementFunc := func(currentIndex int) (*CallSequenceElement, error) {
		if currentIndex < len(callSequence) {
			return callSequence[currentIndex], nil
		}
		return nil, nil
	}

	return ExecuteCallSequenceIteratively(chain, fetchElementFunc, nil)
}

// ExecuteCallSequenceWithExecutionTracer attaches an executiontracer.ExecutionTracer to ExecuteCallSequenceIteratively and attaches execution traces to the call sequence elements.
func ExecuteCallSequenceWithExecutionTracer(testChain *chain.TestChain, contractDefinitions contracts.Contracts, callSequence CallSequence, verboseTracing bool) (CallSequence, error) {
	// Create a new execution tracer
	executionTracer := executiontracer.NewExecutionTracer(contractDefinitions, testChain.CheatCodeContracts())
	defer executionTracer.Close()

	// Execute our sequence with a simple fetch operation provided to obtain each element.
	fetchElementFunc := func(currentIndex int) (*CallSequenceElement, error) {
		if currentIndex < len(callSequence) {
			return callSequence[currentIndex], nil
		}
		return nil, nil
	}

	// Execute the call sequence and attach the execution tracer
	executedCallSeq, err := ExecuteCallSequenceIteratively(testChain, fetchElementFunc, nil, executionTracer.NativeTracer())

	// By default, we only trace the last element in the call sequence.
	traceFrom := len(callSequence) - 1
	// If verbose tracing is enabled, we want to trace all elements in the call sequence.
	if verboseTracing {
		traceFrom = 0
	}

	// Attach the execution trace for each requested call sequence element
	for ; traceFrom < len(callSequence); traceFrom++ {
		callSequenceElement := callSequence[traceFrom]
		hash := utils.MessageToTransaction(callSequenceElement.Call.ToCoreMessage()).Hash()
		callSequenceElement.ExecutionTrace = executionTracer.GetTrace(hash)
	}

	return executedCallSeq, err
}
