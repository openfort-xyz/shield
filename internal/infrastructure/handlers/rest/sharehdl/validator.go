package sharehdl

import "go.openfort.xyz/shield/internal/infrastructure/handlers/rest/api"

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
	case EntropyNone:
	case "":
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
		if share.EncryptionPart == "" {
			return api.ErrBadRequestWithMessage("encryption_part is required when entropy is project")
		}
	default:
		return api.ErrBadRequestWithMessage("invalid entropy")
	}

	return nil
}
