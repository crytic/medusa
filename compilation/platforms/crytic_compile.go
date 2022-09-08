package platforms

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/trailofbits/medusa/compilation/types"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
)

// CryticCompilationConfig represents the various configuration options that can be provided by the user
// while using the `crytic-compile` platform
type CryticCompilationConfig struct {
	// Target is the object that is being compiled. It can be a single `.sol` file or a whole directory
	Target string `json:"target"`
	// SolcVersion is the version of `solc` that will be installed prior to compiling with crytic-compile. If empty,
	// no special version is installed prior to compilation.
	SolcVersion string `json:"solcVersion,omitempty"`
	// BuildDirectory is the location where medusa will search for build artifacts. By default, medusa will look in
	// `./crytic-export`
	BuildDirectory string `json:"buildDirectory,omitempty"`
	// Args are additional arguments that can be provided to `crytic-compile`
	Args []string `json:"args"`
}

// Platform returns the platform type
func (c *CryticCompilationConfig) Platform() string {
	return "crytic-compile"
}

// NewCryticCompilationConfig returns the default configuration options while using `crytic-compile`
func NewCryticCompilationConfig(target string) *CryticCompilationConfig {
	return &CryticCompilationConfig{
		Target:         target,
		BuildDirectory: "",
		Args:           []string{},
		SolcVersion:    "",
	}
}

// ValidateArgs ensures that the additional arguments provided to `crytic-compile` do not contain the `--export-format`
// or the `--export-dir` arguments. This is because `--export-format` has to be `standard` for the `crytic-compile`
// integration to work and CryticCompilationConfig.BuildDirectory option is equivalent to `--export-dir`
func (c *CryticCompilationConfig) ValidateArgs() error {
	// If --export-format or --export-dir are specified in c.Args, throw an error
	for _, arg := range c.Args {
		if arg == "--export-format" {
			return errors.New("do not specify `--export-format` as an argument since the standard export format is required by medusa")
		}
		if arg == "--export-dir" {
			return errors.New("do not specify `--export-dir` as an argument, use the BuildDirectory config variable instead")
		}
	}
	return nil
}

// DeleteBuildDirectory will delete an old crytic-export (or BuildDirectory) so that the new artifacts can be analyzed.
// This prevents cross-contamination of old and new compilation artifacts
func (c *CryticCompilationConfig) DeleteBuildDirectory() error {
	var dirToDelete string
	if c.BuildDirectory == "" {
		dirToDelete = "crytic-export"
	} else {
		dirToDelete = c.BuildDirectory
	}
	dirInfo, err := os.Stat(dirToDelete)

	if err != nil {
		// Directory does not exist, we are good and don't have to delete anything
		if os.IsNotExist(err) {
			return nil
		}
		// If any other type of error, return it
		return err
	}

	// Arguably unnecessary check to make sure dirToDelete is a directory and not a file
	if !dirInfo.IsDir() {
		return fmt.Errorf("Build directory is a file, not a directory")
	}

	// Delete directory and its contents
	err = os.RemoveAll(dirToDelete)
	if err != nil {
		return err
	}
	return nil
}

// GetWorkingDirectoryAndArgs sets the arguments that will be provided to crytic-compile, based on the configuration,
// and identifies the current working directory based on c.Target.
func (c *CryticCompilationConfig) GetWorkingDirectoryAndArgs() (string, []string, error) {
	// Get information on c.Target
	// Using pathInfo to figure out if c.Target is a directory or not
	pathInfo, err := os.Stat(c.Target)
	if err != nil {
		return "", nil, fmt.Errorf("error while trying to get information on directory %s\n", c.Target)
	}
	// args are the arguments to crytic-compile
	var args []string
	// workingDirectory is the current working dir
	var workingDirectory string
	if c.Target == "." {
		// If c.Target is '.', then the current dir is the workingDirectory and args should use the '.'
		// ignoring errors from os.Getwd()
		workingDirectory, _ = os.Getwd()
		args = append([]string{".", "--export-format", "standard"})
	} else if !pathInfo.IsDir() {
		// If c.Target is a single file, then the workingDirectory is the parent dir of the file and args should use c.Target
		workingDirectory = filepath.Dir(c.Target)
		args = append([]string{c.Target, "--export-format", "standard"})
	} else {
		// c.Target is a directory, so the workingDirectory is the current dir and args should use c.Target
		workingDirectory = c.Target
		args = append([]string{c.Target, "--export-format", "standard"})
	}
	// Add --export-dir option if BuildDirectory is specified
	if c.BuildDirectory != "" {
		args = append(args, []string{"--export-dir", c.BuildDirectory}...)
	}
	// Add remaining args
	args = append(args, c.Args...)
	return workingDirectory, args, nil
}

