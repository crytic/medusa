package fuzzing

import (
	"math/big"
	"math/rand"

	"github.com/crytic/medusa/fuzzing/calls"
	"github.com/crytic/medusa/fuzzing/valuegeneration"
)

// removeReverts removes all reverted transactions (except the last). Uses existing
// execution results attached to CallSequenceElement (no re-execution needed).
//
// This function processes all transactions except the last one. Reverted transactions
// are simply removed without preserving their delays, as they don't change contract
// state. Note: This means timing-related bugs (that depend on specific block numbers
// or timestamps) may not be properly shrunk if reverted transactions were needed to
// advance time. A future enhancement would be to implement NoCall transactions that
// preserve timing without executing contract code, which would improve shrinking for
// time-dependent bugs.
//
// The last transaction is always kept, even if it reverted, as it may be the failing
// transaction that triggered shrinking.
//
// Returns a new CallSequence with reverted transactions removed.
func removeReverts(sequence calls.CallSequence) calls.CallSequence {
	if len(sequence) <= 1 {
		return sequence
	}

	result := make(calls.CallSequence, 0, len(sequence))

	// Process all-but-last transactions
	for i := 0; i < len(sequence)-1; i++ {
		element := sequence[i]

		// Check if this transaction reverted using existing execution results
		reverted := false
		if element.ChainReference != nil {
			msgResult := element.ChainReference.MessageResults()
			reverted = msgResult.ExecutionResult.Err != nil
		}

		if !reverted {
			// Transaction succeeded - keep it
			result = append(result, element)
		}
		// If reverted, simply skip it (don't preserve delays)
	}

	// Always keep last transaction
	result = append(result, sequence[len(sequence)-1])

	return result
}

// shortenSequence removes one random transaction from the sequence (with delay transfer).
// The delays from the removed transaction are transferred to the previous transaction
// to preserve timing behavior.
//
// If the sequence has only 1 or 0 transactions, it is returned unchanged.
// If the removed transaction is at index 0, no delay transfer occurs.
// Otherwise, delays are added to the transaction at index removeIndex-1.
//
// Returns a new CallSequence with one transaction removed.
func shortenSequence(txs calls.CallSequence, randProvider *rand.Rand) calls.CallSequence {
	if len(txs) <= 1 {
		return txs
	}

	// Pick random index to remove
	removeIndex := randProvider.Intn(len(txs))
	removedTx := txs[removeIndex]

	// Create result without the removed transaction
	result := make(calls.CallSequence, 0, len(txs)-1)
	result = append(result, txs[:removeIndex]...)
	result = append(result, txs[removeIndex+1:]...)

	// Transfer delays to previous transaction if it exists
	if removeIndex > 0 && removeIndex <= len(result) {
		prev, _ := result[removeIndex-1].Clone()
		prev.BlockNumberDelay += removedTx.BlockNumberDelay
		prev.BlockTimestampDelay += removedTx.BlockTimestampDelay
		result[removeIndex-1] = prev
	}

	return result
}

// shrinkAllTransactions shrinks all transactions in the sequence.
// For each transaction, randomly picks one aspect to shrink using shrinkOneTransactionAspect.
// Possible aspects include: arguments, value, gasprice, or delays.
//
// Returns a new CallSequence with all transactions shrunk.
func shrinkAllTransactions(
	txs calls.CallSequence,
	randomProvider *rand.Rand,
	valueMutator valuegeneration.ValueMutator,
	valueGenerator valuegeneration.ValueGenerator,
) calls.CallSequence {
	result := make(calls.CallSequence, len(txs))

	for i, tx := range txs {
		// For each transaction, randomly pick one aspect to shrink
		result[i] = shrinkOneTransactionAspect(tx, randomProvider, valueMutator, valueGenerator)
	}

	return result
}

