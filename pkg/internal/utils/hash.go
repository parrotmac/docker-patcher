package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
)

func CalculateFileSha256Sum(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	sha := sha256.New()
	if _, err := io.Copy(sha, f); err != nil {
		if err != nil {
			return "", err
		}
	}
	return hex.EncodeToString(sha.Sum(nil)), nil
}
