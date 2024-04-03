package cypher

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"github.com/codahale/sss"
	"io"
)

func generateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := io.ReadFull(rand.Reader, b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func generateRandomString(n int) (string, error) {
	b, err := generateRandomBytes(n)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

func Encrypt(plaintext, share1, share2 string) (string, error) {
	key, err := combineShares(share1, share2)
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

	nonce, err := generateRandomBytes(aesGCM.NonceSize())
	if err != nil {
		return "", err
	}

	ciphertext := aesGCM.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func Decrypt(encrypted, share1, share2 string) (string, error) {
	key, err := combineShares(share1, share2)
	if err != nil {
		return "", err
	}

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
		return "", err
	}

	nonce, ciphertext := encryptedBytes[:nonceSize], encryptedBytes[nonceSize:]
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

func GenerateEncryptionKey() (string, string, error) {
	key, err := generateRandomString(32)
	if err != nil {
		return "", "", err
	}

	return splitKey(key)
}

func splitKey(key string) (string, string, error) {
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

func combineShares(share1, share2 string) (string, error) {
	rawShare1, err := base64.StdEncoding.DecodeString(share1)
	if err != nil {
		return "", err
	}
	rawShare2, err := base64.StdEncoding.DecodeString(share2)
	if err != nil {
		return "", err
	}

	subset := make(map[byte][]byte, 2)
	subset[0] = rawShare1
	subset[1] = rawShare2

	key := sss.Combine(subset)

	return base64.StdEncoding.EncodeToString(key), nil
}
