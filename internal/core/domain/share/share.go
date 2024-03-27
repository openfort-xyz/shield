package share

type Share struct {
	ID          string
	Data        string
	UserID      string
	UserEntropy bool
	Salt        string
	Iterations  int
	Length      int
	Digest      string
}
