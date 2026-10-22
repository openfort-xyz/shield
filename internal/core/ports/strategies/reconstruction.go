package strategies

import "github.com/openfort-xyz/shield/internal/core/domain/project"

type ReconstructionStrategy interface {
	Split(key string) (project.StoredPart, project.ProjectPart, error)
	Reconstruct(storedPart project.StoredPart, projectPart project.ProjectPart) (string, error)
}
