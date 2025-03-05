package share

type Share struct {
	ID                   string
	Secret               string
	UserID               string
	KeychainID           *string
	Reference            *string
	Entropy              Entropy
	EncryptionParameters *EncryptionParameters
}

func (s *Share) RequiresEncryption() bool {
	return s.Entropy == EntropyProject
}

const DefaultReference = "default"
