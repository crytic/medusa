package utils

import (
	"crypto/ecdsa"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/pkg/errors"
)

// GetPrivateKey will return a private key object given a byte slice. Only slices between lengths 1 and 32 (inclusive)
// are valid.
func GetPrivateKey(b []byte) (*ecdsa.PrivateKey, error) {
	// Make sure that private key is not zero
	if len(b) < 1 || len(b) > 32 {
		return nil, errors.New("invalid private key")
	}

	// Then pad the private key slice to a fixed 32-byte array
	paddedPrivateKey := make([]byte, 32)
	copy(paddedPrivateKey[32-len(b):], b)

	// Next we will actually retrieve the private key object
	privateKey, err := crypto.ToECDSA(paddedPrivateKey[:])
	return privateKey, errors.WithStack(err)
}
