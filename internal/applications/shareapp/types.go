package shareapp

type EncryptionParameters struct {
	Salt       string
	Iterations int
	Length     int
	Digest     string
}
