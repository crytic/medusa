package randomutils

import (
	"math/big"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestWeightedRandomChooserConcurrentAccess exercises a WeightedRandomChooser from several goroutines at once, the
// way fuzzer workers share the corpus mutation chooser. Run it with the race detector (`go test -race`) to verify
// that ChoiceCount and Choose do not read the choice list or total weight while another goroutine adds choices.
func TestWeightedRandomChooserConcurrentAccess(t *testing.T) {
	// Create a chooser with a single initial choice so it is never empty.
	chooser := NewWeightedRandomChooser[int]()
	chooser.AddChoices(NewWeightedRandomChoice(0, big.NewInt(1)))

	// Spawn several goroutines which add choices while counting and choosing from the same chooser.
	const workerCount = 8
	const choicesPerWorker = 100
	var workers sync.WaitGroup
	for worker := 0; worker < workerCount; worker++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for i := 0; i < choicesPerWorker; i++ {
				// Add a new choice, then read the choice count and select a choice concurrently with other workers.
				chooser.AddChoices(NewWeightedRandomChoice(i, big.NewInt(1)))
				assert.NotZero(t, chooser.ChoiceCount())
				value, err := chooser.Choose()
				assert.NoError(t, err)
				if assert.NotNil(t, value) {
					assert.True(t, *value >= 0 && *value < choicesPerWorker)
				}
			}
		}()
	}
	workers.Wait()

	// Every addition should be accounted for once all goroutines have finished.
	assert.EqualValues(t, workerCount*choicesPerWorker+1, chooser.ChoiceCount())
}
