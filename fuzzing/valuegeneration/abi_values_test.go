package valuegeneration

import (
	"fmt"
	"math/rand"
	"reflect"
	"testing"
	"time"

	"github.com/crytic/medusa-geth/accounts/abi"
	"github.com/stretchr/testify/assert"
)

// getTestABIArguments obtains ABI arguments of various types for use in testing ABI value related methods.
func getTestABIArguments() abi.Arguments {
	// Define our argument types to test round trip serialization for.
	args := abi.Arguments{
		{
			Name: "testAddress",
			Type: abi.Type{
				Elem:          nil,
				Size:          20,
				T:             abi.AddressTy,
				TupleRawName:  "",
				TupleElems:    nil,
				TupleRawNames: nil,
				TupleType:     nil,
			},
			Indexed: false,
		},
		{
			Name: "testString",
			Type: abi.Type{
				Elem:          nil,
				Size:          0,
				T:             abi.StringTy,
				TupleRawName:  "",
				TupleElems:    nil,
				TupleRawNames: nil,
				TupleType:     nil,
			},
			Indexed: false,
		},
		{
			Name: "testDynamicBytes",
			Type: abi.Type{
				Elem:          nil,
				Size:          0,
				T:             abi.BytesTy,
				TupleRawName:  "",
				TupleElems:    nil,
				TupleRawNames: nil,
				TupleType:     nil,
			},
			Indexed: false,
		},
		{
			Name: "testBool",
			Type: abi.Type{
				Elem:          nil,
				Size:          0,
				T:             abi.BoolTy,
				TupleRawName:  "",
				TupleElems:    nil,
				TupleRawNames: nil,
				TupleType:     nil,
			},
			Indexed: false,
		},
	}

	// Append all fixed byte sizes
	for i := 1; i <= 32; i++ {
		args = append(args, abi.Argument{
			Name: fmt.Sprintf("testBytes%d", i),
			Type: abi.Type{
				Elem:          nil,
				Size:          i,
				T:             abi.FixedBytesTy,
				TupleRawName:  "",
				TupleElems:    nil,
				TupleRawNames: nil,
				TupleType:     nil,
			},
			Indexed: false,
		})
	}

	// Append all integer types
	for i := 8; i <= 256; i += 8 {
		// Add our signed/unsigned types
		args = append(args,
			abi.Argument{
				Name: fmt.Sprintf("int%d", i),
				Type: abi.Type{
					Elem:          nil,
					Size:          i,
					T:             abi.IntTy,
					TupleRawName:  "",
					TupleElems:    nil,
					TupleRawNames: nil,
					TupleType:     nil,
				},
				Indexed: false,
			},
			abi.Argument{
				Name: fmt.Sprintf("uint%d", i),
				Type: abi.Type{
					Elem:          nil,
					Size:          i,
					T:             abi.UintTy,
					TupleRawName:  "",
					TupleElems:    nil,
					TupleRawNames: nil,
					TupleType:     nil,
				},
				Indexed: false,
			})
	}

	// Save our slice of the arguments with basic types
	basicArgs := args[:]

	// Add arguments that are arrays of each basic type
	for _, basicArg := range basicArgs {
		// Alias our arg so when we get a pointer to it, we don't cause a memory leak in this loop
		basicArg := basicArg

		// Define our array arguments.
		arrayArgs := abi.Arguments{
			abi.Argument{
				Name: fmt.Sprintf("testSlice (%v)", basicArg.Type.GetType().String()),
				Type: abi.Type{
					Elem:          &basicArg.Type,
					Size:          0,
					T:             abi.SliceTy,
					TupleRawName:  "",
					TupleElems:    nil,
					TupleRawNames: nil,
					TupleType:     nil,
				},
				Indexed: false,
			},
			abi.Argument{
				Name: fmt.Sprintf("testArray (%v)", basicArg.Type.GetType().String()),
				Type: abi.Type{
					Elem:          &basicArg.Type,
					Size:          5,
					T:             abi.ArrayTy,
					TupleRawName:  "",
					TupleElems:    nil,
					TupleRawNames: nil,
					TupleType:     nil,
				},
				Indexed: false,
			},
		}

		// Add slice/array for our basic types.
		args = append(args, arrayArgs...)

		// Now for those slices/arrays, we create nested ones
		for _, arrayArg := range arrayArgs {
			arrayArg := arrayArg

			// Add nested slice/arrays.
			args = append(args,
				abi.Argument{
					Name: fmt.Sprintf("testSlice (%v)", arrayArg.Type.GetType().String()),
					Type: abi.Type{
						Elem:          &arrayArg.Type,
						Size:          0,
						T:             abi.SliceTy,
						TupleRawName:  "",
						TupleElems:    nil,
						TupleRawNames: nil,
						TupleType:     nil,
					},
					Indexed: false,
				},
				abi.Argument{
					Name: fmt.Sprintf("testArray (%v)", arrayArg.Type.GetType().String()),
					Type: abi.Type{
						Elem:          &arrayArg.Type,
						Size:          3,
						T:             abi.ArrayTy,
						TupleRawName:  "",
						TupleElems:    nil,
						TupleRawNames: nil,
						TupleType:     nil,
					},
					Indexed: false,
				},
			)
		}
	}

	// TODO: Add tuple argument.
	return args
}

