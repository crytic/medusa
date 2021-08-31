package platforms

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Masterminds/semver"
	"github.com/ethereum/go-ethereum/common/compiler"
	"medusa/compilation/types"
	"os/exec"
	"regexp"
	"strings"
)

type SolcCompilationConfig struct {
	Target string `json:"target"`
}

func NewSolcCompilationConfig(target string) *SolcCompilationConfig {
	return &SolcCompilationConfig{
		Target: target,
	}
}

func GetSystemSolcVersion() (*semver.Version, error) {
	// Run solc --version to obtain our compiler version.
	out, err := exec.Command("solc", "--version").CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("error while executing solc:\nOUTPUT:\n%s\nERROR: %s\n", string(out), err.Error())
	}

	// Parse the compiler version out of the output
	exp := regexp.MustCompile("\\d+\\.\\d+\\.\\d+\\S*")
	versionStr := exp.FindString(string(out))
	if versionStr == "" {
		return nil, errors.New("could not parse solc version using 'solc --version'")
	}

	// Parse our semver string and return it
	return semver.NewVersion(versionStr)
}

func (s *SolcCompilationConfig) Compile() ([]types.Compilation, error) {
	// Obtain our solc version string
	v, err := GetSystemSolcVersion()
	if err != nil {
		return nil, err
	}

	// Determine which compiler options we need.
	var outputOptions string
	if (v.Major() == 0 && v.Minor() == 3) || (v.Major() == 0 && v.Minor() == 4 && v.Patch() <= 12) {
		outputOptions = "abi,ast,bin,bin-runtime,srcmap,srcmap-runtime,userdoc,devdoc"
	} else {
		outputOptions = "abi,ast,bin,bin-runtime,srcmap,srcmap-runtime,userdoc,devdoc,hashes,compact-format"
	}

	// Execute solc to compile our target.
	out, err := exec.Command("solc", s.Target, "--combined-json", outputOptions).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("error while executing solc:\nOUTPUT:\n%s\nERROR: %s\n", string(out), err.Error())
	}

	// Our compilation succeeded, load the JSON
	var results map[string]interface{}
	err = json.Unmarshal(out, &results)
	if err != nil {
		return nil, err
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

	return []types.Compilation{*compilation}, nil
}
