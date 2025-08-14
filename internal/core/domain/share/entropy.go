package share

type Entropy int8

const (
	EntropyNone Entropy = iota + 1
	EntropyUser
	EntropyProject
	EntropyPasskey
)
