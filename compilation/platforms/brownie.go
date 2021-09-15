package platforms

import (
	"encoding/json"
	"fmt"
	"github.com/trailofbits/medusa/compilation/types"
	"io/ioutil"
	"os/exec"
	"path"
	"path/filepath"
)

type BrownieCompilationConfig struct {
	Target string `json:"target"`
	BuildDirectory string `json:"build_directory,omitempty"`
}

func NewBrownieCompilationConfig(target string) *BrownieCompilationConfig {
	return &BrownieCompilationConfig{
		Target: target,
		BuildDirectory: "",
	}
}

func (s *BrownieCompilationConfig) Compile() ([]types.Compilation, string, error) {
	var cmd *exec.Cmd = exec.Command("brownie", "compile")

	cmd.Dir = s.Target
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, "", fmt.Errorf("error while executing Brownie:\nOUTPUT:\n%s\nERROR: %s\n", string(out), err.Error())
	}

	// Create a compilation unit out of this.
	compilation := types.NewCompilation()

	// Find all the compiled Brownie artifacts
	targetDirectory := path.Dir(s.Target)
	buildDirectory := s.BuildDirectory
	if buildDirectory == "" {
		buildDirectory = path.Join(targetDirectory, "build", "contracts")
	}
	matches, err := filepath.Glob(path.Join(buildDirectory, "*.json"))
	if err != nil {
		return nil, "", err
	}

	// Define our Brownie structure to parse
	type BrownieCompiledJson struct {
		ContractName string `json:"contractName"`
		Abi interface{} `json:"abi"`
		Bytecode string `json:"bytecode"`
		DeployedBytecode string `json:"deployedBytecode"`
		SourceMap string `json:"sourceMap"`
		DeployedSourceMap string `json:"deployedSourceMap"`
		Source string `json:"source"`
		SourcePath string `json:"sourcePath"`
		Ast interface{} `json:"ast"`
	}

	// Loop for each Brownie artifact to parse our compilations.
	for i := 0; i < len(matches); i++ {
		// Read the compiled JSON file data
		b, err := ioutil.ReadFile(matches[i])
		if err != nil {
			return nil, "", err
		}

		// Parse the JSON
		var compiledJson BrownieCompiledJson
		err = json.Unmarshal(b, &compiledJson)
		if err != nil {
			return nil, "", err
		}

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
			Abi: *contractAbi,
			RuntimeBytecode: compiledJson.DeployedBytecode,
			InitBytecode: compiledJson.Bytecode,
			SrcMapsInit: compiledJson.SourceMap,
			SrcMapsRuntime: compiledJson.DeployedSourceMap,
		}
	}

	return []types.Compilation{*compilation}, string(out), nil
}
