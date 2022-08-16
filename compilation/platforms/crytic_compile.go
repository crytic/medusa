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
	fmt.Printf("Build directory is %s\n", buildDirectory)
	matches, err := filepath.Glob(path.Join(buildDirectory, "*.json"))
	if err != nil {
		return nil, "", err
	}

	// Create a compilation unit out of this.
	compilation := types.NewCompilation()

	// Define various structures to parse `standard` formatting
	type ContractData struct {
		Abi               interface{} `json:"abi"`
		Bytecode          string      `json:"bin"`
		DeployedBytecode  string      `json:"bin-runtime"`
		SourceMap         string      `json:"srcmap"`
		DeployedSourceMap string      `json:"srcmap-runtime"`
	}
	type ContractMapMap map[string]ContractData
	type ContractMap map[string]ContractMapMap
	type CompilationUnit struct {
		Asts      interface{} `json:"asts"`
		Contracts ContractMap `json:"contracts"`
	}
	type CompilationUnitMap map[string]CompilationUnit
	type CryticCompileCompiledJson struct {
		CompilationUnits CompilationUnitMap `json:"compilation_units"`
	}

	// Loop for each truffle artifact to parse our compilations.
	for i := 0; i < len(matches); i++ {
		// Read the compiled JSON file data
		b, err := ioutil.ReadFile(matches[i])
		//fmt.Printf("b is %s\n", b)
		if err != nil {
			return nil, "", err
		}

		// Parse the JSON
		var compiledJson CryticCompileCompiledJson
		err = json.Unmarshal(b, &compiledJson)
		if err != nil {
			return nil, "", err
		}
		for relativeFilePath, _ := range compiledJson.CompilationUnits {
			ast := compiledJson.CompilationUnits[relativeFilePath].Asts
			for fileName, _ := range compiledJson.CompilationUnits[relativeFilePath].Contracts {
				for contractName, _ := range compiledJson.CompilationUnits[relativeFilePath].Contracts[fileName] {
					fmt.Printf("something is %s\n", compiledJson.CompilationUnits[relativeFilePath].Contracts[fileName][contractName].Abi)
					contractAbi, err := types.InterfaceToABI(compiledJson.CompilationUnits[relativeFilePath].Contracts[fileName][contractName].Abi)
					if err != nil {
						continue
					}
					compilation.Sources[contractName] = types.CompiledSource{
						Ast:       ast,
						Contracts: make(map[string]types.CompiledContract),
					}
					compilation.Sources[contractName].Contracts[contractName] = types.CompiledContract{
						Abi:             *contractAbi,
						RuntimeBytecode: compiledJson.CompilationUnits[relativeFilePath].Contracts[fileName][contractName].DeployedBytecode,
						InitBytecode:    compiledJson.CompilationUnits[relativeFilePath].Contracts[fileName][contractName].Bytecode,
						SrcMapsInit:     compiledJson.CompilationUnits[relativeFilePath].Contracts[fileName][contractName].SourceMap,
						SrcMapsRuntime:  compiledJson.CompilationUnits[relativeFilePath].Contracts[fileName][contractName].DeployedSourceMap,
					}
				}
			}
		}
		/*
			fmt.Printf("contract name is %s\n", compiledJson.ContractName)
			// Convert the abi structure to our parsed abi type
			contractAbi, err := types.InterfaceToABI(compiledJson.Abi)
			if err != nil {
				continue
			}

			// If we don't have a source for this file, create it.
			if _, ok := compilation.Sources[compiledJson.SourcePath]; !ok {
				compilation.Sources[compiledJson.SourcePath] = types.CompiledSource{
					Ast:       compiledJson.Ast,
					Contracts: make(map[string]types.CompiledContract),
				}
			}

			// Add our contract to the source
			compilation.Sources[compiledJson.SourcePath].Contracts[compiledJson.ContractName] = types.CompiledContract{
				Abi:             *contractAbi,
				RuntimeBytecode: compiledJson.DeployedBytecode,
				InitBytecode:    compiledJson.Bytecode,
				SrcMapsInit:     compiledJson.SourceMap,
				SrcMapsRuntime:  compiledJson.DeployedSourceMap,
			}
		*/
	}
	return []types.Compilation{*compilation}, string(out), nil
}
