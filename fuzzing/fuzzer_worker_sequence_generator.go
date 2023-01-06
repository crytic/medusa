package fuzzing

import (
	"fmt"
	"github.com/trailofbits/medusa/fuzzing/calls"
	"github.com/trailofbits/medusa/fuzzing/valuegeneration"
	"github.com/trailofbits/medusa/utils/randomutils"
	"math/big"
)

// CallSequenceGenerator generates call sequences iteratively per element, for use in fuzzing campaigns.
type CallSequenceGenerator struct {
	// worker describes the parent FuzzerWorker using this mutator. Calls will be generated against deployed contract
	// methods known to the worker.
	worker *FuzzerWorker

	mutationChoices *randomutils.WeightedRandomChooser[func() error]

	baseSequence               calls.CallSequence
	index                      int
	modifyCallPreExecutionFunc func(element *calls.CallSequenceElement) error
}

// NewCallSequenceGenerator creates a CallSequenceGenerator to generate call sequences for use in fuzzing campaigns.
func NewCallSequenceGenerator(worker *FuzzerWorker) *CallSequenceGenerator {
	return &CallSequenceGenerator{
		worker: worker,
	}
}

// NewSequence prepares the CallSequenceGenerator so that GenerateElement can be called to obtain each element of the
// call sequence, as specified by the provided length.
func (g *CallSequenceGenerator) NewSequence(length int) {
	// Reset the state of our generator.
	g.baseSequence = make(calls.CallSequence, length)
	g.index = 0
	g.modifyCallPreExecutionFunc = nil

	// Next, we'll decide whether to create a new call sequence or mutating existing corpus call sequences.
	// If we wish to generate an entirely new call, we leave our array entry nil.
	// If we based our call sequence off of corpus entries, we may apply mutations per call as each element is
	// popped from this provider, allowing each mutation to leverage time-accurate runtime data.
}

// GenerateElement obtains the next element for our call sequence requested by NewSequence. If there are no elements
// left to return, this method returns an error.
func (g *CallSequenceGenerator) GenerateElement() (*calls.CallSequenceElement, error) {
	// If the call sequence length is zero, there is no work to be done.
	if g.index >= len(g.baseSequence) {
		return nil, fmt.Errorf("call sequence element could not be generated as there are no calls to make")
	}

	// Obtain our base call element
	element := g.baseSequence[g.index]

	// If it is nil, we generate an entirely new call. Otherwise, we apply pre-execution modifications.
	var err error
	if element == nil {
		element, err = g.generateNewElement()
		if err != nil {
			return nil, err
		}
	} else {
		// We have an element, if our generator set a post-call modify for this function, execute it now to modify
		// our call prior to return. This allows mutations to be applied on a per-call time frame, rather than
		// per-sequence, making use of the most recent runtime data.
		if g.modifyCallPreExecutionFunc != nil {
			err = g.modifyCallPreExecutionFunc(element)
			if err != nil {
				return nil, err
			}
		}
	}

	// Update our base sequence, advance our position, and return the processed element from this round.
	g.baseSequence[g.index] = element
	g.index++
	return element, nil
}

// generateNewElement generates a new call sequence element which targets a state changing method in a contract
// deployed to the CallSequenceGenerator's parent FuzzerWorker chain, with fuzzed call data.
// Returns the call sequence element, or an error if one was encountered.
func (g *CallSequenceGenerator) generateNewElement() (*calls.CallSequenceElement, error) {
	// Verify we have state changing methods to call
	if len(g.worker.stateChangingMethods) == 0 {
		return nil, fmt.Errorf("cannot generate fuzzed tx as there are no state changing methods to call")
	}

	// Select a random method and sender
	selectedMethod := &g.worker.stateChangingMethods[g.worker.randomProvider.Intn(len(g.worker.stateChangingMethods))]
	selectedSender := g.worker.fuzzer.senders[g.worker.randomProvider.Intn(len(g.worker.fuzzer.senders))]

	// Generate fuzzed parameters for the function call
	args := make([]any, len(selectedMethod.Method.Inputs))
	for i := 0; i < len(args); i++ {
		// Create our fuzzed parameters.
		input := selectedMethod.Method.Inputs[i]
		args[i] = valuegeneration.GenerateAbiValue(g.worker.valueGenerator, &input.Type)
	}

	// If this is a payable function, generate value to send
	var value *big.Int
	value = big.NewInt(0)
	if selectedMethod.Method.StateMutability == "payable" {
		value = g.worker.valueGenerator.GenerateInteger(false, 64)
	}

	// Create our message using the provided parameters.
	// We fill out some fields and populate the rest from our TestChain properties.
	// TODO: We likely want to make gasPrice fluctuate within some sensible range here.
	msg := calls.NewCallMessageWithAbiValueData(selectedSender, &selectedMethod.Address, 0, value, g.worker.fuzzer.config.Fuzzing.TransactionGasLimit, nil, nil, nil, &calls.CallMessageDataAbiValues{
		Method:      &selectedMethod.Method,
		InputValues: args,
	})
	msg.FillFromTestChainProperties(g.worker.chain)

	// Determine our delay values for this element
	// TODO: If we want more txs to be added together in a block, we should add a switch to make a 0 block number
	//  jump occur more often here.
	blockNumberDelay := uint64(0)
	blockTimestampDelay := uint64(0)
	if g.worker.fuzzer.config.Fuzzing.MaxBlockNumberDelay > 0 {
		blockNumberDelay = g.worker.valueGenerator.GenerateInteger(false, 64).Uint64() % (g.worker.fuzzer.config.Fuzzing.MaxBlockNumberDelay + 1)
	}
	if g.worker.fuzzer.config.Fuzzing.MaxBlockTimestampDelay > 0 {
		blockTimestampDelay = g.worker.valueGenerator.GenerateInteger(false, 64).Uint64() % (g.worker.fuzzer.config.Fuzzing.MaxBlockTimestampDelay + 1)
	}

	// For each block we jump, we need a unique time stamp for chain semantics, so if our block number jump is too small,
	// while our timestamp jump is larger, we cap it.
	if blockNumberDelay > blockTimestampDelay {
		if blockTimestampDelay == 0 {
			blockNumberDelay = 0
		} else {
			blockNumberDelay %= blockTimestampDelay
		}
	}

	// Return our call sequence element.
	return calls.NewCallSequenceElement(selectedMethod.Contract, msg, blockNumberDelay, blockTimestampDelay), nil
}
