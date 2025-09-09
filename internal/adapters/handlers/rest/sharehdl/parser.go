package sharehdl

import (
	"go.openfort.xyz/shield/internal/core/domain/share"
)

type parser struct {
	mapEntropyDomain       map[Entropy]share.Entropy
	mapDomainEntropy       map[share.Entropy]Entropy
	mapStorageMethodDomain map[ShareStorageMethodID]share.StorageMethodID
	mapDomainStorageMethod map[share.StorageMethodID]ShareStorageMethodID
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
		mapStorageMethodDomain: map[ShareStorageMethodID]share.StorageMethodID{
			StorageMethodShield:      share.StorageMethodShield,
			StorageMethodGoogleDrive: share.StorageMethodGoogleDrive,
			StorageMethodICloud:      share.StorageMethodICloud,
		},
		mapDomainStorageMethod: map[share.StorageMethodID]ShareStorageMethodID{
			share.StorageMethodShield:      StorageMethodShield,
			share.StorageMethodGoogleDrive: StorageMethodGoogleDrive,
			share.StorageMethodICloud:      StorageMethodICloud,
		},
	}
}

func (p *parser) toPasskeyEnv(s *string) *PasskeyEnv {
	if s != nil {
		matches := share.PasskeyEnvPattern.FindStringSubmatch(*s)
		if matches != nil {
			return &PasskeyEnv{
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
	shr := &share.Share{
		Secret:               s.Secret,
		Entropy:              p.mapEntropyDomain[s.Entropy],
		ShareStorageMethodID: p.mapStorageMethodDomain[s.ShareStorageMethodID],
	}

	if s.KeychainID != "" {
		shr.KeychainID = &s.KeychainID
	}

	if s.Reference != "" {
		shr.Reference = &s.Reference
	}

	if s.EncryptionPart != "" || s.EncryptionSession != "" {
		shr.Entropy = share.EntropyProject
	}

	if s.Salt != "" {
		if shr.EncryptionParameters == nil {
			shr.EncryptionParameters = new(share.EncryptionParameters)
		}
		shr.EncryptionParameters.Salt = s.Salt
	}

	if s.Iterations != 0 {
		if shr.EncryptionParameters == nil {
			shr.EncryptionParameters = new(share.EncryptionParameters)
		}
		shr.EncryptionParameters.Iterations = s.Iterations
	}

	if s.Length != 0 {
		if shr.EncryptionParameters == nil {
			shr.EncryptionParameters = new(share.EncryptionParameters)
		}
		shr.EncryptionParameters.Length = s.Length
	}

	if s.Digest != "" {
		if shr.EncryptionParameters == nil {
			shr.EncryptionParameters = new(share.EncryptionParameters)
		}
		shr.EncryptionParameters.Digest = s.Digest
	}

	if shr.EncryptionParameters != nil {
		shr.Entropy = share.EntropyUser
	}

	if s.Entropy == EntropyPasskey && s.PasskeyReference != nil {
		shr.PasskeyReference = &share.PasskeyReference{
			PasskeyID: *s.PasskeyReference.PasskeyId,
		}
		if s.PasskeyReference.PasskeyEnv != nil {
			shr.PasskeyReference.PasskeyEnv = &share.PasskeyEnv{
				Name:      s.PasskeyReference.PasskeyEnv.Name,
				OS:        s.PasskeyReference.PasskeyEnv.OS,
				OSVersion: s.PasskeyReference.PasskeyEnv.OSVersion,
				Device:    s.PasskeyReference.PasskeyEnv.Device,
			}
		}
	}

	return shr
}

func (p *parser) fromDomain(s *share.Share) *Share {
	shr := &Share{
		Secret:               s.Secret,
		Entropy:              p.mapDomainEntropy[s.Entropy],
		ShareStorageMethodID: p.mapDomainStorageMethod[s.ShareStorageMethodID],
	}

	if s.KeychainID != nil {
		shr.KeychainID = *s.KeychainID
	}

	if s.Reference != nil {
		shr.Reference = *s.Reference
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

	if s.PasskeyReference != nil {
		shr.PasskeyReference = &PasskeyReference{
			PasskeyId: &s.PasskeyReference.PasskeyID,
		}
		if s.PasskeyReference.PasskeyEnv != nil {
			shr.PasskeyReference.PasskeyEnv = &PasskeyEnv{
				Name:      s.PasskeyReference.PasskeyEnv.Name,
				OS:        s.PasskeyReference.PasskeyEnv.OS,
				OSVersion: s.PasskeyReference.PasskeyEnv.OSVersion,
				Device:    s.PasskeyReference.PasskeyEnv.Device,
			}
		}
	}

	return shr
}

func (p *parser) fromDomainShareStorageMethod(s *share.StorageMethod) *ShareStorageMethod {
	return &ShareStorageMethod{
		ID:   s.ID,
		Name: s.Name,
	}
}
