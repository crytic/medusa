package corpus

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	fuzzerTypes "github.com/trailofbits/medusa/fuzzing/coverage"
	"github.com/trailofbits/medusa/utils"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
)

// Corpus describes an archive of fuzzer-generated artifacts used to further fuzzing efforts. These artifacts are
// reusable across fuzzer runs. Changes to the fuzzer/chain configuration or definitions within smart contracts
// may create incompatibilities with corpus items.
type Corpus struct {
	// storageDirectory describes the directory to save corpus callSequencesByFilePath within.
	storageDirectory string

	// coverageMaps describes the total code coverage known to be achieved across all runs.
	coverageMaps *fuzzerTypes.CoverageMaps

	// callSequencesByFilePath is a mapping of file paths to corpus call sequences stored at those paths.
	callSequencesByFilePath map[string]*CorpusCallSequence

	// corpusEntriesLock provides thread synchronization used to prevent concurrent access errors into callSequencesByFilePath.
	corpusEntriesLock sync.Mutex
}

// NewCorpus initializes a new Corpus object, reading artifacts from the provided directory. If the directory refers
// to an empty path, artifacts will not be persistently stored.
func NewCorpus(corpusDirectory string) (*Corpus, error) {
	corpus := &Corpus{
		storageDirectory:        corpusDirectory,
		coverageMaps:            fuzzerTypes.NewCoverageMaps(),
		callSequencesByFilePath: make(map[string]*CorpusCallSequence),
	}

	// If we have a corpus directory set, parse it.
	if corpus.storageDirectory != "" {
		// Read all call sequences discovered in the relevant corpus directory.
		matches, err := filepath.Glob(filepath.Join(corpus.CallSequencesDirectory(), "*.json"))
		if err != nil {
			return nil, err
		}
		for i := 0; i < len(matches); i++ {
			// Alias our file path.
			filePath := matches[i]

			// Read the call sequence data.
			b, err := ioutil.ReadFile(filePath)
			if err != nil {
				return nil, err
			}

			// Parse the call sequence data.
			var entry CorpusCallSequence
			err = json.Unmarshal(b, &entry)
			if err != nil {
				return nil, err
			}

			// Add entry to corpus
			corpus.callSequencesByFilePath[filePath] = &entry
		}
	}
	return corpus, nil
}

// StorageDirectory returns the root directory path of the corpus. If this is empty, it indicates persistent storage
// will not be used.
func (c *Corpus) StorageDirectory() string {
	return c.storageDirectory
}

// CallSequencesDirectory returns the directory path where coverage increasing call sequences should be stored.
// This is a subdirectory of StorageDirectory. If StorageDirectory is empty, this is as well, indicating persistent
// storage will not be used.
func (c *Corpus) CallSequencesDirectory() string {
	if c.storageDirectory == "" {
		return ""
	}

	return filepath.Join(c.StorageDirectory(), "call_sequences")
}

// CoverageMaps returns the total coverage collected across all runs.
func (c *Corpus) CoverageMaps() *fuzzerTypes.CoverageMaps {
	return c.coverageMaps
}

// CallSequenceCount returns the count of call sequences recorded by the corpus which increased coverage.
func (c *Corpus) CallSequenceCount() int {
	return len(c.callSequencesByFilePath)
}

// AddCallSequence adds a CorpusCallSequence to the corpus and returns an error in case of an issue
func (c *Corpus) AddCallSequence(corpusEntry CorpusCallSequence) error {
	// Determine the filepath to write our corpus entry to.

	// Update our map with the new entry. We generate a random UUID until we have a unique one for the filename.
	c.corpusEntriesLock.Lock()
	for {
		filePath := filepath.Join(c.CallSequencesDirectory(), uuid.New().String()+".json")
		if _, existsAlready := c.callSequencesByFilePath[filePath]; existsAlready {
			continue
		}
		c.callSequencesByFilePath[filePath] = &corpusEntry
		break
	}
	c.corpusEntriesLock.Unlock()
	return nil
}

// Flush writes corpus changes to disk. Returns an error if one occurs.
func (c *Corpus) Flush() error {
	// If our corpus directory is empty, it indicates we do not want to write corpus artifacts to persistent storage.
	if c.storageDirectory == "" {
		return nil
	}

	// Ensure the corpus directories exists.
	err := utils.MakeDirectory(c.storageDirectory)
	if err != nil {
		return err
	}
	err = utils.MakeDirectory(c.CallSequencesDirectory())
	if err != nil {
		return err
	}

	// Write all call sequences to disk
	for filePath, simpleEntry := range c.callSequencesByFilePath {
		// If call sequence file already exists, no need to write it again
		if _, err := os.Stat(filePath); err == nil {
			continue
		}

		// Marshal the call sequence
		jsonEncodedData, err := json.MarshalIndent(simpleEntry, "", " ")
		if err != nil {
			return err
		}

		// Write the JSON encoded data.
		err = ioutil.WriteFile(filePath, jsonEncodedData, os.ModePerm)
		if err != nil {
			return fmt.Errorf("An error occurred while writing call sequence to disk: %v\n", err)
		}
	}
	return nil
}
