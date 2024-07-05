package strategies

type ReconstructionStrategy interface {
	Split(data string) ([]string, error)
	Reconstruct(parts []string) (string, error)
}
