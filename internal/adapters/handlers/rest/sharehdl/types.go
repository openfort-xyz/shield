package sharehdl

const EncryptionPartHeader = "X-Encryption-Part"

type Share struct {
	Secret         string  `json:"secret"`
	Entropy        Entropy `json:"entropy"`
	Salt           string  `json:"salt,omitempty"`
	Iterations     int     `json:"iterations,omitempty"`
	Length         int     `json:"length,omitempty"`
	Digest         string  `json:"digest,omitempty"`
	EncryptionPart string  `json:"encryption_part,omitempty"`
}

type RegisterShareRequest Share
type GetShareResponse Share

type Entropy string

const (
	EntropyNone    Entropy = "none"
	EntropyUser    Entropy = "user"
	EntropyProject Entropy = "project"
)
