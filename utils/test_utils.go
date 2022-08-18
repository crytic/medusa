package utils

import (
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"testing"
)

// CopyToTestDirectory copies files or directories from the provided filePath (relative to ./tests/contracts/) to an
// ephemeral directory used for unit tests.
func CopyToTestDirectory(t *testing.T, filePath string) string {
	// Construct our file path relative to our working directory
	cwd, err := os.Getwd()
	assert.Nil(t, err)
	sourcePath := filepath.Join(cwd, filePath)

	// Verify the file path exists
	sourcePathInfo, err := os.Stat(sourcePath)
	assert.False(t, os.IsNotExist(err))
	assert.NotNil(t, sourcePathInfo)

	// Obtain an isolated test directory path.
	targetDirectory := filepath.Join(t.TempDir(), "medusaTest")
	targetPath := filepath.Join(targetDirectory, sourcePathInfo.Name())

	// Copy our source to the target destination
	if sourcePathInfo.IsDir() {
		err = CopyDirectory(sourcePath, targetPath, true)
	} else {
		err = CopyFile(sourcePath, targetPath)
	}
	assert.Nil(t, err)

	// Get a normalized absolute path
	targetPath, err = filepath.Abs(targetPath)
	assert.Nil(t, err)
	return targetPath
}
