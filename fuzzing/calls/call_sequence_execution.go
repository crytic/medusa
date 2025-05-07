package calls

import (
	"fmt"
	"math/big"

	"github.com/crytic/medusa/chain"
	"github.com/crytic/medusa/fuzzing/config"
	"github.com/crytic/medusa/fuzzing/contracts"
	"github.com/crytic/medusa/fuzzing/executiontracer"
	"github.com/crytic/medusa/utils"
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

		// If we have a pending block, but we intend to delay this call from the last:
		//   if possible, we edit the block number and timestamp directly, same as the cheatcodes do it;
		//   otherwise, we commit the pending block and start a new one.
		// If we have a pending contract creation or deletion, we need to commit the pending block to finalize the creation/deletion.
		if chain.PendingBlock() != nil && callSequenceElement.BlockNumberDelay > 0 {
			if !chain.HasPendingStateChanges() && chain.PendingBlockContext() != nil {
				// The minimum step between blocks must be 1 in block number and timestamp, so we ensure this is the
				// case.
				numberDelay := callSequenceElement.BlockNumberDelay
				timeDelay := callSequenceElement.BlockTimestampDelay
				if timeDelay == 0 {
					timeDelay = 1
				}
				// Each timestamp/block number must be unique as well, so we cannot jump more block numbers than time.
				if numberDelay > timeDelay {
					numberDelay = timeDelay
				}

				newBlockNum := big.NewInt(0).Add(chain.PendingBlockContext().BlockNumber, big.NewInt(int64(numberDelay)))
				chain.PendingBlockContext().BlockNumber.Set(newBlockNum)
				chain.PendingBlock().Header.Number.Set(newBlockNum)

				newTimestamp := chain.PendingBlockContext().Time + timeDelay
				chain.PendingBlockContext().Time = newTimestamp
				chain.PendingBlock().Header.Time = newTimestamp
			} else {
				err := chain.PendingBlockCommit()
				if err != nil {
					return callSequenceExecuted, err
				}
			}
		}

		// If we have no pending block to add a tx containing our call to, we must create one.
		if chain.PendingBlock() == nil {
			// The minimum step between blocks must be 1 in block number and timestamp, so we ensure this is the
			// case.
			numberDelay := callSequenceElement.BlockNumberDelay
			timeDelay := callSequenceElement.BlockTimestampDelay
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
				return callSequenceExecuted, err
			}
		}

		// Add our transaction to this block.
		err = chain.PendingBlockAddTx(callSequenceElement.Call.ToCoreMessage(), additionalTracers...)
		if err != nil {
			return callSequenceExecuted, err
		}

		// Update our chain reference for this element.
		callSequenceElement.ChainReference = &CallSequenceElementChainReference{
			Block:            chain.PendingBlock(),
			TransactionIndex: len(chain.PendingBlock().Messages) - 1,
		}

		// Add to our executed call sequence
		callSequenceExecuted = append(callSequenceExecuted, callSequenceElement)

		// We added our call to the block as a transaction. Call our step function with the update and check
		// if it returned an error.
		if executionCheckFunc != nil {
			execCheckFuncRequestedBreak, err = executionCheckFunc(callSequenceExecuted)
			if err != nil {
				return callSequenceExecuted, err
			}
		}

		// If post-execution check requested we break execution, break out of our loop
		if execCheckFuncRequestedBreak {
			break
		}
	}
	return callSequenceExecuted, nil
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
func ExecuteCallSequenceWithExecutionTracer(testChain *chain.TestChain, contractDefinitions contracts.Contracts, callSequence CallSequence, verbosity config.VerbosityLevel) (CallSequence, error) {
	// Create a new execution tracer
	executionTracer := executiontracer.NewExecutionTracer(contractDefinitions, testChain, verbosity)
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

	// Determine which elements of the call sequence to trace based on verbosity level
	traceFrom := len(callSequence) - 1 // Default: only trace the last element

	// VeryVeryVerbose (level 2): Trace all elements in the call sequence
	if verbosity == config.VeryVeryVerbose {
		traceFrom = 0
	}
	// Note: For Verbose (level 0), only top-level call frames are shown, handled in execution_trace.go
	// For VeryVerbose (level 1), full detail for the last element is shown (default behavior)

	// Attach the execution trace for each requested call sequence element
	for ; traceFrom < len(callSequence); traceFrom++ {
		callSequenceElement := callSequence[traceFrom]
		hash := utils.MessageToTransaction(callSequenceElement.Call.ToCoreMessage()).Hash()
		callSequenceElement.ExecutionTrace = executionTracer.GetTrace(hash)
	}

	return executedCallSeq, err
}
