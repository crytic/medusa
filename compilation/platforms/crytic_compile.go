package platforms

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/trailofbits/medusa/compilation/types"
	"github.com/trailofbits/medusa/utils"
	"io/ioutil"
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
	SolcVersion string `json:"solcVersion"`

	// ExportDirectory is the location to search for exported build artifacts. By default, we look in `./crytic-export`
	ExportDirectory string `json:"exportDirectory"`

	// Args are additional arguments that can be provided to `crytic-compile`
	Args []string `json:"args"`
}

// Platform returns the platform type
func (c *CryticCompilationConfig) Platform() string {
	return "crytic-compile"
}

// GetTarget returns the target for compilation
func (c *CryticCompilationConfig) GetTarget() string {
	return c.Target
}

// SetTarget sets the new target for compilation
func (c *CryticCompilationConfig) SetTarget(newTarget string) {
	c.Target = newTarget
}

// NewCryticCompilationConfig returns the default configuration options while using `crytic-compile`
func NewCryticCompilationConfig(target string) *CryticCompilationConfig {
	return &CryticCompilationConfig{
		Target:          target,
		ExportDirectory: "",
		Args:            []string{},
		SolcVersion:     "",
	}
}

// validateArgs ensures that the additional arguments provided to `crytic-compile` do not contain the `--export-format`
// or the `--export-dir` arguments. This is because `--export-format` has to be `standard` for the `crytic-compile`
// integration to work and CryticCompilationConfig.BuildDirectory option is equivalent to `--export-dir`
func (c *CryticCompilationConfig) validateArgs() error {
	// If --export-format or --export-dir are specified in c.Args, throw an error
	for _, arg := range c.Args {
		if arg == "--export-format" {
			return errors.New("do not specify `--export-format` within crytic-compile arguments as the standard export format is always used")
		}
		if arg == "--export-dir" {
			return errors.New("do not specify `--export-dir` as an argument, use the BuildDirectory config variable instead")
		}
	}
	return nil
}

// getArgs returns the arguments to be provided to crytic-compile during compilation, or an error if one occurs.
func (c *CryticCompilationConfig) getArgs() ([]string, error) {
	// By default we export in solc-standard mode.
	args := []string{c.Target, "--export-format", "standard"}

	// Add --export-dir option if ExportDirectory is specified
	if c.ExportDirectory != "" {
		args = append(args, "--export-dir", c.ExportDirectory)
	}

	// Add remaining args
	args = append(args, c.Args...)
	return args, nil
}

// Compile uses the CryticCompilationConfig provided to compile a given target, parse the artifacts, and then
// create a list of types.Compilation.
func (c *CryticCompilationConfig) Compile() ([]types.Compilation, string, error) {
	// Resolve our export directory and delete it if already exists
	exportDirectory := c.ExportDirectory
	if exportDirectory == "" {
		exportDirectory = "crytic-export"
	}
	err := utils.DeleteDirectory(exportDirectory)
	if err != nil {
		return nil, "", err
	}

	// Validate args to make sure --export-format and --export-dir are not specified
	err = c.validateArgs()
	if err != nil {
		return nil, "", err
	}

	// Fetch the arguments to invoke crytic-compile with
	args, err := c.getArgs()
	if err != nil {
		return nil, "", err
	}

	// Get main command and set working directory
	cmd := exec.Command("crytic-compile", args...)

	// Install a specific `solc` version if requested in the config
	if c.SolcVersion != "" {
		out, err := exec.Command("solc-select", "install", c.SolcVersion).CombinedOutput()
		if err != nil {
			return nil, "", fmt.Errorf("error while executing `solc-select install`:\nOUTPUT:\n%s\nERROR: %s\n", string(out), err.Error())
		}
		out, err = exec.Command("solc-select", "use", c.SolcVersion).CombinedOutput()
		if err != nil {
			return nil, "", fmt.Errorf("error while executing `solc-select use`:\nOUTPUT:\n%s\nERROR: %s\n", string(out), err.Error())
		}
	}

	// Run crytic-compile
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, "", fmt.Errorf("error while executing crytic-compile:\nOUTPUT:\n%s\nERROR: %s\n", string(out), err.Error())
	}

	// Find compilation artifacts in the export directory
	matches, err := filepath.Glob(filepath.Join(exportDirectory, "*.json"))
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
		var compiledJson map[string]any
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
		compilationMap, ok := compilationUnits.(map[string]any)
		if !ok {
			return nil, "", fmt.Errorf("compilationUnits is not in the map[string]any format: %s\n", compilationUnits)
		}

		// Iterate through compilationUnits
		for _, compilationUnit := range compilationMap {
			// Create a compilation object that will store the contracts and asts for a single compilation unit
			compilation := types.NewCompilation()

			// Create mapping between key (compiler / asts / contracts) and associated values
			compilationUnitMap, ok := compilationUnit.(map[string]any)
			if !ok {
				return nil, "", fmt.Errorf("compilationUnit is not in the map[string]any format: %s\n", compilationUnit)
			}

			// Create mapping between each file in compilation unit and associated Ast
			AstMap := compilationUnitMap["asts"].(map[string]any)

			// Create mapping between key (file name) and value (associated contracts in that file)
			contractsMap, ok := compilationUnitMap["contracts"].(map[string]any)
			if !ok {
				return nil, "", fmt.Errorf("cannot find 'contracts' key in compilationUnitMap: %s\n", compilationUnitMap)
			}

			// Iterate through each contract FILE (note that each FILE might have more than one contract)
			for _, contractsData := range contractsMap {
				// Create mapping between all contracts in a file (key) to it's data (abi, etc.)
				contractMap, ok := contractsData.(map[string]any)
				if !ok {
					return nil, "", fmt.Errorf("contractsData is not in the map[string]any format: %s\n", contractsData)
				}

				// Iterate through each contract
				for contractName, contractData := range contractMap {
					// Create mapping between contract details (abi, bytecode) to actual values
					contractDataMap, ok := contractData.(map[string]any)
					if !ok {
						return nil, "", fmt.Errorf("contractData is not in the map[string]any format: %s\n", contractData)
					}

					// Create mapping between "filenames" (key) associated with the contract and the various filename
					// types (absolute, relative, short, long)
					fileMap, ok := contractDataMap["filenames"].(map[string]any)
					if !ok {
						return nil, "", fmt.Errorf("cannot find 'filenames' key in contractDataMap: %s\n", contractDataMap)
					}

					// Create unique source path which is going to be absolute path
					sourcePath := fmt.Sprintf("%v", fileMap["absolute"])

					// Parse the ABI
					contractAbi, err := types.ParseABIFromInterface(contractDataMap["abi"])
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
