package types

import (
	"encoding/json"
	"errors"
	"github.com/crytic/medusa/logging"
	"os"
	"os/exec"
	"time"
)

// SlitherConfig determines whether to run slither and whether and where to cache the results from slither
type SlitherConfig struct {
	// UseSlither determines whether to use slither. If CachePath is non-empty, then the cached results will be
	// attempted to be used. Otherwise, slither will be run.
	UseSlither bool `json:"useSlither"`
	// CachePath determines the path where the slither cache file will be located
	CachePath string `json:"cachePath"`
	// OverwriteCache determines whether to overwrite the cache or not
	// We will not serialize this value since it is something we want to control internally
	OverwriteCache bool `json:"-"`
}

// NewDefaultSlitherConfig provides a default configuration to run slither. The default configuration enables the
// running of slither with the use of a cache.
func NewDefaultSlitherConfig() (*SlitherConfig, error) {
	return &SlitherConfig{
		UseSlither:     true,
		CachePath:      "slither_results.json",
		OverwriteCache: false,
	}, nil
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

// RunSlither on the provided compilation target. RunSlither will use cached results if they exist and write to the
// cache if we have not written to the cache already. A SlitherResults data structure is returned.
func (s *SlitherConfig) RunSlither(target string) (*SlitherResults, error) {
	// Return early if we do not want to run slither
	if !s.UseSlither {
		return nil, nil
	}

	// Use the cached slither output if it exists
	var haveCachedResults bool
	var out []byte
	var err error
	if s.CachePath != "" && !s.OverwriteCache {
		// Check to see if the file exists in the first place.
		// If not, we will re-run slither
		if _, err = os.Stat(s.CachePath); os.IsNotExist(err) {
			logging.GlobalLogger.Info("No Slither cached results found at ", s.CachePath)
			haveCachedResults = false
		} else {
			// We found the cached file
			if out, err = os.ReadFile(s.CachePath); err != nil {
				return nil, err
			}
			haveCachedResults = true
			logging.GlobalLogger.Info("Using cached Slither results found at ", s.CachePath)
		}
	}

	// Run slither if we do not have cached results, or we cannot find the cached results
	if !haveCachedResults {
		// Log the command
		cmd := exec.Command("slither", target, "--ignore-compile", "--print", "echidna", "--json", "-")
		logging.GlobalLogger.Info("Running Slither:\n", cmd.String())

		// Run slither
		start := time.Now()
		out, err = cmd.CombinedOutput()
		if err != nil {
			return nil, err
		}
		logging.GlobalLogger.Info("Finished running Slither in ", time.Since(start).Round(time.Second))
	}

	// Capture the slither results
	var slitherResults SlitherResults
	err = json.Unmarshal(out, &slitherResults)
	if err != nil {
		return nil, err
	}

	// Cache the results if we have not cached before. We have also already checked that the output is well-formed
	// (through unmarshal) so we should be safe.
	if !haveCachedResults && s.CachePath != "" {
		// Cache the data
		err = os.WriteFile(s.CachePath, out, 0644)
		if err != nil {
			// If we are unable to write to the cache, we should log the error but continue
			logging.GlobalLogger.Warn("Failed to cache Slither results at ", s.CachePath, " due to an error:", err)
			// It is possible for os.WriteFile to create a partially written file so it is best to try to delete it
			if _, err = os.Stat(s.CachePath); err == nil {
				// We will not handle the error of os.Remove since we have already checked for the file's existence
				// and we have the right permissions.
				os.Remove(s.CachePath)
			}
		}
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

	// If success is not true or there is a non-empty error string, return early
	if !success || slitherError != "" {
		if slitherError != "" {
			return errors.New(slitherError)
		}
		return errors.New("slither returned a failure during parsing")
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

	// We need to de-serialize the description in two separate steps because go is dumb sometimes
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
