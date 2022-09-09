package utils

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

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

// MakeDirectory creates a directory at the given path
func MakeDirectory(dirToMake string) error {
	dirInfo, err := os.Stat(dirToMake)
	if err != nil {
		// Directory does not exist, this is what we expect
		if os.IsNotExist(err) {
			// TODO: Permissions are way too much but even 666 is not working
			err = os.Mkdir(dirToMake, 0777)
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
