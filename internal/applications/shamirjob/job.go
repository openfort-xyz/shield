package shamirjob

import (
	"context"
	"log/slog"
	"sync"

	aesenc "go.openfort.xyz/shield/internal/adapters/encryption/aes_encryption_strategy"
	sssrec "go.openfort.xyz/shield/internal/adapters/encryption/sss_reconstruction_strategy"
	"go.openfort.xyz/shield/internal/core/domain/share"
	"go.openfort.xyz/shield/internal/core/ports/repositories"
	"go.openfort.xyz/shield/internal/core/ports/strategies"
	"go.openfort.xyz/shield/pkg/logger"
)

type Job struct {
	projectRepo            repositories.ProjectRepository
	shareRepo              repositories.ShareRepository
	reconstructionStrategy strategies.ReconstructionStrategy
	logger                 *slog.Logger
	mu                     sync.Mutex
}

func New(projectRepo repositories.ProjectRepository, shareRepo repositories.ShareRepository) *Job {
	return &Job{
		projectRepo:            projectRepo,
		shareRepo:              shareRepo,
		reconstructionStrategy: sssrec.NewSSSReconstructionStrategy(),
		logger:                 logger.New("shamirjob"),
	}
}

func (j *Job) Execute(ctx context.Context, projectID string, storedPart, projectPart, key string) (err error) {
	j.mu.Lock()
	defer j.mu.Unlock()

	j.logger.InfoContext(ctx, "executing job", slog.String("project_id", projectID))

	isMigrated, err := j.projectRepo.HasSuccessfulMigration(ctx, projectID)
	if err != nil {
		j.logger.ErrorContext(ctx, "error checking migration", logger.Error(err))
		return err
	}

	if isMigrated {
		j.logger.InfoContext(ctx, "project already migrated")
		return nil
	}

	defer func() {
		success := err == nil
		err = j.projectRepo.CreateMigration(ctx, projectID, success)
		if err != nil {
			j.logger.ErrorContext(ctx, "error creating migration", logger.Error(err))
		}
	}()

	j.logger.InfoContext(ctx, "reconstructing key")
	reconstructedKey, err := j.reconstructionStrategy.Reconstruct(storedPart, projectPart)
	if err != nil {
		j.logger.ErrorContext(ctx, "error reconstructing key", logger.Error(err))
		return err
	}

	decryptStrategy := aesenc.NewAESEncryptionStrategy(key)
	encryptStrategy := aesenc.NewAESEncryptionStrategy(reconstructedKey)

	j.logger.InfoContext(ctx, "loading shares")
	shares, err := j.shareRepo.ListProjectIDAndEntropy(ctx, projectID, share.EntropyProject)
	if err != nil {
		return err
	}
	j.logger.InfoContext(ctx, "loaded shares", slog.Int("count", len(shares)))

	j.logger.InfoContext(ctx, "re-encrypting shares")
	for _, shr := range shares {
		decr, err := decryptStrategy.Decrypt(shr.Secret)
		if err != nil {
			j.logger.ErrorContext(ctx, "error decrypting", logger.Error(err))
			return err
		}

		encr, err := encryptStrategy.Encrypt(decr)
		if err != nil {
			j.logger.ErrorContext(ctx, "error encrypting", logger.Error(err))
			return err
		}

		shr.Secret = encr
	}

	j.logger.InfoContext(ctx, "updating shares")
	err = j.shareRepo.BulkUpdate(ctx, shares)
	if err != nil {
		j.logger.ErrorContext(ctx, "error updating shares", logger.Error(err))
		return err
	}

	return nil
}
