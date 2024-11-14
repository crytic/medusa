package types

import (
	"encoding/json"
)

// Printer contains the slither printer echidna result
type Slither struct {
	// Success is true if the printer succeeded
	Success bool `json:"success"`
	// Error is the eventual error reported by slither
	Error string `json:"error"`
	// ConstantsUsed the constants extracted by slither 
	ConstantsUsed []ConstantUsed `json:"constantsUsed"`
}


type ConstantUsed struct {
	Type string `json:"type"`
	Value string `json:"value"`
}


func (s *Slither) UnmarshalJSON(d []byte) error {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(d, &obj); err != nil {
		return err
	}

	// Decode success and error. They are always present
	json.Unmarshal(obj["success"], &s.Success)
	json.Unmarshal(obj["error"], &s.Error)

	s.ConstantsUsed = make([]ConstantUsed, 0)
	
	var results map[string]json.RawMessage
	json.Unmarshal(obj["results"], &results)

	var printersList []json.RawMessage
	json.Unmarshal(results["printers"], &printersList)

	var printerEchidna map[string]json.RawMessage
	json.Unmarshal(printersList[0], &printerEchidna)

	var description string
	json.Unmarshal(printerEchidna["description"], &description)
	
	var descriptionJson map[string]json.RawMessage
	json.Unmarshal([]byte(description), &descriptionJson)

	var contracts map[string]json.RawMessage
	json.Unmarshal(descriptionJson["constants_used"], &contracts)

	for _, val := range contracts {
		var functions map[string]json.RawMessage
		json.Unmarshal(val, &functions)
		for _, val := range functions {
			var constants [][]ConstantUsed
			json.Unmarshal(val, &constants)
			for _, val := range constants {
				// Slither output the value of a constant as list
				// however we know there can be only 1 so we take index 0
				s.ConstantsUsed = append(s.ConstantsUsed, val[0])
			}
		}
	}

	return nil
}
