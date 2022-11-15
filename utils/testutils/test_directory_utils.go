package testutils

import (
	"github.com/stretchr/testify/assert"
	"github.com/trailofbits/medusa/utils"
	"os"
	"path/filepath"
	"testing"
)

// CopyToTestDirectory copies files or directories from the provided filePath (relative to ./tests/contracts/) to an
// ephemeral directory used for unit tests.
func CopyToTestDirectory(t *testing.T, filePath string) string {
	// Construct our file path relative to our working directory
	cwd, err := os.Getwd()
	assert.NoError(t, err)
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
		err = utils.CopyDirectory(sourcePath, targetPath, true)
	} else {
		err = utils.CopyFile(sourcePath, targetPath)
	}
	assert.NoError(t, err)

	// Get a normalized absolute path
	targetPath, err = filepath.Abs(targetPath)
	assert.NoError(t, err)
	return targetPath
}

// ExecuteInDirectory executes the given method in a given test directory. It changes the current working directory
// to the directory specified, runs the provided method, then restores the working directory. This wraps tests so
// any file artifacts generated do not end up in the codebase directories.
func ExecuteInDirectory(t *testing.T, testPath string, method func()) {
	// Backup our old working directory
	cwd, err := os.Getwd()
	assert.NoError(t, err)

	// Check if the test path refers to a file or directory, as we'll want to change our working directory to a
	// directory path.
	testPathInfo, err := os.Stat(testPath)
	assert.NoError(t, err)

	// Ensure we obtained a directory from our path
	testDirectory := testPath
	if !testPathInfo.IsDir() {
		testDirectory = filepath.Dir(testPath)
	}

	// Change our working directory to the test directory
	err = os.Chdir(testDirectory)
	assert.NoError(t, err)

	// Execute the given method
	method()

	// Restore our working directory (we must leave the test directory or else clean up will fail post testing)
	err = os.Chdir(cwd)
	assert.NoError(t, err)
}
