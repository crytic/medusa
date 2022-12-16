package slither

import (
	"encoding/json"
	"fmt"
	"github.com/trailofbits/medusa/utils"
	"os/exec"
)

// RunPrinter will run Slither's echidna printer and returns an object that will store the output data.
// Note that the function will not compile the target since it is expected that the compilation artifacts already exist
// at the target
func RunPrinter(target string) (*SlitherData, error) {
	// Make sure target is not the empty string
	if target == "" {
		return nil, fmt.Errorf("must provide a target to run slither's echidna printer")
	}

	// Set up the arguments necessary for a no-compile slither printer run
	args := []string{target, "--print", "echidna", "--ignore-compile", "--json", "-"}

	// Run the command
	cmd := exec.Command("slither", args...)
	cmdStdout, _, cmdCombined, err := utils.RunCommandWithOutputAndError(cmd)

	// If we failed, exit out
	if err != nil {
		return nil, fmt.Errorf("error while running slither:\n%s\n\nCommand Output:\n%s\n", err.Error(), string(cmdCombined))
	}

	// The actual printer data will be stored in the results[`results`][`printers`] key. The value of `printers` is
	// actually a _list_ of mappings where the `description` key at the 0th index will hold the data for the `echidna` printer
	type rawSlitherOutput struct {
		Success bool                        `json:"success"`
		Error   any                         `json:"error"`
		Results map[string][]map[string]any `json:"results"`
	}

	// Unmarshal stdout
	var rawOutput rawSlitherOutput
	err = json.Unmarshal(cmdStdout, &rawOutput)
	if err != nil {
		return nil, fmt.Errorf("error while unmarshaling slither's output: %v\n", err.Error())
	}

	// For some reason, the printer or slither failed to run, exit out
	if rawOutput.Error != nil || !rawOutput.Success {
		return nil, fmt.Errorf("slither returned the following error: %v\n", rawOutput.Error)
	}

	rawResults := rawOutput.Results

	// Make sure there is only one printer result and those results are from the `echidna` printer
	if len(rawResults["printers"]) != 1 || rawResults["printers"][0]["printer"] != ECHIDNA_PRINTER {
		return nil, fmt.Errorf("expected the slither output to contain the results from the echidna printer")
	}

	var slitherData *SlitherData

	// Unmarshal the data into a SlitherData type
	err = json.Unmarshal([]byte(rawResults["printers"][0]["description"].(string)), &slitherData)
	if err != nil {
		return nil, fmt.Errorf("error while unmarshaling slither's echidna printer results: %v\n", err.Error())
	}

	// Create the parse-able version of all the constants found in the target
	slitherData.Constants, err = makeConstantsList(slitherData.ConstantsUsed)
	if err != nil {
		return nil, fmt.Errorf("error while parsing the constants in the output: %v\n", err.Error())
	}

	return slitherData, nil
}
