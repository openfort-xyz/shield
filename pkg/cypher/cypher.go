package cypher

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/codahale/sss"
	"github.com/openfort-xyz/shield/pkg/random"
)

// ErrAuthenticationFailed is returned by Decrypt when the ciphertext does not
// authenticate under the given key: the key is wrong or the data was altered.
var ErrAuthenticationFailed = errors.New("authentication failed")

func Encrypt(plaintext, key string) (string, error) {
	keyBytes, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce, err := random.GenerateRandomBytes(aesGCM.NonceSize())
	if err != nil {
		return "", err
	}

	ciphertext := aesGCM.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func Decrypt(encrypted, key string) (string, error) {
	encryptedBytes, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return "", err
	}

	keyBytes, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := aesGCM.NonceSize()
	if len(encryptedBytes) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, ciphertext := encryptedBytes[:nonceSize], encryptedBytes[nonceSize:]
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrAuthenticationFailed, err)
	}

	return string(plaintext), nil
}

func SplitEncryptionKey(key string) (string, string, error) {
	rawKey, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return "", "", err
	}

	shares, err := sss.Split(2, 2, rawKey)
	if err != nil {
		return "", "", err
	}

	if len(shares) != 2 {
		return "", "", errors.New("expected 2 shares")
	}

	subset := make([][]byte, 0)
	for _, share := range shares {
		subset = append(subset, share)
	}

	return base64.StdEncoding.EncodeToString(subset[0]), base64.StdEncoding.EncodeToString(subset[1]), nil
}

func ReconstructEncryptionKey(part1, part2 string) (string, error) {
	rawPart1, err := base64.StdEncoding.DecodeString(part1)
	if err != nil {
		return "", err
	}
	rawPart2, err := base64.StdEncoding.DecodeString(part2)
	if err != nil {
		return "", err
	}

	subset := make(map[byte][]byte, 2)
	subset[0] = rawPart1
	subset[1] = rawPart2

	key := sss.Combine(subset)

	return base64.StdEncoding.EncodeToString(key), nil
}

const keyCheckDomain = "openfort-shield/encryption-key-check/v1"

// KeyCheck returns a value that identifies an encryption key without
// revealing it: hex(HMAC-SHA256(key, domain)). Storing it lets a
// reconstructed key be verified before use, which Shamir combine cannot do.
func KeyCheck(key string) (string, error) {
	keyBytes, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, keyBytes)
	mac.Write([]byte(keyCheckDomain))
	return hex.EncodeToString(mac.Sum(nil)), nil
}

// VerifyKeyCheck reports in constant time whether key produces keyCheck.
func VerifyKeyCheck(key, keyCheck string) bool {
	got, err := KeyCheck(key)
	if err != nil {
		return false
	}
	return hmac.Equal([]byte(got), []byte(keyCheck))
}
