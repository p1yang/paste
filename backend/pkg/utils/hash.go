package utils

import (
	"crypto/sha256"
	"encoding/hex"
)

func ComputeHash(data []byte) string {
	h := sha256.New()
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

func ComputeTextHash(text string) string {
	return ComputeHash([]byte(text))
}
