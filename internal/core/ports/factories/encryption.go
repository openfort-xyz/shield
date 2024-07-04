package factories

import (
	"context"
	"errors"
	"go.openfort.xyz/shield/internal/core/ports/repositories"
	"go.openfort.xyz/shield/pkg/cypher"
)

type EncryptionFactory interface {
	CreateEncryptionStrategy() EncryptionStrategy
}

type EncryptionStrategy interface {
	Encrypt(ctx context.Context, plain string) (string, error)
	Decrypt(ctx context.Context, encrypted string) (string, error)
}

type EncryptionKeyBuilder interface {
	SetEncryptionPart(ctx context.Context, part string) EncryptionKeyBuilder
	SetSessionPart(ctx context.Context, sessionID string) (EncryptionKeyBuilder, error)
	SetDatabasePart(ctx context.Context, projectID string) (EncryptionKeyBuilder, error)
	Build(ctx context.Context) (string, error)
}

type EncryptionKeyBuilderImpl struct {
	projectPart         string
	databasePart        string
	encryptionPartsRepo repositories.EncryptionPartsRepository
	projectRepo         repositories.ProjectRepository
}

func NewEncryptionKeyBuilder() EncryptionKeyBuilder {
	return &EncryptionKeyBuilderImpl{
		projectPart:  "",
		databasePart: "",
	}
}

func (b *EncryptionKeyBuilderImpl) SetEncryptionPart(ctx context.Context, part string) EncryptionKeyBuilder {
	b.projectPart = part
	return b
}

func (b *EncryptionKeyBuilderImpl) SetSessionPart(ctx context.Context, sessionID string) (EncryptionKeyBuilder, error) {
	part, err := b.encryptionPartsRepo.Get(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	b.projectPart = part
	return b, nil
}

func (b *EncryptionKeyBuilderImpl) SetDatabasePart(ctx context.Context, projectID string) (EncryptionKeyBuilder, error) {
	part, err := b.projectRepo.GetEncryptionPart(ctx, projectID)
	if err != nil {
		return nil, err
	}

	b.databasePart = part
	return b, nil
}

func (b *EncryptionKeyBuilderImpl) Build(ctx context.Context) (string, error) {
	if b.projectPart == "" {
		return "", errors.New("project part is required") // TODO extract error
	}

	if b.databasePart == "" {
		return "", errors.New("database part is required") // TODO extract error
	}

	return cypher.ReconstructEncryptionKey(b.projectPart, b.databasePart)
}
