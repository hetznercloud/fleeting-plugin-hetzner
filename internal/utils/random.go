package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

func GenerateRandomID() string {
	b := make([]byte, 4)
	_, err := rand.Read(b)
	if err != nil {
		panic(fmt.Errorf("failed to generate random string: %w", err))
	}
	return hex.EncodeToString(b)
}
