package platforms

import (
	"encoding/json"
	"fmt"
	"github.com/trailofbits/medusa/compilation/types"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
)

type CryticCompileCompilationConfig struct {
	Target         string   `json:"target"`
	SolcVersion    string   `json:"solcVersion"`
	SolcInstall    bool     `json:"solcINstall"`
	BuildDirectory string   `json:"buildDirectory"`
	Args           []string `json:"args,omitempty"`
}

func NewCryticCompileCompilationConfig(target string) *CryticCompileCompilationConfig {
	return &CryticCompileCompilationConfig{
		Target:         target,
		BuildDirectory: "",
		Args:           []string{},
		SolcVersion:    "",
	}
}

func (s *CryticCompileCompilationConfig) Compile() ([]types.Compilation, string, error) {
	// Get information on s.Target
	// Primarily using pathInfo to figure out if s.Target is a directory or not
	pathInfo, err := os.Stat(s.Target)
	if err != nil {
		return nil, "", fmt.Errorf("error while trying to get information on directory %s\n", s.Target)
	}

	// TODO: can catch this upstream if we want
	// Figure out whether s.Target is a file or directory
	// If it is a directory, parentDir is s.Target
	parentDirectory := s.Target
	if !pathInfo.IsDir() {
		// If it is a file, get the parent directory of s.Target
		parentDirectory = path.Dir(s.Target)
	}

	// TODO: what if s.Args also contains --export-format?
	// Ensure that the export format is `solc` for parsing. Append additional crytic-compile args as well
	args := append([]string{".", "--export-format", "standard"}, s.Args...)
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
			return nil, "", fmt.Errorf("error while parsing compiledJson: %s\n", compiledJson)
		}
		// Create a mapping between key (filename) and value (contract and ast information) each compilation unit
		compilationMap, ok := compilationUnits.(map[string]interface{})
		if !ok {
			return nil, "", fmt.Errorf("error while parsing compilationUnits: %s\n", compilationUnits)
		}
		for _, contractsAndAst := range compilationMap {
			// Create mapping between key (compiler / asts / contracts) and associated values
			contractsAndAstMap, ok := contractsAndAst.(map[string]interface{})
			if !ok {
				return nil, "", fmt.Errorf("error while parsing contractsAndAst: %s\n", contractsAndAst)
			}
			Ast := contractsAndAstMap["asts"]
			// Create mapping between key (file name) and value (associated contracts in that file)
			contractsMap, ok := contractsAndAstMap["contracts"].(map[string]interface{})
			if !ok {
				return nil, "", fmt.Errorf("error while parsing contractsAndAstMap: %s\n", contractsAndAstMap)
			}
			// Iterate through each contract FILE
			for fileName, contractsData := range contractsMap {
				// Create mapping between all contracts in a file (key) to it's data (abi, etc.)
				contractMap, ok := contractsData.(map[string]interface{})
				if !ok {
					return nil, "", fmt.Errorf("error while parsing contractsData: %s\n", contractsData)
				}
				// Iterate through each contract
				for contractName, contractData := range contractMap {
					// Create unique source path
					sourcePath := fileName + ":" + contractName
					// Create mapping between contract details (abi, bytecode) to actual values
					contractDataMap, ok := contractData.(map[string]interface{})
					if !ok {
						return nil, "", fmt.Errorf("error while parsing contractData: %s\n", contractData)
					}
					contractAbi, err := types.InterfaceToABI(contractDataMap["abi"])
					if err != nil {
						// TODO: Throw error here?
						continue
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