// TestABIRoundtripEncodingAllTypes runs tests to ensure ABI value encoding works round-trip for argument values of all
// types. It generates values using a ValueGenerator, then encodes them, decodes them, and re-encodes them again to
// verify re-encoded data matches the originally encoded data.
func TestABIRoundtripEncodingAllTypes(t *testing.T) {
	// Create a value generator
	valueGenerator := NewRandomValueGenerator(&RandomValueGeneratorConfig{
		GenerateRandomArrayMinSize:  3,
		GenerateRandomArrayMaxSize:  10,
		GenerateRandomBytesMinSize:  5,
		GenerateRandomBytesMaxSize:  200,
		GenerateRandomStringMinSize: 5,
		GenerateRandomStringMaxSize: 200,
	}, rand.New(rand.NewSource(time.Now().UnixNano())))

	// Obtain our test ABI arguments
	args := getTestABIArguments()

	// Loop for each input argument
	for _, arg := range args {
		// Test each argument round trip serialization with different generated values (iterate a number of times).
		for i := 0; i < 10; i++ {
			// Generate a value for this argument
			value := GenerateAbiValue(valueGenerator, &arg.Type)

			// Encode the generated value for this argument
			encodedValue, err := encodeJSONArgument(&arg.Type, value)
			assert.NoError(t, err)

			// Decode the generated value
			decodedValue, err := decodeJSONArgument(&arg.Type, encodedValue, nil)
			assert.NoError(t, err)

			// Re-encode the generated value for this argument
			reencodedValue, err := encodeJSONArgument(&arg.Type, decodedValue)
			assert.NoError(t, err)

			// Compare the encoded and re-encoded values.
			matched := reflect.DeepEqual(encodedValue, reencodedValue)
			assert.True(t, matched, "round trip encoded->decoded->re-encoded ABI argument values did not match for '%v'.\nENCODED1: %v\nENCODED2: %v\n", arg.Name, encodedValue, reencodedValue)
		}
	}
}

// TestABIGenerationAndMutation runs tests to ABI value encoding works round-trip for argument values of all types.
// It generates values using a ValueGenerator, then encodes them, decodes them, and re-encodes them again to ensure
// re-encoded data matches the originally encoded data.
func TestABIGenerationAndMutation(t *testing.T) {
	// Create a value generator
	mutationalGeneratorConfig := &MutationalValueGeneratorConfig{
		MinMutationRounds:               0,
		MaxMutationRounds:               1,
		GenerateRandomAddressBias:       0.5,
		GenerateRandomIntegerBias:       0.5,
		GenerateRandomStringBias:        0.5,
		GenerateRandomBytesBias:         0.5,
		MutateAddressProbability:        0.8,
		MutateArrayStructureProbability: 0.8,
		MutateBoolProbability:           0.8,
		MutateBytesProbability:          0.8,
		MutateBytesGenerateNewBias:      0.45,
		MutateFixedBytesProbability:     0.8,
		MutateStringProbability:         0.8,
		MutateStringGenerateNewBias:     0.7,
		MutateIntegerProbability:        0.8,
		MutateIntegerGenerateNewBias:    0.5,
		RandomValueGeneratorConfig: &RandomValueGeneratorConfig{
			GenerateRandomArrayMinSize:  0,
			GenerateRandomArrayMaxSize:  100,
			GenerateRandomBytesMinSize:  0,
			GenerateRandomBytesMaxSize:  100,
			GenerateRandomStringMinSize: 0,
			GenerateRandomStringMaxSize: 100,
		},
	}
	mutationalGenerator := NewMutationalValueGenerator(mutationalGeneratorConfig, NewValueSet(), rand.New(rand.NewSource(time.Now().UnixNano())))

	// Obtain our test ABI arguments
	args := getTestABIArguments()

	// Loop for each input argument
	for _, arg := range args {
		// Test each argument round trip serialization with different generated values (iterate a number of times).
		for i := 0; i < 5; i++ {
			// Generate a value for this argument
			value := GenerateAbiValue(mutationalGenerator, &arg.Type)

			// Mutate and ensure no error occurred.
			mutatedValue, err := MutateAbiValue(mutationalGenerator, mutationalGenerator, &arg.Type, value)
			assert.NoError(t, err)

			// Verify the types of the value and mutated value are the same
			assert.EqualValues(t, reflect.ValueOf(value).Type().String(), reflect.ValueOf(mutatedValue).Type().String())
		}
	}
}

