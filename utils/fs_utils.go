package utils

import (
	"fmt"
	"github.com/pkg/errors"
	"io"
	"os"
	"path/filepath"
)

// CreateFile will create a file at the given path and file name combination. If the path is the empty string, the
// file will be created in the current working directory
func CreateFile(path string, fileName string) (*os.File, error) {
	// By default, the path will be the name of the file
	filePath := fileName

	// Check to see if the file needs to be created in another directory or the working directory
	if path != "" {
		// Make the directory, if it does not exist already
		err := MakeDirectory(path)
		if err != nil {
			return nil, err
		}
		// Since the path is non-empty, concatenate the path with the name of the file
		filePath = filepath.Join(path, fileName)
	}

	// Create the file
	file, err := os.Create(filePath)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return file, nil
}

// CopyFile copies a file from a source path to a destination path. File permissions are retained. Returns an error
// if one occurs.
func CopyFile(sourcePath string, targetPath string) error {
	// Obtain file info for the source file
	sourceInfo, err := os.Stat(sourcePath)
	if err != nil {
		return err
	}

	// If the path refers to a directory, return an error
	if sourceInfo.IsDir() {
		return fmt.Errorf("could not copy file from '%s' to '%s' because the source path refers to a directory", sourcePath, targetPath)
	}

	// Ensure the existence of the directory we wish to copy to.
	targetDirectory := filepath.Dir(targetPath)
	err = os.MkdirAll(targetDirectory, 0777)
	if err != nil {
		return err
	}

	// Open a handle to the source file
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	// Get a handle to the created target file
	targetFile, err := os.Create(targetPath)
	if err != nil {
		return err
	}
	defer targetFile.Close()

	// Copy contents from one file handle to the other
	_, err = io.Copy(targetFile, sourceFile)
	if err != nil {
		return err
	}

	// Modify the permissions of the file
	return os.Chmod(targetPath, sourceInfo.Mode())
}

// MoveFile will move a given file from the source path to the target path. Returns an error if one occured.
func MoveFile(sourcePath string, targetPath string) error {
	// Obtain file info for the source file
	sourceInfo, err := os.Stat(sourcePath)
	if err != nil {
		return err
	}

	// If the path refers to a directory, return an error
	if sourceInfo.IsDir() {
		return fmt.Errorf("could not copy file from '%s' to '%s' because the source path refers to a directory", sourcePath, targetPath)
	}

	// Ensure the existence of the directory we wish to copy to.
	targetDirectory := filepath.Dir(targetPath)
	err = os.MkdirAll(targetDirectory, 0777)
	if err != nil {
		return err
	}

	// Move the file from the source path to the target path
	err = os.Rename(sourcePath, targetPath)
	if err != nil {
		return err
	}

	return nil
}

// GetFileNameWithoutExtension obtains a filename without the extension. This does not contain any preceding directory
// paths.
func GetFileNameWithoutExtension(filePath string) string {
	return GetFilePathWithoutExtension(filepath.Base(filePath))
}

// GetFilePathWithoutExtension obtains a file path without the extension. This retains all preceding directory paths.
func GetFilePathWithoutExtension(filePath string) string {
	return filePath[:len(filePath)-len(filepath.Ext(filePath))]
}

// MakeDirectory creates a directory at the given path, including any parent directories which do not exist.
// Returns an error, if one occurred.
func MakeDirectory(dirToMake string) error {
	dirInfo, err := os.Stat(dirToMake)
	if err != nil {
		// Directory does not exist, as expected.
		if os.IsNotExist(err) {
			// TODO: Permissions are way too much but even 666 is not working
			err = os.MkdirAll(dirToMake, 0777)
			if err != nil {
				return err
			}

			// Successfully made the directory
			return nil
		}
		// Some other sort of error, throw it
		return err
	}

	// dirToMake is a file, throw an error accordingly
	if !dirInfo.IsDir() {
		return fmt.Errorf("there is a file with the same name as %s\n", dirInfo)
	}

	// Directory already exists, good to go
	return nil
}

// CopyDirectory copies a directory from a source path to a destination path. If recursively, all subdirectories will be
// copied. If not, only files within the directory will be copied. Returns an error if one occurs.
func CopyDirectory(sourcePath string, targetPath string, recursively bool) error {
	// Obtain directory info for the source path
	sourceInfo, err := os.Stat(sourcePath)
	if err != nil {
		return err
	}

	// If the path does not refer to a directory, return an error
	if !sourceInfo.IsDir() {
		return fmt.Errorf("could not copy directory from '%s' to '%s' because the source path does not refer to a valid directory", sourcePath, targetPath)
	}

	// Create the destination folder with the given permissions
	err = os.MkdirAll(targetPath, sourceInfo.Mode())
	if err != nil {
		return err
	}

	// Read all file descriptors in the source directory
	dirEntries, err := os.ReadDir(sourcePath)
	if err != nil {
		return err
	}

	// Loop for each directory entry
	for _, dirEntry := range dirEntries {
		// Determine our source/target paths for this entry
		entSourcePath := filepath.Join(sourcePath, dirEntry.Name())
		entTargetPath := filepath.Join(targetPath, dirEntry.Name())

		if dirEntry.IsDir() {
			// If we're copying recursively, we copy directories too.
			if recursively {
				err = CopyDirectory(entSourcePath, entTargetPath, recursively)
				if err != nil {
					return err
				}
			}
		} else {
			// Copy this file
			err = CopyFile(entSourcePath, entTargetPath)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// DeleteDirectory deletes a directory at the provided path. Returns an error if one occurred.
func DeleteDirectory(directoryPath string) error {
	// Get information on the directory
	dirInfo, err := os.Stat(directoryPath)
	if err != nil {
		// If the directory does not exist, nothing needs to be done
		if os.IsNotExist(err) {
			return nil
		}
		// If any other type of error occurred, return it
		return err
	}

	// Make sure the path is a directory and not a file
	if !dirInfo.IsDir() {
		return fmt.Errorf("cannot delete directory as the provided path refers to a file")
	}

	// Delete directory and its contents
	err = os.RemoveAll(directoryPath)
	return err
}
