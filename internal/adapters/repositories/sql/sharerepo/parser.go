package sharerepo

import (
	"fmt"

	"go.openfort.xyz/shield/internal/core/domain/share"
)

type parser struct {
	mapEntropyDomain       map[Entropy]share.Entropy
	mapDomainEntropy       map[share.Entropy]Entropy
	mapDomainStorageMethod map[share.StorageMethodID]ShareStorageMethodID
	mapStorageMethodDomain map[ShareStorageMethodID]share.StorageMethodID
}

func newParser() *parser {
	return &parser{
		mapEntropyDomain: map[Entropy]share.Entropy{
			EntropyNone:    share.EntropyNone,
			EntropyUser:    share.EntropyUser,
			EntropyProject: share.EntropyProject,
			EntropyPasskey: share.EntropyPasskey,
		},
		mapDomainEntropy: map[share.Entropy]Entropy{
			share.EntropyNone:    EntropyNone,
			share.EntropyUser:    EntropyUser,
			share.EntropyProject: EntropyProject,
			share.EntropyPasskey: EntropyPasskey,
		},
		mapDomainStorageMethod: map[share.StorageMethodID]ShareStorageMethodID{
			share.StorageMethodShield:      StorageMethodShield,
			share.StorageMethodGoogleDrive: StorageMethodGoogleDrive,
			share.StorageMethodICloud:      StorageMethodICloud,
		},
		mapStorageMethodDomain: map[ShareStorageMethodID]share.StorageMethodID{
			StorageMethodShield:      share.StorageMethodShield,
			StorageMethodGoogleDrive: share.StorageMethodGoogleDrive,
			StorageMethodICloud:      share.StorageMethodICloud,
		},
	}
}

func databaseToEnv(s *string) *share.PasskeyEnv {
	if s != nil {
		matches := share.PasskeyEnvPattern.FindStringSubmatch(*s)
		if matches != nil {
			return &share.PasskeyEnv{
				Name:      &matches[1],
				OS:        &matches[2],
				OSVersion: &matches[3],
				Device:    &matches[4],
			}
		}
		return nil
	}
	return nil
}

func (p *parser) toDomain(s *Share) *share.Share {
	var encryptionParameters *share.EncryptionParameters

	if s.Salt != "" {
		encryptionParameters = new(share.EncryptionParameters)
		encryptionParameters.Salt = s.Salt
	}

	if s.Iterations != 0 {
		if encryptionParameters == nil {
			encryptionParameters = new(share.EncryptionParameters)
		}
		encryptionParameters.Iterations = s.Iterations
	}

	if s.Length != 0 {
		if encryptionParameters == nil {
			encryptionParameters = new(share.EncryptionParameters)
		}
		encryptionParameters.Length = s.Length
	}

	if s.Digest != "" {
		if encryptionParameters == nil {
			encryptionParameters = new(share.EncryptionParameters)
		}
		encryptionParameters.Digest = s.Digest
	}

	var passkeyReference *share.PasskeyReference
	if s.Entropy == EntropyPasskey && s.PasskeyReference != nil {
		passkeyReference = &share.PasskeyReference{
			PasskeyID:  s.PasskeyReference.PasskeyID,
			PasskeyEnv: databaseToEnv(s.PasskeyReference.PasskeyEnv),
		}
	}

	usrID := ""
	if s.UserID != nil {
		usrID = *s.UserID
	}
	return &share.Share{
		ID:                   s.ID,
		Secret:               s.Data,
		UserID:               usrID,
		Entropy:              p.mapEntropyDomain[s.Entropy],
		EncryptionParameters: encryptionParameters,
		KeychainID:           s.KeyChainID,
		Reference:            s.Reference,
		ShareStorageMethodID: p.mapStorageMethodDomain[s.ShareStorageMethodID],
		PasskeyReference:     passkeyReference,
	}
}

func coalesceToUnknown(s *string) string {
	if s == nil {
		return "unknown"
	}
	return *s
}

func envToDatabase(p *share.PasskeyEnv) *string {
	if p != nil {
		ret := fmt.Sprintf(
			"name=%s;os=%s;osVersion=%s;device=%s",
			coalesceToUnknown(p.Name),
			coalesceToUnknown(p.OS),
			coalesceToUnknown(p.OSVersion),
			coalesceToUnknown(p.Device),
		)
		return &ret
	}
	return nil
}

func (p *parser) toDatabase(s *share.Share) *Share {
	var usrID *string
	if s.UserID != "" {
		usrID = &s.UserID
	}
	shr := &Share{
		ID:                   s.ID,
		Data:                 s.Secret,
		UserID:               usrID,
		KeyChainID:           s.KeychainID,
		Reference:            s.Reference,
		ShareStorageMethodID: p.mapDomainStorageMethod[s.ShareStorageMethodID],
		Entropy:              p.mapDomainEntropy[s.Entropy],
	}

	if s.EncryptionParameters != nil {
		if s.EncryptionParameters.Salt != "" {
			shr.Salt = s.EncryptionParameters.Salt
		}
		if s.EncryptionParameters.Iterations != 0 {
			shr.Iterations = s.EncryptionParameters.Iterations
		}
		if s.EncryptionParameters.Length != 0 {
			shr.Length = s.EncryptionParameters.Length
		}
		if s.EncryptionParameters.Digest != "" {
			shr.Digest = s.EncryptionParameters.Digest
		}
	}

	if s.Entropy == share.EntropyPasskey && s.PasskeyReference != nil {
		shr.PasskeyReference = &PasskeyReference{
			PasskeyID:      s.PasskeyReference.PasskeyID,
			PasskeyEnv:     envToDatabase(s.PasskeyReference.PasskeyEnv),
			ShareReference: s.ID,
		}
	}

	return shr
}

func (p *parser) toDomainShareStorageMethod(dbMethod *ShareStorageMethod) *share.StorageMethod {
	return &share.StorageMethod{
		ID:   dbMethod.ID,
		Name: dbMethod.Name,
	}
}
