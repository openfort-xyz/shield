package random

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"
)

// UUIDv7 generates a UUIDv7, code from https://antonz.org/uuidv7
func UUIDv7() (string, error) {
	var value [16]byte
	_, err := rand.Read(value[:])
	if err != nil {
		return "", err
	}

	timestamp := big.NewInt(time.Now().UnixMilli())
	timestamp.FillBytes(value[0:6])

	value[6] = (value[6] & 0x0F) | 0x70
	value[8] = (value[8] & 0x3F) | 0x80
	return fmt.Sprintf("%x", value), nil
}
