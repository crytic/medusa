package platforms

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/trailofbits/medusa/compilation/types"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
)

type CryticCompilationConfig struct {
	Target         string   `json:"target"`
	SolcVersion    string   `json:"solcVersion"`
	SolcInstall    bool     `json:"solcINstall"`
	BuildDirectory string   `json:"buildDirectory"`
	Args           []string `json:"args,omitempty"`
}

func NewCryticCompilationConfig(target string) *CryticCompilationConfig {
	return &CryticCompilationConfig{
		Target:         target,
		BuildDirectory: "",
		Args:           []string{},
		SolcVersion:    "",
	}
}

func (s *CryticCompilationConfig) ValidateArgs() error {
	// If --export-format or --export-dir are specified in s.Args, throw an error
	for _, arg := range s.Args {
		if arg == "--export-format" {
			return errors.New("do not specify `--export-format` as an argument since the standard export format is required by medusa")
		}
		if arg == "--export-dir" {
			return errors.New("do not specify `--export-dir` as an argument, use the BuildDirectory config variable instead")
		}
	}
	return nil
}

func (s *CryticCompilationConfig) Compile() ([]types.Compilation, string, error) {
	// Validate args to make sure --export-format and --export-dir are not specified
	err := s.ValidateArgs()
	if err != nil {
		return nil, "", err
	}
	// Get information on s.Target
	// Primarily using pathInfo to figure out if s.Target is a directory or not
	pathInfo, err := os.Stat(s.Target)
	if err != nil {
		return nil, "", fmt.Errorf("error while trying to get information on directory %s\n", s.Target)
	}

	// TODO: can catch this upstream if we want
	// Figure out whether s.Target is a file or directory
	parentDirectory := s.Target
	// Since we are compiling a whole directory, use "."
	args := append([]string{".", "--export-format", "standard"}, s.Args...)
	if !pathInfo.IsDir() {
		// If it is a file, get the parent directory of s.Target
		parentDirectory = path.Dir(s.Target)
		// Since we are compiling a file, use s.Target
		args = append([]string{s.Target, "--export-format", "standard"}, s.Args...)
	}

	// Get main command and set working directory
	cmd := exec.Command("crytic-compile", args...)
	cmd.Dir = parentDirectory

	// Install a specific `solc` version if requested in the config
	if s.SolcVersion != "" {
		err := exec.Command("solc-select", "install", s.SolcVersion).Run()
		if err != nil {
			return nil, "", fmt.Errorf("error while executing solc-select:\n\nERROR: %s\n", err.Error())
		}
	}

	// Run crytic-compile
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, "", fmt.Errorf("error while executing crytic-compile:\nOUTPUT:\n%s\nERROR: %s\n", string(out), err.Error())
	}

	// Find build directory
	buildDirectory := s.BuildDirectory
	if buildDirectory == "" {
		buildDirectory = path.Join(parentDirectory, "crytic-export")
	}
	matches, err := filepath.Glob(path.Join(buildDirectory, "*.json"))
	if err != nil {
		return nil, "", err
	}

	// Create a compilation unit out of this.
	compilation := types.NewCompilation()

	// Loop for each crytic artifact to parse our compilations.
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
			return nil, "", fmt.Errorf("cannot find 'compilation_units' key in compiledJson: %s\n", compiledJson)
		}
		// Create a mapping between key (filename) and value (contract and ast information) each compilation unit
		compilationMap, ok := compilationUnits.(map[string]interface{})
		if !ok {
			return nil, "", fmt.Errorf("compilationUnits is not in the map[string]interface{} format: %s\n", compilationUnits)
		}
		for _, contractsAndAst := range compilationMap {
			// Create mapping between key (compiler / asts / contracts) and associated values
			contractsAndAstMap, ok := contractsAndAst.(map[string]interface{})
			if !ok {
				return nil, "", fmt.Errorf("contractsAndAst is not in the map[string]interface{} format: %s\n", contractsAndAst)
			}
			Ast := contractsAndAstMap["asts"]
			// Create mapping between key (file name) and value (associated contracts in that file)
			contractsMap, ok := contractsAndAstMap["contracts"].(map[string]interface{})
			if !ok {
				return nil, "", fmt.Errorf("cannot find 'contracts' key in contractsAndAstMap: %s\n", contractsAndAstMap)
			}
			// Iterate through each contract FILE
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
					// Create unique source path which is going to be absolute path
					fileMap, ok := contractDataMap["filenames"].(map[string]interface{})
					if !ok {
						return nil, "", fmt.Errorf("cannot find 'filenames' key in contractsAndAstMap: %s\n", contractsAndAstMap)
					}
					sourcePath := fmt.Sprintf("%v", fileMap["absolute"])
					// Get ABI
					contractAbi, err := types.InterfaceToABI(contractDataMap["abi"])
					if err != nil {
						return nil, "", fmt.Errorf("Unable to parse ABI: %s\n", contractDataMap["abi"])
					}
					// Check if source is already in compilation object
					if _, ok := compilation.Sources[sourcePath]; !ok {
						compilation.Sources[sourcePath] = types.CompiledSource{
							Ast:       Ast,
							Contracts: make(map[string]types.CompiledContract),
						}
					}
					compilation.Sources[sourcePath].Contracts[contractName] = types.CompiledContract{
						Abi:             *contractAbi,
						RuntimeBytecode: fmt.Sprintf("%v", contractDataMap["bin-runtime"]),
						InitBytecode:    fmt.Sprintf("%v", contractDataMap["bin"]),
						SrcMapsInit:     fmt.Sprintf("%v", contractDataMap["srcmap"]),
						SrcMapsRuntime:  fmt.Sprintf("%v", contractDataMap["srcmap-runtime"]),
					}

				}
			}
		}
	}
	return []types.Compilation{*compilation}, string(out), nil
}
