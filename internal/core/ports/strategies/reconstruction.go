package strategies

type ReconstructionStrategy interface {
	Split(data string) (storedPart string, projectPart string, err error)
	Reconstruct(storedPart string, projectPart string) (string, error)
}
