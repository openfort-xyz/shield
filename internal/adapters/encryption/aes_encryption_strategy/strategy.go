package aesenc

import "go.openfort.xyz/shield/pkg/cypher"

type AESEncryptionStrategy struct {
	key string
}

func NewAESEncryptionStrategy(key string) *AESEncryptionStrategy {
	return &AESEncryptionStrategy{key: key}
}

func (s *AESEncryptionStrategy) Encrypt(data string) (string, error) {
	return cypher.Encrypt(data, s.key)
}

func (s *AESEncryptionStrategy) Decrypt(data string) (string, error) {
	return cypher.Decrypt(data, s.key)
}
