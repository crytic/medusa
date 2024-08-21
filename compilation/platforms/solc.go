package platforms

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"github.com/Masterminds/semver"
	"github.com/crytic/medusa/compilation/types"
	"github.com/crytic/medusa/utils"
	"github.com/ethereum/go-ethereum/common/compiler"
)

type SolcCompilationConfig struct {
	Target string `json:"target"`
}

func NewSolcCompilationConfig(target string) *SolcCompilationConfig {
	return &SolcCompilationConfig{
		Target: target,
	}
}

func (s *SolcCompilationConfig) Platform() string {
	return "solc"
}

// GetTarget returns the target for compilation
func (s *SolcCompilationConfig) GetTarget() string {
	return s.Target
}

// SetTarget sets the new target for compilation
func (s *SolcCompilationConfig) SetTarget(newTarget string) {
	s.Target = newTarget
}

func GetSystemSolcVersion() (*semver.Version, error) {
	// Run solc --version to obtain our compiler version.
	out, err := exec.Command("solc", "--version").CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("error while executing solc:\nOUTPUT:\n%s\nERROR: %s\n", string(out), err.Error())
	}

	// Parse the compiler version out of the output
	exp := regexp.MustCompile(`\d+\.\d+\.\d+`)
	versionStr := exp.FindString(string(out))
	if versionStr == "" {
		return nil, errors.New("could not parse solc version using 'solc --version'")
	}

	// Parse our semver string and return it
	return semver.NewVersion(versionStr)
}