// shrinkOneTransactionAspect randomly picks one aspect of the transaction to shrink.
// Possible aspects: arguments (all at once), value, gasprice, or delay.
//
// This function builds a list of possible shrinking operations and picks one uniformly
// at random. Each operation shrinks a different aspect of the transaction.
//
// Returns a new CallSequenceElement with one aspect shrunk.
func shrinkOneTransactionAspect(
	element *calls.CallSequenceElement,
	randProvider *rand.Rand,
	valueMutator valuegeneration.ValueMutator,
	valueGenerator valuegeneration.ValueGenerator,
) *calls.CallSequenceElement {
	// Build list of possible shrinking operations
	operations := []func(*calls.CallSequenceElement, *rand.Rand, valuegeneration.ValueMutator, valuegeneration.ValueGenerator) *calls.CallSequenceElement{
		shrinkAllArguments,
		shrinkValue,
		shrinkGasPrice,
		shrinkDelay,
	}

	// Pick one uniformly at random
	operation := operations[randProvider.Intn(len(operations))]
	return operation(element, randProvider, valueMutator, valueGenerator)
}

// shrinkAllArguments shrinks all function arguments using existing ShrinkingValueMutator.
// This preserves the existing logic from lines 600-607 of fuzzer_worker.go.
//
// Returns a cloned element with all arguments mutated, or the original if no ABI values exist.
func shrinkAllArguments(
	element *calls.CallSequenceElement,
	_ *rand.Rand,
	valueMutator valuegeneration.ValueMutator,
	valueGenerator valuegeneration.ValueGenerator,
) *calls.CallSequenceElement {
	// Skip if no ABI values (e.g., fallback/receive functions)
	if element.Call.DataAbiValues == nil {
		return element
	}

	cloned, _ := element.Clone()
	abiValuesMsgData := cloned.Call.DataAbiValues

	// Mutate all arguments (same as existing implementation)
	for j := 0; j < len(abiValuesMsgData.InputValues); j++ {
		mutatedInput, err := valuegeneration.MutateAbiValue(
			valueGenerator,
			valueMutator,
			&abiValuesMsgData.Method.Inputs[j].Type,
			abiValuesMsgData.InputValues[j],
		)
		if err != nil {
			// If mutation fails, keep the original value
			continue
		}
		abiValuesMsgData.InputValues[j] = mutatedInput
	}

	// Re-encode the message's calldata
	cloned.Call.WithDataAbiValues(abiValuesMsgData)

	return cloned
}

// shrinkValue shrinks the ETH value toward zero with 50% bias for zero.
//
// Returns a cloned element with shrunk value, or the original if value is already zero.
func shrinkValue(
	element *calls.CallSequenceElement,
	randProvider *rand.Rand,
	_ valuegeneration.ValueMutator,
	_ valuegeneration.ValueGenerator,
) *calls.CallSequenceElement {
	if element.Call.Value.Sign() == 0 {
		return element
	}

	cloned, _ := element.Clone()
	cloned.Call.Value = lower(element.Call.Value, randProvider)
	return cloned
}

// shrinkGasPrice shrinks the gas price toward zero with 50% bias for zero.
//
// Returns a cloned element with shrunk gas price, or the original if gas price is already zero.
func shrinkGasPrice(
	element *calls.CallSequenceElement,
	randProvider *rand.Rand,
	_ valuegeneration.ValueMutator,
	_ valuegeneration.ValueGenerator,
) *calls.CallSequenceElement {
	if element.Call.GasPrice.Sign() == 0 {
		return element
	}

	cloned, _ := element.Clone()
	cloned.Call.GasPrice = lower(element.Call.GasPrice, randProvider)
	return cloned
}

