package share

type Entropy int8

const (
	EntropyNone Entropy = iota
	EntropyUser
	EntropyProject
)
