package simple_corpus

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"github.com/trailofbits/medusa/utils/test_utils"
	"path/filepath"
	"testing"
)

// TestSimpleCorpus_WriteAndReadCorpus first writes the corpus to disk and then reads it back from the disk
func TestSimpleCorpus_WriteAndReadCorpus(t *testing.T) {
	// Create a mock corpus
	numEntries := 5
	corpus := getMockSimpleCorpus(numEntries)
	test_utils.ExecuteInDirectory(t, t.TempDir(), func() {
		// Write to disk
		err := corpus.WriteCorpusToDisk("corpus")
		assert.Nil(t, err)
		// Ensure that there are numEntries json files
		matches, err := filepath.Glob(filepath.Join("corpus", "*.json"))
		assert.Nil(t, err)
		assert.True(t, len(matches) == numEntries, "Did not find numEntries matches")
		// Wipe corpus clean so that you can now read it in from disk
		corpus = NewSimpleCorpus()
		// Read from disk
		err = corpus.ReadCorpusFromDisk("corpus")
		// Ensure that numEntries entries are in the in-memory corpus
		assert.Nil(t, err)
		assert.True(t, len(corpus.Entries()) == numEntries, "Could not read the corpus into memory")
		// TODO: Do we need to check that corpus entries still are the same by hashing them? I feel like it is not necessary
	})
}

// TestSimpleCorpusEntry_MarshalJSONAndUnmarshalJSONAreMirrorOperations ensures that a corpus entry that is marshaled
// and then unmarshaled preserves the original data.
func TestSimpleCorpusEntry_MarshalJSONAndUnmarshalJSONAreMirrorOperations(t *testing.T) {
	// Create a mock corpus
	numEntries := 5
	corpus := getMockSimpleCorpus(numEntries)
	// For each entry, marshal it and then unmarshal the byte array
	for _, entry := range corpus.CorpusEntries {
		// Marshal the entry
		b, err := json.Marshal(entry)
		assert.Nil(t, err)
		var sameEntry SimpleCorpusEntry
		// Unmarshal byte array
		err = json.Unmarshal(b, &sameEntry)
		assert.Nil(t, err)
		// Hash entries to test equality
		entryHash, _ := entry.Hash()
		sameEntryHash, _ := sameEntry.Hash()
		assert.True(t, entryHash == sameEntryHash, "Entries are not the same")
	}
}
