package campaign

import (
	"crypto/rand"
	"encoding/hex"
)

// IDGenerator is injectable so service failures and IDs can be tested without
// weakening the production source of randomness.
type IDGenerator func() (string, error)

// NewCryptoIDGenerator returns opaque campaign IDs made from 128 random bits.
func NewCryptoIDGenerator() IDGenerator {
	return func() (string, error) {
		bytes := make([]byte, 16)
		if _, err := rand.Read(bytes); err != nil {
			return "", err
		}
		return "c_" + hex.EncodeToString(bytes), nil
	}
}
