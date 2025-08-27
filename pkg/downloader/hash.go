package downloader

import (
	"crypto/sha256"
	"encoding/hex"
)

// HashString generates a 12-character SHA256 hash
// from a given string.
// It should not be used for cryptographic operations.
func HashString(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])[:12]
}
