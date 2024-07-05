package strategies

type EncryptionStrategy interface {
	Encrypt(data string) (string, error)
	Decrypt(data string) (string, error)
}
