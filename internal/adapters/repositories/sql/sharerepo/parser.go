package sharerepo

import (
	"fmt"

	"go.openfort.xyz/shield/internal/core/domain/share"
	"gorm.io/gorm"
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
	if s.PasskeyReference != nil {
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

	if s.PasskeyReference != nil {
		shr.PasskeyReference = &PasskeyReference{
			PasskeyID:  s.PasskeyReference.PasskeyID,
			PasskeyEnv: envToDatabase(s.PasskeyReference.PasskeyEnv),
		}
	}

	return shr
}

func (p *parser) toUpdates(s *share.Share) map[string]interface{} {
	updates := make(map[string]interface{})

	if s.KeychainID != nil {
		updates["keychain_id"] = s.KeychainID
	}

	if s.Reference != nil {
		updates["reference"] = s.Reference
	}

	if s.Secret != "" {
		updates["data"] = s.Secret
	}

	if s.Entropy != 0 {
		updates["entropy"] = p.mapDomainEntropy[s.Entropy]
	}

	if s.Entropy != share.EntropyUser {
		updates["salt"] = gorm.Expr("NULL")
		updates["iterations"] = gorm.Expr("NULL")
		updates["length"] = gorm.Expr("NULL")
		updates["digest"] = gorm.Expr("NULL")
	}

	if s.EncryptionParameters != nil && s.Entropy == share.EntropyUser {
		if s.EncryptionParameters.Salt != "" {
			updates["salt"] = s.EncryptionParameters.Salt
		}
		if s.EncryptionParameters.Iterations != 0 {
			updates["iterations"] = s.EncryptionParameters.Iterations
		}
		if s.EncryptionParameters.Length != 0 {
			updates["length"] = s.EncryptionParameters.Length
		}
		if s.EncryptionParameters.Digest != "" {
			updates["digest"] = s.EncryptionParameters.Digest
		}
	}

	return updates
}

func (p *parser) toDomainShareStorageMethod(dbMethod *ShareStorageMethod) *share.StorageMethod {
	return &share.StorageMethod{
		ID:   dbMethod.ID,
		Name: dbMethod.Name,
	}
}