// Compile uses the CryticCompilationConfig provided to compile a given target, parse the artifacts, and then
// create a list of types.Compilation.
func (c *CryticCompilationConfig) Compile() ([]types.Compilation, string, error) {
	// Delete buildDirectory if it already exists
	err := c.DeleteBuildDirectory()
	if err != nil {
		return nil, "", err
	}
	// Validate args to make sure --export-format and --export-dir are not specified
	err = c.ValidateArgs()
	if err != nil {
		return nil, "", err
	}

	// Need to figure out the args to provide crytic-compile and the workingDirectory
	workingDirectory, args, err := c.GetWorkingDirectoryAndArgs()

	// Get main command and set working directory
	cmd := exec.Command("crytic-compile", args...)
	// Set working directory
	cmd.Dir = workingDirectory

	// Install a specific `solc` version if requested in the config
	if c.SolcVersion != "" {
		err := exec.Command("solc-select", "install", c.SolcVersion).Run()
		if err != nil {
			return nil, "", fmt.Errorf("error while executing solc-select:\n\nERROR: %s\n", err.Error())
		}
	}

	// Run crytic-compile
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, "", fmt.Errorf("error while executing crytic-compile:\nOUTPUT:\n%s\nERROR: %s\n", string(out), err.Error())
	}

	// Get build directory
	buildDirectory := c.BuildDirectory
	// Default build directory is crytic-export
	if buildDirectory == "" {
		buildDirectory = "crytic-export"
	}
	matches, err := filepath.Glob(filepath.Join(buildDirectory, "*.json"))
	if err != nil {
		return nil, "", err
	}

	// Create a compilation list for a list of compilation units.
	var compilationList []types.Compilation

	// Loop through each .json file for compilation units.
	for i := 0; i < len(matches); i++ {
		// Read the compiled JSON file data
		b, err := ioutil.ReadFile(matches[i])
		if err != nil {
			return nil, "", err
		}
		// Parse the JSON
		var compiledJson map[string]interface{}
		err = json.Unmarshal(b, &compiledJson)
		if err != nil {
			return nil, "", err
		}
		// Index into "compilation_units" key
		compilationUnits, ok := compiledJson["compilation_units"]
		if !ok {
			// If our json file does not have any compilation units, it is not a file of interest
			continue
		}
		// Create a mapping between key (filename) and value (contract and ast information) each compilation unit
		compilationMap, ok := compilationUnits.(map[string]interface{})
		if !ok {
			return nil, "", fmt.Errorf("compilationUnits is not in the map[string]interface{} format: %s\n", compilationUnits)
		}
		// Iterate through compilationUnits
		for _, compilationUnit := range compilationMap {
			// Create a compilation object that will store the contracts and asts for a single compilation unit
			compilation := types.NewCompilation()
			// Create mapping between key (compiler / asts / contracts) and associated values
			compilationUnitMap, ok := compilationUnit.(map[string]interface{})
			if !ok {
				return nil, "", fmt.Errorf("compilationUnit is not in the map[string]interface{} format: %s\n", compilationUnit)
			}
			// Create mapping between each file in compilation unit and associated Ast
			AstMap := compilationUnitMap["asts"].(map[string]interface{})
			// Create mapping between key (file name) and value (associated contracts in that file)
			contractsMap, ok := compilationUnitMap["contracts"].(map[string]interface{})
			if !ok {
				return nil, "", fmt.Errorf("cannot find 'contracts' key in compilationUnitMap: %s\n", compilationUnitMap)
			}
			// Iterate through each contract FILE (note that each FILE might have more than one contract)
			for _, contractsData := range contractsMap {
				// Create mapping between all contracts in a file (key) to it's data (abi, etc.)
				contractMap, ok := contractsData.(map[string]interface{})
				if !ok {
					return nil, "", fmt.Errorf("contractsData is not in the map[string]interface{} format: %s\n", contractsData)
				}
				// Iterate through each contract
				for contractName, contractData := range contractMap {
					// Create mapping between contract details (abi, bytecode) to actual values
					contractDataMap, ok := contractData.(map[string]interface{})
					if !ok {
						return nil, "", fmt.Errorf("contractData is not in the map[string]interface{} format: %s\n", contractData)
					}
					// Create mapping between "filenames" (key) associated with the contract and the various filename
					// types (absolute, relative, short, long)
					fileMap, ok := contractDataMap["filenames"].(map[string]interface{})
					if !ok {
						return nil, "", fmt.Errorf("cannot find 'filenames' key in contractDataMap: %s\n", contractDataMap)
					}
					// Create unique source path which is going to be absolute path
					sourcePath := fmt.Sprintf("%v", fileMap["absolute"])
					// Get ABI
					contractAbi, err := types.InterfaceToABI(contractDataMap["abi"])
					if err != nil {
						return nil, "", fmt.Errorf("Unable to parse ABI: %s\n", contractDataMap["abi"])
					}
					// Check if sourcePath has already been set (note that a sourcePath (i.e., file) can have more
					// than one contract)
					// sourcePath is also the key for the AstMap
					if _, ok := compilation.Sources[sourcePath]; !ok {
						compilation.Sources[sourcePath] = types.CompiledSource{
							Ast:       AstMap[sourcePath],
							Contracts: make(map[string]types.CompiledContract),
						}
					}
					// Add contract details
					compilation.Sources[sourcePath].Contracts[contractName] = types.CompiledContract{
						Abi:             *contractAbi,
						RuntimeBytecode: fmt.Sprintf("%v", contractDataMap["bin-runtime"]),
						InitBytecode:    fmt.Sprintf("%v", contractDataMap["bin"]),
						SrcMapsInit:     fmt.Sprintf("%v", contractDataMap["srcmap"]),
						SrcMapsRuntime:  fmt.Sprintf("%v", contractDataMap["srcmap-runtime"]),
					}
				}
			}
			// Append compilation object to compilationList
			compilationList = append(compilationList, *compilation)
		}
	}
	// Return the compilationList
	return compilationList, string(out), nil
}
