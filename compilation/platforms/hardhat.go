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

type HardhatCompilationConfig struct {
	Target string `json:"target"`
	UseNpx bool `json:"use_npx"`
	Command string `json:"command,omitempty"`
	BuildDirectory string `json:"build_directory,omitempty"`
}

func NewHardhatCompilationConfig(target string) *HardhatCompilationConfig {
	return &HardhatCompilationConfig{
		Target: target,
		UseNpx: true,
		Command: "",
		BuildDirectory: "",
	}
}

func (s *HardhatCompilationConfig) Compile() ([]types.Compilation, string, error) {
	// Determine the base command to use.
	var baseCommandStr = "hardhat"
	if s.Command != "" {
		baseCommandStr = s.Command
	}

	// Execute solc to compile our target.
	var cmd *exec.Cmd
	if s.UseNpx {
		cmd = exec.Command("npx", baseCommandStr, "compile")
	} else {
		cmd = exec.Command(baseCommandStr, "compile")
	}
	cmd.Dir = s.Target
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, "", fmt.Errorf("error while executing Hardhat:\nOUTPUT:\n%s\nERROR: %s\n", string(out), err.Error())
	}

	// Create a compilation unit out of this.
	compilation := types.NewCompilation()

	// Find all the compiled Hardhat artifacts
	targetDirectory := path.Dir(s.Target)
	buildDirectory := s.BuildDirectory
	if buildDirectory == "" {
		buildDirectory = path.Join(targetDirectory, "artifacts", "contracts")
	}
	matches, err := filepath.Glob(path.Join(buildDirectory, "*.sol/*.json"))
	if err != nil {
		return nil, "", err
	}

	// Define our Hardhat structure to parse
	type HardhatCompiledJson struct {
		ContractName string `json:"contractName"`
		Abi interface{} `json:"abi"`
		Bytecode string `json:"bytecode"`
		DeployedBytecode string `json:"deployedBytecode"`
		SourceName string `json:"sourceName"`
		/*
		SourceMap string `json:"sourceMap"`					// TKTK: the fields from here onward aren't present
		DeployedSourceMap string `json:"deployedSourceMap"`
		Source string `json:"source"`
		Ast interface{} `json:"ast"`
		*/
	}

	// Loop for each Hardhat artifact to parse our compilations.
	for i := 0; i < len(matches); i++ {
		// Read the compiled JSON file data
		b, err := ioutil.ReadFile(matches[i])
		if err != nil {
			return nil, "", err
		}

		// Parse the JSON
		var compiledJson HardhatCompiledJson
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
		if _, ok := compilation.Sources[compiledJson.SourceName]; !ok {
			compilation.Sources[compiledJson.SourceName] = types.CompiledSource{
				//Ast:       compiledJson.Ast,
				Contracts: make(map[string]types.CompiledContract),
			}
		}

		// Add our contract to the source
		compilation.Sources[compiledJson.SourceName].Contracts[compiledJson.ContractName] = types.CompiledContract{
			Abi: *contractAbi,
			RuntimeBytecode: compiledJson.DeployedBytecode,
			InitBytecode: compiledJson.Bytecode,
			//SrcMapsInit: compiledJson.SourceMap,
			//SrcMapsRuntime: compiledJson.DeployedSourceMap,
		}
	}

	return []types.Compilation{*compilation}, string(out), nil
}
