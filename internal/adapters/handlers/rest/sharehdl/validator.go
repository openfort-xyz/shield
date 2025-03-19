package sharehdl

import "go.openfort.xyz/shield/internal/adapters/handlers/rest/api"

type validator struct {
}

func newValidator() *validator {
	return &validator{}
}

func (v *validator) validateShare(share *Share) *api.Error {
	if share.Secret == "" {
		return api.ErrBadRequestWithMessage("secret is required")
	}

	switch share.Entropy {
	case "", EntropyNone:
		if share.Salt != "" || share.Iterations != 0 || share.Length != 0 || share.Digest != "" || share.EncryptionPart != "" || share.EncryptionSession != "" {
			return api.ErrBadRequestWithMessage("if entropy is not set, encryption parameters should not be set")
		}

		share.Entropy = EntropyNone
	case EntropyUser:
		if share.Salt == "" {
			return api.ErrBadRequestWithMessage("salt is required when entropy is user")
		}
		if share.Iterations == 0 {
			return api.ErrBadRequestWithMessage("iterations is required when entropy is user")
		}
		if share.Length == 0 {
			return api.ErrBadRequestWithMessage("length is required when entropy is user")
		}
		if share.Digest == "" {
			return api.ErrBadRequestWithMessage("digest is required when entropy is user")
		}
	case EntropyProject:
		if share.Salt != "" || share.Iterations != 0 || share.Length != 0 || share.Digest != "" {
			return api.ErrBadRequestWithMessage("if user entropy is not set, encryption parameters should not be set")
		}

		if share.EncryptionPart == "" && share.EncryptionSession == "" {
			return api.ErrBadRequestWithMessage("encryption_part or encryption_session is required when entropy is project")
		}
	default:
		return api.ErrBadRequestWithMessage("invalid entropy")
	}

	return nil
}
