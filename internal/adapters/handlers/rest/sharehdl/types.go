package sharehdl

const EncryptionPartHeader = "X-Encryption-Part"
const EncryptionSessionHeader = "X-Encryption-Session"

type Share struct {
	Secret               string               `json:"secret"`
	Entropy              Entropy              `json:"entropy"`
	Salt                 string               `json:"salt,omitempty"`
	Iterations           int                  `json:"iterations,omitempty"`
	Length               int                  `json:"length,omitempty"`
	Digest               string               `json:"digest,omitempty"`
	EncryptionPart       string               `json:"encryption_part,omitempty"`
	EncryptionSession    string               `json:"encryption_session,omitempty"`
	Reference            string               `json:"reference,omitempty"`
	ShareStorageMethodID ShareStorageMethodID `json:"storage_method_id,omitempty"`
	KeychainID           string               `json:"keychain_id,omitempty"`
}

type RegisterShareRequest Share
type GetShareResponse Share
type UpdateShareRequest Share
type UpdateShareResponse Share

type Entropy string

const (
	EntropyNone    Entropy = "none"
	EntropyUser    Entropy = "user"
	EntropyProject Entropy = "project"
	EntropyPasskey Entropy = "passkey"
)

type ShareStorageMethodID int32

const (
	StorageMethodShield ShareStorageMethodID = iota
	StorageMethodGoogleDrive
	StorageMethodICloud
)

func (s ShareStorageMethodID) IsValid() bool {
	return s == StorageMethodShield || s == StorageMethodGoogleDrive || s == StorageMethodICloud
}

type GetShareEncryptionResponse struct {
	Entropy    Entropy `json:"entropy"`
	Salt       *string `json:"salt,omitempty"`
	Iterations *int    `json:"iterations,omitempty"`
	Length     *int    `json:"length,omitempty"`
	Digest     *string `json:"digest,omitempty"`
}

type KeychainResponse struct {
	Shares []*Share `json:"shares"`
}

type ShareStorageMethod struct {
	ID   int32  `json:"id"`
	Name string `json:"name"`
}

type GetShareStorageMethodsResponse struct {
	Methods []*ShareStorageMethod `json:"methods"`
}

type GetSharesEncryptionForReferencesRequest struct {
	References []string `json:"references"`
}

type EncryptionTypeStatus string

const (
	EncryptionTypeStatusNotFound EncryptionTypeStatus = "not-found"
	EncryptionTypeStatusFound    EncryptionTypeStatus = "found"
)

type EncryptionTypeResponse struct {
	Status         EncryptionTypeStatus `json:"status"`
	EncryptionType *Entropy             `json:"encryption_type"`
}

type GetSharesEncryptionForReferencesResponse struct {
	EncryptionTypes map[string]EncryptionTypeResponse `json:"encryption_types"`
}

type GetSharesEncryptionForUsersRequest struct {
	UserIDs []string `json:"user_ids"`
}

type GetSharesEncryptionForUsersResponse struct {
	EncryptionTypes map[string]EncryptionTypeResponse `json:"encryption_types"`
}
