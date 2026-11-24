package sharehdl

import "github.com/openfort-xyz/shield/internal/adapters/handlers/rest/api"

type validator struct {
}

func newValidator() *validator {
	return &validator{}
}

func shareHasUserEntropyParams(share *Share) bool {
	return share.Salt != "" || share.Iterations != 0 || share.Length != 0 || share.Digest != ""
}

func validateUserEntropyParams(salt string, iterations, length int, digest string) *api.Error {
	if salt == "" {
		return api.ErrBadRequestWithMessage("salt is required when entropy is user")
	}
	if iterations <= 0 {
		return api.ErrBadRequestWithMessage("iterations must be positive when entropy is user")
	}

	switch digest {
	case "argon2":
		if length != 32 {
			return api.ErrBadRequestWithMessage("argon2 key length must be 32 bytes")
		}
	case "sha256", "sha512":
		// The lowercase legacy SHA digests report their key length in bits.
		if length < 128 || length > 512 || length%8 != 0 {
			return api.ErrBadRequestWithMessage("SHA key length must be between 128 and 512 bits in multiples of 8")
		}
	case "SHA-256", "SHA-512":
		if length < 16 || length > 64 {
			return api.ErrBadRequestWithMessage("SHA key length must be between 16 and 64 bytes")
		}
	default:
		return api.ErrBadRequestWithMessage("unsupported digest for user entropy")
	}
	return nil
}

func (v *validator) validateShare(share *Share) *api.Error {
	if share.Secret == "" {
		return api.ErrBadRequestWithMessage("secret is required")
	}

	if !share.ShareStorageMethodID.IsValid() {
		return api.ErrBadRequestWithMessage("invalid storage method")
	}

	switch share.Entropy {
	case "", EntropyNone:
		return api.ErrBadRequestWithMessage("require share entropy to be set")
	case EntropyUser:
		return validateUserEntropyParams(share.Salt, share.Iterations, share.Length, share.Digest)
	case EntropyProject:
		if share.ShareStorageMethodID != StorageMethodShield {
			return api.ErrBadRequestWithMessage("storage_method must be Shield if entropy is project")
		}

		if shareHasUserEntropyParams(share) {
			return api.ErrBadRequestWithMessage("if user entropy is not set, encryption parameters should not be set")
		}

		if share.EncryptionPart == "" && share.EncryptionSession == "" {
			return api.ErrBadRequestWithMessage("encryption_part or encryption_session is required when entropy is project")
		}
	case EntropyPasskey:
		if share.ShareStorageMethodID != StorageMethodShield {
			return api.ErrBadRequestWithMessage("storage_method must be Shield if entropy is passkey")
		}

		if share.Reference == "" || share.Reference == "default" {
			return api.ErrBadRequestWithMessage("share needs a valid share reference if entropy is passkey")
		}

		if share.PasskeyReference == nil {
			return api.ErrBadRequestWithMessage("passkey_reference must be set if entropy is passkey")
		}

		if share.PasskeyReference.PasskeyID == nil {
			return api.ErrBadRequestWithMessage("passkey_reference must contain passkey_id if entropy is passkey")
		}

		// User entroy parameters should not be set for passkey entropy
		if shareHasUserEntropyParams(share) {
			return api.ErrBadRequestWithMessage("if user entropy is not set, encryption parameters should not be set")
		}

		// Encryption part/encryption session belong to project entropy use case
		// Here we're only storing an encrypted share and hinting how it's encrypted, like we're doing for user entropy
		// Except we rely on proper user authentication for correct passkey retrieval and decryption
		if share.EncryptionPart != "" || share.EncryptionSession != "" {
			return api.ErrBadRequestWithMessage("encryption parameters should not be set for passkey entropy")
		}
	default:
		return api.ErrBadRequestWithMessage("invalid entropy")
	}

	return nil
}
