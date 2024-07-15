package cypher

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"errors"

	"github.com/codahale/sss"
	"go.openfort.xyz/shield/pkg/random"
)

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
		return "", err
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