// shrinkDelay shrinks block/time delays toward zero with leveling.
// Leveling means: if either delay becomes 0, make both 0.
//
// Uses three random strategies:
//   - Lower only block delay
//   - Lower only time delay
//   - Lower both delays
//
// Returns a cloned element with shrunk delays, or the original if both delays are already zero.
func shrinkDelay(
	element *calls.CallSequenceElement,
	randProvider *rand.Rand,
	_ valuegeneration.ValueMutator,
	_ valuegeneration.ValueGenerator,
) *calls.CallSequenceElement {
	if element.BlockNumberDelay == 0 && element.BlockTimestampDelay == 0 {
		return element
	}

	cloned, _ := element.Clone()

	// Three strategies, pick one uniformly
	strategy := randProvider.Intn(3)

	switch strategy {
	case 0:
		// Lower only block delay
		cloned.BlockNumberDelay = lowerUint64(element.BlockNumberDelay, randProvider)
	case 1:
		// Lower only time delay
		cloned.BlockTimestampDelay = lowerUint64(element.BlockTimestampDelay, randProvider)
	case 2:
		// Lower both
		cloned.BlockNumberDelay = lowerUint64(element.BlockNumberDelay, randProvider)
		cloned.BlockTimestampDelay = lowerUint64(element.BlockTimestampDelay, randProvider)
	}

	// Apply leveling: if either is 0, make both 0
	if cloned.BlockNumberDelay == 0 || cloned.BlockTimestampDelay == 0 {
		cloned.BlockNumberDelay = 0
		cloned.BlockTimestampDelay = 0
	}

	return cloned
}

// lower shrinks a big.Int toward zero with 50% bias for zero.
// Returns 0 half the time, and a random value between 0 and x the other half.
func lower(x *big.Int, randProvider *rand.Rand) *big.Int {
	if x.Sign() == 0 {
		return big.NewInt(0)
	}

	// 50% chance to use 0, 50% chance to use random value between 0 and x
	if randProvider.Float32() < 0.5 {
		return big.NewInt(0)
	}

	// Generate random value between 0 and x
	randomVal := new(big.Int).Rand(randProvider, x)
	return randomVal
}

// lowerUint64 shrinks a uint64 toward zero with 50% bias for zero.
// Returns 0 half the time, and a random value between 0 and x the other half.
func lowerUint64(x uint64, randProvider *rand.Rand) uint64 {
	if x == 0 {
		return 0
	}

	// 50% chance to use 0, 50% chance to use random value between 0 and x
	if randProvider.Float32() < 0.5 {
		return 0
	}

	return uint64(randProvider.Int63n(int64(x)))
}

// canShrinkTransaction returns true if a single transaction has potential for shrinking.
// A transaction can be shrunk if it has:
// - Non-zero value
// - Non-zero gas price
// - Non-zero delays (block number or timestamp)
// - ABI arguments that could potentially be shrunk
//
// Returns false if the transaction is already minimal.
func canShrinkTransaction(element *calls.CallSequenceElement) bool {
	// Check if value can be shrunk
	if element.Call.Value != nil && element.Call.Value.Sign() != 0 {
		return true
	}

	// Check if gas price can be shrunk
	if element.Call.GasPrice != nil && element.Call.GasPrice.Sign() != 0 {
		return true
	}

	// Check if delays can be shrunk
	if element.BlockNumberDelay != 0 || element.BlockTimestampDelay != 0 {
		return true
	}

	// Check if arguments can be shrunk (conservative check)
	// If DataAbiValues exists, we assume arguments may be shrinkable
	if element.Call.DataAbiValues != nil {
		return true
	}

	// Transaction is already minimal
	return false
}

// canShrinkFurther returns true if the sequence has potential for further shrinking.
// A sequence can be shrunk if:
// - It has more than 1 transaction, OR
// - Any transaction can be shrunk (checked via canShrinkTransaction)
//
// This function is used as an early exit check in the shrinking loop to avoid
// unnecessary shrinking attempts when the sequence is already minimal.
func canShrinkFurther(sequence calls.CallSequence) bool {
	// If we have more than one transaction, we can always try to remove one
	if len(sequence) > 1 {
		return true
	}

	// If we have exactly one transaction, check if it can be shrunk
	if len(sequence) == 1 {
		return canShrinkTransaction(sequence[0])
	}

	// Empty sequence cannot be shrunk further
	return false
}
