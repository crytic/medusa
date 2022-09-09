package simple_corpus

import (
	"encoding/json"
	"fmt"
	corpusTypes "github.com/trailofbits/medusa/fuzzing/corpus/types"
	"github.com/trailofbits/medusa/fuzzing/testnode"
	"github.com/trailofbits/medusa/utils"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
)

// SimpleCorpus implements the generic Corpus interface and is the simplest implementation of a corpus for coverage-guided fuzzing.
type SimpleCorpus struct {
	// corpusEntries is a list of SimpleCorpusEntry
	corpusEntries []*SimpleCorpusEntry
	// mutex is used to prevent races to write to corpus
	mutex sync.Mutex
}

// NewSimpleCorpus initializes a new SimpleCorpus object for the Fuzzer
func NewSimpleCorpus() *SimpleCorpus {
	return &SimpleCorpus{
		corpusEntries: []*SimpleCorpusEntry{},
	}
}

// Entries returns the list of SimpleCorpusEntry objects that are stored in the corpus
func (m *SimpleCorpus) Entries() []corpusTypes.CorpusEntry {
	var entries []corpusTypes.CorpusEntry
	for _, simpleEntry := range m.corpusEntries {
		entry := corpusTypes.CorpusEntry(simpleEntry)
		entries = append(entries, entry)
	}
	return entries
}

// AddEntry adds a SimpleCorpusEntry to the corpus and returns an error in case of an issue
func (c *SimpleCorpus) AddEntry(corpusEntry corpusTypes.CorpusEntry) error {
	// Add to corpus; we do not care about duplicates
	c.mutex.Lock() // lock
	c.corpusEntries = append(c.corpusEntries, corpusEntry.(*SimpleCorpusEntry))
	c.mutex.Unlock() // unlock
	return nil
}

// RemoveEntry removes a SimpleCorpusEntry from the corpus and returns an error in case of an issue
func (c *SimpleCorpus) RemoveEntry(entry corpusTypes.CorpusEntry) error {
	return nil
}

// RemoveEntryAt removes the SimpleCorpusEntry at index from the corpus and returns an error in case of an issue
func (c *SimpleCorpus) RemoveEntryAt(index uint64) error {
	return nil
}

// GetRandomEntry returns a random SimpleCorpusEntry from the corpus and throws an error in case of an issue
func (c *SimpleCorpus) GetRandomEntry() (corpusTypes.CorpusEntry, error) {
	return nil, nil
}

// GetEntry returns the SimpleCorpusEntry at index and returns an error in case of an issue
func (c *SimpleCorpus) GetEntry(index uint64) (corpusTypes.CorpusEntry, error) {
	return nil, nil
}

// WriteCorpusToDisk writes the SimpleCorpus to disk at writeDirectory and throws an error in case of an issue
func (c *SimpleCorpus) WriteCorpusToDisk(writeDirectory string) error {
	// Make the writeDirectory, if it does not exist
	err := utils.MakeDirectory(writeDirectory)
	if err != nil {
		return err
	}
	// Write all sequences to corpus
	for _, simpleEntry := range c.corpusEntries {
		// Get hash of the sequence
		simpleEntryHash, err := simpleEntry.Hash()
		if err != nil {
			return err
		}
		fileName := simpleEntryHash + ".json"
		// If corpus file already exists, no need to write it again
		if _, err := os.Stat(fileName); err == nil {
			continue
		}
		// Marshal the sequence
		jsonString, err := json.MarshalIndent(simpleEntry, "", " ")
		if err != nil {
			return err
		}
		// Write the byte string
		err = ioutil.WriteFile(filepath.Join(writeDirectory, fileName), jsonString, os.ModePerm)
		if err != nil {
			return fmt.Errorf("Some error here: %v\n", err)
		}

	}
	return nil
}

// ReadCorpusFromDisk reads the SimpleCorpus from disk at readDirectory/corpus and throws an error in case of an issue
func (c *SimpleCorpus) ReadCorpusFromDisk(readDirectory string) error {
	// Get .json files from the corpus/corpus subdirectory
	// Each .json file is a SimpleCorpusEntry
	matches, err := filepath.Glob(filepath.Join(readDirectory, "*.json"))
	if err != nil {
		return err
	}
	// If matches is nil, corpus (aka readDirectory) does not exist
	if matches == nil {
		return nil
	}
	// Found some matches
	for i := 0; i < len(matches); i++ {
		// Read the JSON file data
		b, err := ioutil.ReadFile(matches[i])
		if err != nil {
			return err
		}
		// Read JSON file into SimpleCorpusEntry
		var simpleEntry SimpleCorpusEntry
		err = json.Unmarshal(b, &simpleEntry)
		if err != nil {
			return err
		}
		// Add entry to corpus
		entry := corpusTypes.CorpusEntry(&simpleEntry)
		err = c.AddEntry(entry)
		if err != nil {
			return err
		}
	}

	return nil
}

// TestSequenceToCorpusEntry takes an array of TestNodeBlocks and converts it into a SimpleCorpusEntry
func (c *SimpleCorpus) TestSequenceToCorpusEntry(testNodeBlockSequence []*testnode.TestNodeBlock) (corpusTypes.CorpusEntry, error) {
	simpleEntry := NewSimpleCorpusEntry()
	for _, testNodeBlock := range testNodeBlockSequence {
		// Convert TestNodeBlock to SimpleCorpusBlock
		simpleBlock := testNodeBlockToCorpusBlock(testNodeBlock)
		// Add block to list
		err := simpleEntry.AddCorpusBlock(simpleBlock)
		if err != nil {
			return simpleEntry, err
		}
	}
	return simpleEntry, nil
}