// SetSolcOutputOptions determines what outputOptions should be provided to solc given a semver.Version
func (s *SolcCompilationConfig) SetSolcOutputOptions(v *semver.Version) string {
	// useCompactFormat will add the compact-format output option
	// if version is 0.4.12-0.4.26 or 0.5.0-0.5.17 or 0.6.0-0.6.12 or 0.7.0-0.7.6 or 0.8.0-0.8.9
	useCompactFormat := (v.Major() == 0 && v.Minor() == 4 && v.Patch() >= 12 && v.Patch() <= 26) ||
		(v.Major() == 0 && v.Minor() == 5 && v.Patch() <= 17) ||
		(v.Major() == 0 && v.Minor() == 6 && v.Patch() <= 12) ||
		(v.Major() == 0 && v.Minor() == 7 && v.Patch() <= 6) ||
		(v.Major() == 0 && v.Minor() == 8 && v.Patch() <= 9)

	// if version is 0.3.0-0.3.6 or 0.4.0-0.4.11 no 'hashes' outputOption
	if (v.Major() == 0 && v.Minor() == 4 && v.Patch() <= 11) || (v.Major() == 0 && v.Minor() == 3 && v.Patch() <= 6) {
		return "abi,ast,bin,bin-runtime,srcmap,srcmap-runtime,userdoc,devdoc"
	} else if useCompactFormat {
		// Both 'hashes' and 'compact-format' are allowed as outputOptions
		return "abi,ast,bin,bin-runtime,srcmap,srcmap-runtime,userdoc,devdoc,hashes,compact-format"
	} else {
		// Can't use 'compact-format' but 'hashes' is allowed as outputOption
		return "abi,ast,bin,bin-runtime,srcmap,srcmap-runtime,userdoc,devdoc,hashes"
	}
}
func (s *SolcCompilationConfig) Compile() ([]types.Compilation, string, error) {
	// Obtain our solc version string
	v, err := GetSystemSolcVersion()
	if err != nil {
		return nil, "", err
	}

	// Determine which compiler options we need.
	outputOptions := s.SetSolcOutputOptions(v)

	// Create our command
	cmd := exec.Command("solc", s.Target, "--combined-json", outputOptions)
	cmdStdout, cmdStderr, cmdCombined, err := utils.RunCommandWithOutputAndError(cmd)
	if err != nil {
		return nil, "", fmt.Errorf("error while executing solc:\n%s\n\nCommand Output:\n%s\n", err.Error(), string(cmdCombined))
	}

	// Our compilation succeeded, load the JSON
	var results map[string]any
	err = json.Unmarshal(cmdStdout, &results)
	if err != nil {
		return nil, "", err
	}

	// Create a compilation unit out of this.
	compilation := types.NewCompilation()

	// Create a map of contract names to their kinds
	contractKinds := make(map[string]types.ContractKind)

	// Parse our sources from solc output
	if sources, ok := results["sources"]; ok {
		if sourcesMap, ok := sources.(map[string]any); ok {
			for sourcePath, source := range sourcesMap {
				// Treat our source as a key-value lookup
				sourceDict, sourceCorrectType := source.(map[string]any)
				if !sourceCorrectType {
					return nil, "", fmt.Errorf("could not parse compiled source artifact because it could not be casted to a dictionary")
				}

				// Try to obtain our AST key
				origAST, hasAST := sourceDict["AST"]
				if !hasAST {
					return nil, "", fmt.Errorf("could not parse AST from sources, AST field could not be found")
				}

				// Convert the AST into our version of the AST (types.AST)
				var ast types.AST
				b, err := json.Marshal(origAST)
				if err != nil {
					return nil, "", fmt.Errorf("could not encode AST from sources: %v", err)
				}
				err = json.Unmarshal(b, &ast)
				if err != nil {
					return nil, "", fmt.Errorf("could not parse AST from sources, error: %v", err)
				}

				// From the AST, extract the contract kinds where the contract definition could be for a contract, library,
				// or interface
				for _, node := range ast.Nodes {
					if node.GetNodeType() == "ContractDefinition" {
						contractDefinition := node.(types.ContractDefinition)
						contractKinds[contractDefinition.CanonicalName] = contractDefinition.Kind
					}
				}

				// Get the source unit ID
				sourceUnitId := ast.GetSourceUnitID()
				// Construct our compiled source object
				compilation.SourcePathToArtifact[sourcePath] = types.SourceArtifact{
					// TODO our types.AST is not the same as the original AST but we could parse it and avoid using "any"
					Ast:          origAST,
					Contracts:    make(map[string]types.CompiledContract),
					SourceUnitId: sourceUnitId,
				}
				compilation.SourceIdToPath[sourceUnitId] = sourcePath

			}
		}
	}

	// Parse our contracts from solc output
	contracts, err := compiler.ParseCombinedJSON(cmdStdout, "solc", v.String(), v.String(), "")
	if err != nil {
		return nil, "", err
	}

	for name, contract := range contracts {
		// Split our name which should be of form "filename:contractname"
		nameSplit := strings.Split(name, ":")
		sourcePath := strings.Join(nameSplit[0:len(nameSplit)-1], ":")
		contractName := nameSplit[len(nameSplit)-1]

		// Convert the abi structure to our parsed abi type
		contractAbi, err := types.ParseABIFromInterface(contract.Info.AbiDefinition)
		if err != nil {
			continue
		}

		// Decode our init and runtime bytecode
		initBytecode, err := hex.DecodeString(strings.TrimPrefix(contract.Code, "0x"))
		if err != nil {
			return nil, "", fmt.Errorf("unable to parse init bytecode for contract '%s'\n", contractName)
		}
		runtimeBytecode, err := hex.DecodeString(strings.TrimPrefix(contract.RuntimeCode, "0x"))
		if err != nil {
			return nil, "", fmt.Errorf("unable to parse runtime bytecode for contract '%s'\n", contractName)
		}

		// Construct our compiled contract
		compilation.SourcePathToArtifact[sourcePath].Contracts[contractName] = types.CompiledContract{
			Abi:             *contractAbi,
			InitBytecode:    initBytecode,
			RuntimeBytecode: runtimeBytecode,
			SrcMapsInit:     contract.Info.SrcMap.(string),
			SrcMapsRuntime:  contract.Info.SrcMapRuntime,
			Kind:            contractKinds[contractName],
		}
	}

	return []types.Compilation{*compilation}, string(cmdStderr), nil
}
