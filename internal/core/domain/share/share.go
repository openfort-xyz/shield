package share

type Share struct {
	ID                   string
	Secret               string
	UserID               string
	EncryptionParameters *EncryptionParameters
}

func (s *Share) RequiresEncryption() bool {
	return s.EncryptionParameters != nil && s.EncryptionParameters.Entropy == EntropyProject
}
