package config

import (
	"encoding/hex"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/crytic/medusa-geth/crypto"
)

// customErrorSignatureRegex matches a custom error signature such as `MyError()` or `MyError(uint256,(address,bool))`,
// capturing the parameter list. It only verifies the error name is a valid Solidity identifier followed by a
// parenthesized parameter list. The parameter list itself is parsed and validated by canonicalizeTypeList.
var customErrorSignatureRegex = regexp.MustCompile(`^[a-zA-Z_$][a-zA-Z0-9_$]*\((.*)\)$`)

// arraySuffixRegex matches zero or more array dimensions following a type, such as `[]`, `[3]` or `[2][]`.
var arraySuffixRegex = regexp.MustCompile(`^(\[([1-9][0-9]*)?\])*$`)

// sizedElementaryTypeRegex matches sized elementary ABI types, capturing the type prefix and its size(s), such as
// `uint256` (prefix `uint`, size `256`), `bytes4` (prefix `bytes`, size `4`), or `fixed128x18` (prefix `fixed`, sizes
// `128` and `18`).
var sizedElementaryTypeRegex = regexp.MustCompile(`^(uint|int|bytes|fixed|ufixed)([0-9]+)(?:x([0-9]+))?$`)

// typeAliases maps Solidity elementary type aliases to the canonical type names used when computing selectors.
// Reference: https://docs.soliditylang.org/en/latest/abi-spec.html#types
var typeAliases = map[string]string{
	"uint":   "uint256",
	"int":    "int256",
	"byte":   "bytes1",
	"fixed":  "fixed128x18",
	"ufixed": "ufixed128x18",
}

// ParseCustomErrorSelector parses a single FailOnCustomErrors entry and returns the 4-byte error selector it
// represents. The entry may either be a 4-byte selector encoded as hex with a `0x` prefix, or a custom error signature,
// in which case the selector is computed as the first 4 bytes of the Keccak256 hash of its canonical form. Type aliases
// in the signature (e.g. `uint` or `byte`) are canonicalized, including within arrays and tuples. Returns an error if
// the entry is malformed or contains a parameter type which cannot be resolved to a canonical ABI type.
func ParseCustomErrorSelector(customError string) ([4]byte, error) {
	var selector [4]byte

	// If the entry is hex-prefixed, we treat it as a raw selector and decode it directly.
	if strings.HasPrefix(customError, "0x") || strings.HasPrefix(customError, "0X") {
		decoded, err := hex.DecodeString(customError[2:])
		if err != nil || len(decoded) != len(selector) {
			return selector, fmt.Errorf("invalid custom error selector '%s': selectors must be exactly 4 bytes of hex data (e.g. 0xcf479181)", customError)
		}
		copy(selector[:], decoded)
		return selector, nil
	}

	// Otherwise, we treat the entry as an error signature and compute its selector from its canonical form.
	canonicalSignature, err := canonicalizeCustomErrorSignature(customError)
	if err != nil {
		return selector, fmt.Errorf("invalid custom error signature '%s': %w", customError, err)
	}
	copy(selector[:], crypto.Keccak256([]byte(canonicalSignature))[:len(selector)])
	return selector, nil
}

// canonicalizeCustomErrorSignature takes a custom error signature and returns its canonical ABI form, which is the
// form the compiler hashes to produce the error's selector. Any signature which cannot be canonicalized is rejected
// rather than hashed as-is, as it would otherwise produce a selector which silently never matches.
func canonicalizeCustomErrorSignature(signature string) (string, error) {
	// Whitespace is rejected rather than stripped, as it most likely indicates parameter names were included, which
	// would otherwise be mistaken for part of a type.
	if strings.ContainsAny(signature, " \t\r\n") {
		return "", fmt.Errorf("signatures must not contain parameter names or whitespace (e.g. InsufficientBalance(uint256,uint256))")
	}

	// Split the signature into the error name and its parameter list.
	matches := customErrorSignatureRegex.FindStringSubmatch(signature)
	if matches == nil {
		return "", fmt.Errorf("signatures must be an error name followed by a parenthesized parameter list (e.g. InsufficientBalance(uint256,uint256))")
	}
	name := signature[:len(signature)-len(matches[1])-2]

	// Canonicalize the parameter list.
	params, err := canonicalizeTypeList(matches[1])
	if err != nil {
		return "", err
	}
	return name + "(" + params + ")", nil
}

