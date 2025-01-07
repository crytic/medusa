package types

import (
	"encoding/json"
	"fmt"
	"os/exec"
)

// SlitherConfig determines whether to run slither and whether to cache the results from slither
type SlitherConfig struct {
	// RunSlither determines whether to run slither
	RunSlither bool `json:"runSlither"`
	// UseCache determines whether the results of slither should be cached or not
	UseCache bool `json:"useCache"`
}

// NewDefaultSlitherConfig provides a default configuration to run slither
func NewDefaultSlitherConfig() (*SlitherConfig, error) {
	return &SlitherConfig{RunSlither: true, UseCache: true}, nil
}

// SlitherResults describes a data structures that holds the interesting constants returned from slither
type SlitherResults struct {
	// Constants holds the constants extracted by slither
	Constants []Constant `json:"constantsUsed"`
}

// Constant defines a constant that was extracted by slither while parsing the compilation target
type Constant struct {
	// Type represents the ABI type of the constant
	Type string `json:"type"`
	// Value represents the value of the constant
	Value string `json:"value"`
}

// Run slither on the provided compilation target. A SlitherResults data structure is returned.
func (s *SlitherConfig) Run(target string) (*SlitherResults, error) {
	// Return early if we do not want to run slither
	if !s.RunSlither {
		return nil, nil
	}

	// Use the cached slither output if it exists
	if s.UseCache {
		// TODO
	}

	out, err := exec.Command("slither", target, "--ignore-compile", "--print", "echidna", "--json", "-").CombinedOutput()

	if err != nil {
		return nil, err
	}

	// Capture the slither results
	var slitherResults SlitherResults
	err = json.Unmarshal(out, &slitherResults)
	if err != nil {
		return nil, err
	}

	return &slitherResults, nil
}

// UnmarshalJSON unmarshals the slither output into a Slither type
func (s *SlitherResults) UnmarshalJSON(d []byte) error {
	// Extract the top-level JSON object
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(d, &obj); err != nil {
		return err
	}

	// Decode success and error. They are always present in the slither output
	var success bool
	var slitherError string
	if err := json.Unmarshal(obj["success"], &success); err != nil {
		return err
	}

	if err := json.Unmarshal(obj["error"], &slitherError); err != nil {
		return err
	}

	// If success is not true or there is a non-null error, return early
	if !success || slitherError != "" {
		return fmt.Errorf("failed to parse slither results")
	}

	// Now we will extract the constants
	s.Constants = make([]Constant, 0)

	// Iterate through the JSON object until we get to the constants_used key
	// First, retrieve the results
	var results map[string]json.RawMessage
	if err := json.Unmarshal(obj["results"], &results); err != nil {
		return err
	}

	// Retrieve the printers data
	var printers []json.RawMessage
	if err := json.Unmarshal(results["printers"], &printers); err != nil {
		return err
	}

	// Since we are running the echidna printer, we know that the first element is the one we care about
	var echidnaPrinter map[string]json.RawMessage
	if err := json.Unmarshal(printers[0], &echidnaPrinter); err != nil {
		return err
	}

	var descriptionString string
	if err := json.Unmarshal(echidnaPrinter["description"], &descriptionString); err != nil {
		return err
	}

	var description map[string]json.RawMessage
	if err := json.Unmarshal([]byte(descriptionString), &description); err != nil {
		return err
	}

	// Capture all the constants extracted across all the contracts in scope
	var constantsInContracts map[string]json.RawMessage
	if err := json.Unmarshal(description["constants_used"], &constantsInContracts); err != nil {
		return err
	}

	// Iterate across the constants in each contract
	for _, constantsInContract := range constantsInContracts {
		// Capture all the constants in a given function
		var constantsInFunctions map[string]json.RawMessage
		if err := json.Unmarshal(constantsInContract, &constantsInFunctions); err != nil {
			return err
		}

		// Iterate across each function
		for _, constantsInFunction := range constantsInFunctions {
			// Each constant is provided as its own list, so we need to create a matrix
			var constants [][]Constant
			if err := json.Unmarshal(constantsInFunction, &constants); err != nil {
				return err
			}
			for _, constant := range constants {
				// Slither outputs the value of a constant as a list
				// However we know there can be only 1 so we take index 0
				s.Constants = append(s.Constants, constant[0])
			}
		}
	}

	return nil
}
