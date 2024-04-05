package share

type EncryptionParameters struct {
	Entropy    Entropy
	Salt       string
	Iterations int
	Length     int
	Digest     string
}
