package share

type Share struct {
	ID                   string
	Secret               string
	UserID               string
	KeychainID           *string
	Reference            *string
	Entropy              Entropy
	ShareStorageMethodID StorageMethodID
	EncryptionParameters *EncryptionParameters
	PasskeyReference     *PasskeyReference
}

func (s *Share) RequiresEncryption() bool {
	return s.Entropy == EntropyProject
}

const DefaultReference = "default"
