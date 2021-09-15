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

type WaffleCompilationConfig struct {
	Target string `json:"target"`
	UseNpx bool `json:"use_npx"`
	BuildDirectory string `json:"build_directory,omitempty"`
}

func NewWaffleCompilationConfig(target string) *WaffleCompilationConfig {
	return &WaffleCompilationConfig{
		Target: target,
		UseNpx: true,
		BuildDirectory: "",
	}
}

func (s *WaffleCompilationConfig) Compile() ([]types.Compilation, string, error) {
	// Determine the base command to use.
	var cmd *exec.Cmd = exec.Command("npm", "run", "build")

	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, "", fmt.Errorf("error while executing Waffle:\nOUTPUT:\n%s\nERROR: %s\n", string(out), err.Error())
	}

	// Create a compilation unit out of this.
	compilation := types.NewCompilation()

	// Find all the compiled Waffle artifacts
	targetDirectory := path.Dir(s.Target)
	buildDirectory := s.BuildDirectory
	if buildDirectory == "" {
		buildDirectory = path.Join(targetDirectory, "build"	)
	}
	matches, err := filepath.Glob(path.Join(buildDirectory, "*.json"))
	if err != nil {
		return nil, "", err
	}

	// Define our Waffle structure to parse
	type WaffleCompiledJson struct {
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

	// Loop for each Waffle artifact to parse our compilations.
	for i := 0; i < len(matches); i++ {
		// Read the compiled JSON file data
		b, err := ioutil.ReadFile(matches[i])
		if err != nil {
			return nil, "", err
		}

		// Parse the JSON
		var compiledJson WaffleCompiledJson
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
