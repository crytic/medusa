package platforms

import (
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common/compiler"
	"github.com/trailofbits/medusa/compilation/types"
	"os/exec"
	"strings"
)

type DappCompilationConfig struct {
	Target string `json:"target"`
	BuildDirectory string `json:"build_directory,omitempty"`
}

func NewDappCompilationConfig(target string) *DappCompilationConfig {
	return &DappCompilationConfig{
		Target: target,
		BuildDirectory: "",
	}
}

func (s *DappCompilationConfig) Compile() ([]types.Compilation, string, error) {
	// Obtain our solc version string
	v, err := GetSystemSolcVersion()
	if err != nil {
		return nil, "", err
	}

	// Execute solc to compile our target.
	var cmd *exec.Cmd = exec.Command("dapp", "buld")

	cmd.Dir = s.Target
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, "", fmt.Errorf("error while executing Dapp:\nOUTPUT:\n%s\nERROR: %s\n", string(out), err.Error())
	}

	// Our compilation succeeded, load the JSON
	var results map[string]interface{}
	err = json.Unmarshal(out, &results)
	if err != nil {
		return nil, "", err
	}

	// Create a compilation unit out of this.
	compilation := types.NewCompilation()

	// Parse our sources from solc output
	if sources, ok := results["sources"]; ok {
		if sourcesMap, ok := sources.(map[string]interface{}); ok {
			for name, source := range sourcesMap {
				// Try to obtain our AST key
				ast, _ := source.(map[string]interface{})

				// Construct our compiled source object
				compilation.Sources[name] = types.CompiledSource{
					Ast: ast,
					Contracts: make(map[string]types.CompiledContract),
				}
			}
		}
	}

	// Parse our contracts from solc output
	contracts, err := compiler.ParseCombinedJSON(out, "solc", v.String(), v.String(), "")
	for name, contract := range contracts {
		// Split our name which should be of form "filename:contractname"
		nameSplit := strings.Split(name, ":")
		sourcePath := strings.Join(nameSplit[0:len(nameSplit)-1], ":")
		contractName := nameSplit[len(nameSplit)-1]

		// Convert the abi structure to our parsed abi type
		contractAbi, err := types.InterfaceToABI(contract.Info.AbiDefinition)
		if err != nil {
			continue
		}

		// Construct our compiled contract
		compilation.Sources[sourcePath].Contracts[contractName] = types.CompiledContract{
			Abi: *contractAbi,
			RuntimeBytecode: contract.RuntimeCode,
			InitBytecode: contract.Code,
			SrcMapsInit: contract.Info.SrcMap.(string),
			SrcMapsRuntime: contract.Info.SrcMapRuntime,
		}
	}

	return []types.Compilation{*compilation}, "", nil
}