// canonicalizeTypeList takes a comma-separated list of ABI types (the contents of a parameter list or tuple) and
// returns the list with each type canonicalized. An empty list is returned as-is.
func canonicalizeTypeList(typeList string) (string, error) {
	if typeList == "" {
		return "", nil
	}

	// Split the list on commas which are not nested within a tuple, tracking parenthesis depth as we go.
	types := make([]string, 0)
	depth, start := 0, 0
	for i, c := range typeList {
		switch c {
		case '(':
			depth++
		case ')':
			depth--
			if depth < 0 {
				return "", fmt.Errorf("unbalanced parentheses in parameter list '%s'", typeList)
			}
		case ',':
			if depth == 0 {
				types = append(types, typeList[start:i])
				start = i + 1
			}
		}
	}
	if depth != 0 {
		return "", fmt.Errorf("unbalanced parentheses in parameter list '%s'", typeList)
	}
	types = append(types, typeList[start:])

	// Canonicalize each type in the list.
	for i, typ := range types {
		canonicalType, err := canonicalizeType(typ)
		if err != nil {
			return "", err
		}
		types[i] = canonicalType
	}
	return strings.Join(types, ","), nil
}

// canonicalizeType takes a single ABI type, which may be an elementary type, a tuple, or an array of either, and
// returns its canonical form. Elementary type aliases are replaced with their canonical names, and tuples are
// canonicalized recursively. Returns an error if the type is not a valid ABI type.
func canonicalizeType(typ string) (string, error) {
	if typ == "" {
		return "", fmt.Errorf("empty parameter type")
	}

	// If this is a tuple, find its matching closing parenthesis, canonicalize its components recursively, and retain
	// any array dimensions which follow it.
	if typ[0] == '(' {
		depth, closeIndex := 0, -1
		for i, c := range typ {
			if c == '(' {
				depth++
			} else if c == ')' {
				depth--
				if depth == 0 {
					closeIndex = i
					break
				}
			}
		}
		if closeIndex == -1 {
			return "", fmt.Errorf("unbalanced parentheses in tuple type '%s'", typ)
		}
		components, suffix := typ[1:closeIndex], typ[closeIndex+1:]
		if components == "" {
			return "", fmt.Errorf("tuple type '%s' must have at least one component", typ)
		}
		if !arraySuffixRegex.MatchString(suffix) {
			return "", fmt.Errorf("invalid array dimensions '%s' in tuple type '%s'", suffix, typ)
		}
		canonicalComponents, err := canonicalizeTypeList(components)
		if err != nil {
			return "", err
		}
		return "(" + canonicalComponents + ")" + suffix, nil
	}

	// Otherwise, this is an elementary type, optionally followed by array dimensions. Replace any alias in the base type.
	base, suffix := typ, ""
	if bracketIndex := strings.IndexByte(typ, '['); bracketIndex != -1 {
		base, suffix = typ[:bracketIndex], typ[bracketIndex:]
	}
	if !arraySuffixRegex.MatchString(suffix) {
		return "", fmt.Errorf("invalid array dimensions '%s' in type '%s'", suffix, typ)
	}
	if canonicalBase, isAlias := typeAliases[base]; isAlias {
		base = canonicalBase
	}

	// Verify the resulting base type is a valid elementary ABI type. User-defined types (structs, enums, contracts) are
	// not valid here, as the compiler replaces them with their underlying ABI types when computing selectors.
	if !isElementaryABIType(base) {
		return "", fmt.Errorf("unsupported parameter type '%s': use canonical ABI types (e.g. a tuple for a struct, uint8 for an enum, address for a contract)", typ)
	}
	return base + suffix, nil
}

// isElementaryABIType determines whether the provided type name is a canonical elementary ABI type, as defined in the
// Solidity ABI specification: https://docs.soliditylang.org/en/latest/abi-spec.html#types
func isElementaryABIType(typ string) bool {
	// Check for unsized elementary types.
	switch typ {
	case "address", "bool", "string", "bytes", "function":
		return true
	}

	// Check for sized elementary types and verify their sizes are within the bounds the ABI specification allows.
	matches := sizedElementaryTypeRegex.FindStringSubmatch(typ)
	if matches == nil {
		return false
	}
	prefix, hasDecimals := matches[1], matches[3] != ""
	size, err := strconv.Atoi(matches[2])
	if err != nil {
		return false
	}
	switch prefix {
	case "uint", "int":
		// uint<M>/int<M>: 0 < M <= 256, M % 8 == 0
		return !hasDecimals && size > 0 && size <= 256 && size%8 == 0
	case "bytes":
		// bytes<M>: 0 < M <= 32
		return !hasDecimals && size > 0 && size <= 32
	default:
		// fixed<M>x<N>/ufixed<M>x<N>: 8 <= M <= 256, M % 8 == 0, 0 < N <= 80
		if !hasDecimals {
			return false
		}
		decimals, err := strconv.Atoi(matches[3])
		return err == nil && size >= 8 && size <= 256 && size%8 == 0 && decimals > 0 && decimals <= 80
	}
}
