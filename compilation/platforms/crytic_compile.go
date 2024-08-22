package platforms

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/crytic/medusa/compilation/types"
	"github.com/crytic/medusa/logging"
	"github.com/crytic/medusa/utils"
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
// or the `--export-dir` arguments. This is because `--export-format` has to be `solc` for the `crytic-compile`
// integration to work and CryticCompilationConfig.BuildDirectory option is equivalent to `--export-dir`
func (c *CryticCompilationConfig) validateArgs() error {
	// If --export-format or --export-dir are specified in c.Args, throw an error
	for _, arg := range c.Args {
		if arg == "--export-format" {
			return errors.New("do not specify `--export-format` within crytic-compile arguments as the solc export format is always used")
		}
		if arg == "--export-dir" {
			return errors.New("do not specify `--export-dir` as an argument, use the BuildDirectory config variable instead")
		}
	}
	return nil
}

// getArgs returns the arguments to be provided to crytic-compile during compilation, or an error if one occurs.
func (c *CryticCompilationConfig) getArgs() ([]string, error) {
	// By default we export in solc mode.
	args := []string{c.Target, "--export-format", "solc"}

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
		return nil, "", fmt.Errorf("could not delete crytic-compile's export directory prior to compilation, error: %v", err)
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
	logging.GlobalLogger.Info("Running command:\n", cmd.String())

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

	// Run crytic-compile to compile and export our compilation artifacts.
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, "", fmt.Errorf("error while executing crytic-compile:\nOUTPUT:\n%s\nERROR: %s\n", string(out), err.Error())
	}

	// Find compilation artifacts in the export directory
	matches, err := filepath.Glob(filepath.Join(exportDirectory, "*.json"))
	if err != nil {
		return nil, "", err
	}

	// Create a slice to track all our compilations parsed.
	var compilationList []types.Compilation

	// Define the structure of our crytic-compile export data.
	type solcSourceUnit struct {
		AST any `json:"AST"`
	}
	type solcExportContract struct {
		SrcMap        string `json:"srcmap"`
		SrcMapRuntime string `json:"srcmap-runtime"`
		Abi           any    `json:"abi"`
		Bin           string `json:"bin"`
		BinRuntime    string `json:"bin-runtime"`
	}
	type solcExportData struct {
		Sources   map[string]solcSourceUnit     `json:"sources"`
		Contracts map[string]solcExportContract `json:"contracts"`
	}

	// Loop through each .json file for compilation units.
	for i := 0; i < len(matches); i++ {
		// Read the compiled JSON file data
		b, err := os.ReadFile(matches[i])
		if err != nil {
			return nil, "", fmt.Errorf("could not parse crytic-compile's exported solc data at path '%s', error: %v", matches[i], err)
		}

		// Parse the JSON
		var solcExport solcExportData
		err = json.Unmarshal(b, &solcExport)
		if err != nil {
			return nil, "", fmt.Errorf("could not parse crytic-compile's exported solc data, error: %v", err)
		}

		// Create a compilation object that will store the contracts and source information.
		compilation := types.NewCompilation()

		// Create a map of contract names to their kinds
		contractKinds := make(map[string]types.ContractKind)

		// Loop through all sources and parse them into our types.
		for sourcePath, source := range solcExport.Sources {
			// Convert the AST into our version of the AST (types.AST)
			var ast types.AST
			b, err = json.Marshal(source.AST)
			if err != nil {
				return nil, "", fmt.Errorf("could not encode AST from sources: %v", err)
			}
			err = json.Unmarshal(b, &ast)
			if err != nil {
				return nil, "", fmt.Errorf("could not parse AST from sources: %v", err)
			}

			// From the AST, extract the contract kinds where the contract definition could be for a contract, library,
			// or interface
			for _, node := range ast.Nodes {
				if node.GetNodeType() == "ContractDefinition" {
					contractDefinition := node.(types.ContractDefinition)
					contractKinds[contractDefinition.CanonicalName] = contractDefinition.Kind
				}
			}

			// Retrieve the source unit ID
			sourceUnitId := ast.GetSourceUnitID()
			compilation.SourcePathToArtifact[sourcePath] = types.SourceArtifact{
				// TODO: Our types.AST is not the same as the original AST but we could parse it and avoid using "any"
				Ast:          source.AST,
				Contracts:    make(map[string]types.CompiledContract),
				SourceUnitId: sourceUnitId,
			}
			compilation.SourceIdToPath[sourceUnitId] = sourcePath
		}

		// Loop through all contracts and parse them into our types.
		for sourceAndContractPath, contract := range solcExport.Contracts {
			// Split our source and contract path, as it takes the form sourcePath:contractName
			splitIndex := strings.LastIndex(sourceAndContractPath, ":")
			if splitIndex == -1 {
				return nil, "", fmt.Errorf("expected contract path to be of form \"<source path>:<contract_name>\"")
			}
			sourcePath := sourceAndContractPath[:splitIndex]
			contractName := sourceAndContractPath[splitIndex+1:]

			// Ensure a source exists for this, or create one if our path somehow differed from any
			// path not existing in the "sources" key at the root of the export.
			if _, ok := compilation.SourcePathToArtifact[sourcePath]; !ok {
				parentSource := types.SourceArtifact{
					Ast:       nil,
					Contracts: make(map[string]types.CompiledContract),
				}
				compilation.SourcePathToArtifact[sourcePath] = parentSource
			}

			// Parse the ABI
			contractAbi, err := types.ParseABIFromInterface(contract.Abi)
			if err != nil {
				return nil, "", fmt.Errorf("unable to parse ABI for contract '%s'\n", contractName)
			}

			// Decode our init and runtime bytecode
			initBytecode, err := hex.DecodeString(strings.TrimPrefix(contract.Bin, "0x"))
			if err != nil {
				return nil, "", fmt.Errorf("unable to parse init bytecode for contract '%s'\n", contractName)
			}
			runtimeBytecode, err := hex.DecodeString(strings.TrimPrefix(contract.BinRuntime, "0x"))
			if err != nil {
				return nil, "", fmt.Errorf("unable to parse runtime bytecode for contract '%s'\n", contractName)
			}

			// Add contract details
			compilation.SourcePathToArtifact[sourcePath].Contracts[contractName] = types.CompiledContract{
				Abi:             *contractAbi,
				InitBytecode:    initBytecode,
				RuntimeBytecode: runtimeBytecode,
				SrcMapsInit:     contract.SrcMap,
				SrcMapsRuntime:  contract.SrcMapRuntime,
				Kind:            contractKinds[contractName],
			}
		}

		compilationList = append(compilationList, *compilation)
	}
	// Return the compilationList
	return compilationList, string(out), nil
}