// TestEncodeABIArgumentToString runs tests to ensure that  a provided go-ethereum ABI packable input value of a given
// type is encoded to string in the specific format, depending on the input's type.
func TestEncodeABIArgumentToString(t *testing.T) {
	// Create a value generator
	valueGenerator := NewRandomValueGenerator(&RandomValueGeneratorConfig{
		GenerateRandomArrayMinSize:  3,
		GenerateRandomArrayMaxSize:  10,
		GenerateRandomBytesMinSize:  5,
		GenerateRandomBytesMaxSize:  200,
		GenerateRandomStringMinSize: 5,
		GenerateRandomStringMaxSize: 200,
	}, rand.New(rand.NewSource(time.Now().UnixNano())))

	// Obtain our test ABI arguments
	args := getTestABIArguments()

	// Loop for each input argument
	for _, arg := range args {
		// Test each argument encoding to string with different generated values (iterate a number of times).
		for i := 0; i < 10; i++ {
			// Generate a value for this argument
			value := GenerateAbiValue(valueGenerator, &arg.Type)

			// Encode the generated value for this argument and ensure no error occurred.
			_, err := encodeABIArgumentToString(&arg.Type, value, nil)
			assert.NoError(t, err)
		}
	}
}

// TestStringWithNullBytes verifies that strings containing null bytes are correctly serialized
// and deserialized through JSON encoding. This is a regression test for issue #279.
func TestStringWithNullBytes(t *testing.T) {
	t.Parallel()

	stringType := abi.Type{
		Elem:          nil,
		Size:          0,
		T:             abi.StringTy,
		TupleRawName:  "",
		TupleElems:    nil,
		TupleRawNames: nil,
		TupleType:     nil,
	}

	testCases := []struct {
		name  string
		input string
	}{
		{
			name:  "single null byte",
			input: "\x00",
		},
		{
			name:  "null byte at start",
			input: "\x00hello",
		},
		{
			name:  "null byte at end",
			input: "hello\x00",
		},
		{
			name:  "embedded null byte",
			input: "hello\x00world",
		},
		{
			name:  "multiple null bytes",
			input: "\x00\x00\x00",
		},
		{
			name:  "mixed null and regular characters",
			input: "a\x00b\x00c",
		},
		{
			name:  "normal string without null bytes",
			input: "hello world",
		},
		{
			name:  "empty string",
			input: "",
		},
		{
			name:  "string with other control characters",
			input: "hello\x01\x02world",
		},
		{
			name:  "string with newlines and tabs (should not be hex encoded)",
			input: "hello\nworld\ttab",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Encode the string
			encoded, err := encodeJSONArgument(&stringType, tc.input)
			assert.NoError(t, err, "encoding should not fail")

			// Decode the string
			decoded, err := decodeJSONArgument(&stringType, encoded, nil)
			assert.NoError(t, err, "decoding should not fail")

			// Verify the decoded string matches the original
			decodedStr, ok := decoded.(string)
			assert.True(t, ok, "decoded value should be a string")
			assert.Equal(t, tc.input, decodedStr, "decoded string should match original")

			// Re-encode and verify it matches the first encoding (round-trip consistency)
			reencoded, err := encodeJSONArgument(&stringType, decodedStr)
			assert.NoError(t, err, "re-encoding should not fail")
			assert.Equal(t, encoded, reencoded, "re-encoded value should match first encoding")
		})
	}
}

// TestStringNeedsHexEncoding verifies the helper function correctly identifies strings that need
// hex encoding.
func TestStringNeedsHexEncoding(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "null byte only",
			input:    "\x00",
			expected: true,
		},
		{
			name:     "embedded null byte",
			input:    "hello\x00world",
			expected: true,
		},
		{
			name:     "normal ascii string",
			input:    "hello world",
			expected: false,
		},
		{
			name:     "empty string",
			input:    "",
			expected: false,
		},
		{
			name:     "newline only",
			input:    "\n",
			expected: false,
		},
		{
			name:     "tab only",
			input:    "\t",
			expected: false,
		},
		{
			name:     "carriage return only",
			input:    "\r",
			expected: false,
		},
		{
			name:     "control character (bell)",
			input:    "\x07",
			expected: true,
		},
		{
			name:     "unicode string",
			input:    "hello \u4e16\u754c",
			expected: false,
		},
		{
			name:     "invalid utf8 sequence",
			input:    string([]byte{0xff, 0xfe}),
			expected: true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := stringNeedsHexEncoding(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}
