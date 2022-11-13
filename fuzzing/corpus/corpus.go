package corpus

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/trailofbits/medusa/fuzzing/coverage"
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
	coverageMaps *coverage.CoverageMaps

	// callSequences is a list of call sequences that increased coverage or otherwise were found to be valuable
	// to the fuzzer.
	callSequences []*corpusFile[*CorpusCallSequence]

	// callSequencesLock provides thread synchronization to prevent concurrent access errors into
	// callSequences.
	callSequencesLock sync.Mutex

	// flushLock provides thread synchronization to prevent concurrent access errors when calling Flush.
	flushLock sync.Mutex
}

// corpusFile represents corpus data and its state on the filesystem.
type corpusFile[T any] struct {
	// filePath describes the path the file should be written to. If blank, this indicates it has not yet been written.
	filePath string

	// data describes an object whose data should be written to the file.
	data T
}

// NewCorpus initializes a new Corpus object, reading artifacts from the provided directory. If the directory refers
// to an empty path, artifacts will not be persistently stored.
func NewCorpus(corpusDirectory string) (*Corpus, error) {
	corpus := &Corpus{
		storageDirectory: corpusDirectory,
		coverageMaps:     coverage.NewCoverageMaps(),
		callSequences:    make([]*corpusFile[*CorpusCallSequence], 0),
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
			corpus.callSequences = append(corpus.callSequences, &corpusFile[*CorpusCallSequence]{
				filePath: filePath,
				data:     &entry,
			})
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
func (c *Corpus) CoverageMaps() *coverage.CoverageMaps {
	return c.coverageMaps
}

// CallSequenceCount returns the count of call sequences recorded in the corpus.
func (c *Corpus) CallSequenceCount() int {
	return len(c.callSequences)
}

// AddCallSequence adds a CorpusCallSequence to the corpus and returns an error in case of an issue
func (c *Corpus) AddCallSequence(corpusEntry CorpusCallSequence) error {
	// Update our sequences with the new entry.
	c.callSequencesLock.Lock()
	c.callSequences = append(c.callSequences, &corpusFile[*CorpusCallSequence]{
		filePath: "",
		data:     &corpusEntry,
	})
	c.callSequencesLock.Unlock()
	return nil
}

// CallSequences returns all the CorpusCallSequence known to the corpus. This should not be called frequently,
// as the slice returned by this method is computed each time it is called.
func (c *Corpus) CallSequences() []*CorpusCallSequence {
	return utils.SliceSelect(c.callSequences, func(file *corpusFile[*CorpusCallSequence]) *CorpusCallSequence {
		return file.data
	})
}

// Flush writes corpus changes to disk. Returns an error if one occurs.
func (c *Corpus) Flush() error {
	// If our corpus directory is empty, it indicates we do not want to write corpus artifacts to persistent storage.
	if c.storageDirectory == "" {
		return nil
	}

	// Lock while flushing the corpus items to avoid concurrent access issues.
	c.flushLock.Lock()
	defer c.flushLock.Unlock()
	c.callSequencesLock.Lock()
	defer c.callSequencesLock.Unlock()

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
	// TODO: This can be optimized by storing/indexing unwritten sequences separately and only iterating over those.
	for _, sequenceFile := range c.callSequences {
		if sequenceFile.filePath == "" {
			// Determine the file path to write this to.
			sequenceFile.filePath = filepath.Join(c.CallSequencesDirectory(), uuid.New().String()+".json")

			// Marshal the call sequence
			jsonEncodedData, err := json.MarshalIndent(sequenceFile.data, "", " ")
			if err != nil {
				return err
			}

			// Write the JSON encoded data.
			err = ioutil.WriteFile(sequenceFile.filePath, jsonEncodedData, os.ModePerm)
			if err != nil {
				return fmt.Errorf("An error occurred while writing call sequence to disk: %v\n", err)
			}
		}
	}
	return nil
}
