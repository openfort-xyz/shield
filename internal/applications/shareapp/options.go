package shareapp

type options struct {
	encryptionPart    *string
	encryptionSession *string
	requireOTPCheck   bool
}

// partSource names where the caller's half of the encryption key came from,
// so a log line can attribute a key mismatch to the right input.
func (o options) partSource() string {
	switch {
	case o.encryptionPart != nil && *o.encryptionPart != "":
		return "part"
	case o.encryptionSession != nil && *o.encryptionSession != "":
		return "session"
	default:
		return ""
	}
}

type Option func(*options)

func WithEncryptionPart(encryptionPart string) Option {
	return func(o *options) {
		o.encryptionPart = &encryptionPart
	}
}

func WithEncryptionSession(encryptionSession string) Option {
	return func(o *options) {
		o.encryptionSession = &encryptionSession
	}
}
