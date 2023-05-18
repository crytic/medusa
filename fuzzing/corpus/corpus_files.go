package corpus

import (
	"encoding/json"
	"fmt"
	"github.com/crytic/medusa/utils"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// corpusFile represents corpus data and its state on the filesystem.
type corpusFile[T any] struct {
	// fileName describes the filename the file should be written with, in the corpusDirectory.path.
	fileName string

	// data describes an object whose data should be written to the file.
	data T

	// writtenToDisk indicates whether the corpus item has been flushed to disk yet. If this is false, it signals that
	// the data should be written or overwritten on disk.
	writtenToDisk bool
}

// corpusDirectory is a provider for corpusFile items in a given directory, offering read/write operations to
// automatically JSON serialize/deserialize items of a given type to a directory.
type corpusDirectory[T any] struct {
	// path signifies the directory to store corpusFile items within. If the path is an empty string, files
	// will not be read from, or written to disk.
	path string

	// files represents the corpusFile items stored/to be stored in the specified directory.
	files []*corpusFile[T]

	// filesLock represents a thread lock used when editing files.
	filesLock sync.Mutex
}

// newCorpusDirectory returns a new corpusDirectory with the provided directory path set.
// If the directory path is an empty string, then files will not be read from, or written to disk.
func newCorpusDirectory[T any](path string) *corpusDirectory[T] {
	return &corpusDirectory[T]{
		path:  path,
		files: make([]*corpusFile[T], 0),
	}
}

// addFile adds a given file to the file list (to later be written to the directory if a path was provided).
// If a corpusFile exists with the provided file name, it is overwritten in the list (but not yet flushed to disk).
// If a corpusFile does not exist with the provided file name, it is added.
// Returns an error, if one occurred.
func (cd *corpusDirectory[T]) addFile(fileName string, data T) error {
	// Lock to avoid concurrency issues when accessing the files list
	cd.filesLock.Lock()
	defer cd.filesLock.Unlock()

	// First we make sure this file doesn't already exist, if it does, we overwrite its data and mark it unwritten.
	lowerFileName := strings.ToLower(fileName)
	for i := 0; i < len(cd.files); i++ {
		if lowerFileName == strings.ToLower(cd.files[i].fileName) {
			cd.files[i].data = data
			cd.files[i].writtenToDisk = false
			return nil
		}
	}

	// If the file otherwise did not exist, we add it.
	cd.files = append(cd.files, &corpusFile[T]{
		fileName:      fileName,
		data:          data,
		writtenToDisk: false,
	})
	return nil
}

// removeFile removes a given file from the file list. This does not delete it from disk.
// Returns a boolean indicating if a corpusFile with the provided file name was found and removed.
func (cd *corpusDirectory[T]) removeFile(fileName string) bool {
	// Lock to avoid concurrency issues when accessing the files list
	cd.filesLock.Lock()
	defer cd.filesLock.Unlock()

	// If we find the filename, remove it from our list of files.
	lowerFileName := strings.ToLower(fileName)
	for i := 0; i < len(cd.files); i++ {
		if lowerFileName == strings.ToLower(cd.files[i].fileName) {
			cd.files = append(cd.files[:i], cd.files[i+1:]...)
			return true
		}
	}
	return false
}

// readFiles takes a provided glob pattern representing files to parse within the corpusDirectory.path.
// It parses any matching file into a corpusFile and adds it to the corpusDirectory.
// Returns an error, if one occurred.
func (cd *corpusDirectory[T]) readFiles(filePattern string) error {
	// If our directory path specified is empty, we do not read/write to disk.
	if cd.path == "" {
		return nil
	}

	// Discover all corpus files in the given directory.
	filePaths, err := filepath.Glob(filepath.Join(cd.path, filePattern))
	if err != nil {
		return err
	}

	// Refresh our files list
	cd.files = make([]*corpusFile[T], 0)

	// Loop for every file path provided
	for _, filePath := range filePaths {
		// Read the file data.
		b, err := os.ReadFile(filePath)
		if err != nil {
			return err
		}

		// Parse the call sequence data.
		var fileData T
		err = json.Unmarshal(b, &fileData)
		if err != nil {
			return err
		}

		// Add entry to corpus
		cd.files = append(cd.files, &corpusFile[T]{
			fileName:      filepath.Base(filePath),
			data:          fileData,
			writtenToDisk: true,
		})
	}
	return nil
}

// writeFiles flushes all corpusDirectory.files to disk, if they have corpusFile.writtenToDisk set as false.
// It then sets corpusFile.writtenToDisk as true for each flushed to disk.
// Returns an error, if one occurred.
func (cd *corpusDirectory[T]) writeFiles() error {
	// TODO: This can be optimized by storing/indexing unwritten sequences separately and only iterating over those.

	// If our directory path is empty, we do not write anything.
	if cd.path == "" {
		return nil
	}

	// Lock to avoid concurrency issues when accessing the files list
	cd.filesLock.Lock()
	defer cd.filesLock.Unlock()

	// Ensure the corpus directory path exists.
	err := utils.MakeDirectory(cd.path)
	if err != nil {
		return err
	}

	// For each file which does not have an assigned file path yet, we flush it to disk.
	for _, file := range cd.files {
		if !file.writtenToDisk {
			// If we don't have a filename, throw an error.
			if len(file.fileName) == 0 {
				return fmt.Errorf("failed to flush corpus item to disk as it does not have a filename")
			}

			// Determine the file path to write this to.
			filePath := filepath.Join(cd.path, file.fileName)

			// Marshal the data
			jsonEncodedData, err := json.MarshalIndent(file.data, "", " ")
			if err != nil {
				return err
			}

			// Write the JSON encoded data.
			err = os.WriteFile(filePath, jsonEncodedData, os.ModePerm)
			if err != nil {
				return fmt.Errorf("An error occurred while writing corpus data to file: %v\n", err)
			}

			// Update our written to disk status.
			file.writtenToDisk = true
		}
	}
	return nil
}
