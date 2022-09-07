package simple_corpus

import (
	"encoding/json"
	corpusTypes "github.com/trailofbits/medusa/fuzzing/corpus/types"
	"github.com/trailofbits/medusa/fuzzing/testnode"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
)

// SimpleCorpus implements the generic Corpus interface and is the simplest implementation of a corpus for coverage-guided fuzzing.
type SimpleCorpus struct {
	// CorpusEntries is a list of SimpleCorpusEntry
	CorpusEntries []*SimpleCorpusEntry
	// Mutex is used to prevent races to write to corpus
	Mutex sync.Mutex
}

// NewSimpleCorpus initializes a new SimpleCorpus object for the Fuzzer
func NewSimpleCorpus() *SimpleCorpus {
	return &SimpleCorpus{
		CorpusEntries: []*SimpleCorpusEntry{},
	}
}

// Entries returns the list of SimpleCorpusEntry objects that are stored in the corpus
func (m *SimpleCorpus) Entries() []corpusTypes.CorpusEntry {
	var entries []corpusTypes.CorpusEntry
	for _, simpleEntry := range m.CorpusEntries {
		entry := corpusTypes.CorpusEntry(simpleEntry)
		entries = append(entries, entry)
	}
	return entries
}

// AddEntry adds a SimpleCorpusEntry to the corpus and returns an error in case of an issue
func (c *SimpleCorpus) AddEntry(corpusEntry corpusTypes.CorpusEntry) error {
	// Add to corpus; we do not care about duplicates
	c.Mutex.Lock() // lock
	c.CorpusEntries = append(c.CorpusEntries, corpusEntry.(*SimpleCorpusEntry))
	c.Mutex.Unlock() // unlock
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
	// Get current working directory
	currentDir, err := os.Getwd()
	if err != nil {
		return err
	}
	// Move to corpus/corpus subdirectory
	err = os.Chdir(filepath.Join(currentDir, writeDirectory, "/corpus"))
	if err != nil {
		return err
	}
	// Write all sequences to corpus
	for _, simpleEntry := range c.CorpusEntries {
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
		err = ioutil.WriteFile(fileName, jsonString, os.ModePerm)
		if err != nil {
			return err
		}
	}
	// Change back to original directory
	err = os.Chdir(currentDir)
	if err != nil {
		return err
	}

	return nil
}

// ReadCorpusFromDisk reads the SimpleCorpus from disk at readDirectory and throws an error in case of an issue
func (c *SimpleCorpus) ReadCorpusFromDisk(readDirectory string) error {
	// Get .json files from the corpus/corpus subdirectory
	// Each .json file is a SimpleCorpusEntry
	matches, err := filepath.Glob(filepath.Join(readDirectory, "corpus", "*.json"))
	if err != nil {
		return err
	}
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
