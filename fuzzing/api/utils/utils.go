package utils

import (
	"encoding/json"
	"github.com/crytic/medusa/fuzzing"
)

type testCaseMarshaler struct {
	testCase fuzzing.TestCase
}

func (m *testCaseMarshaler) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{
		"Status":     m.testCase.Status(),
		"Name":       m.testCase.Name(),
		"LogMessage": m.testCase.LogMessage(),
		"Message":    m.testCase.Message(),
		"ID":         m.testCase.ID(),
	})
}

func MarshalTestCases(testCases []fuzzing.TestCase) []testCaseMarshaler {
	marshaledTestCases := make([]testCaseMarshaler, len(testCases))
	for i, tc := range testCases {
		marshaledTestCases[i] = testCaseMarshaler{testCase: tc}
	}
	return marshaledTestCases
}
