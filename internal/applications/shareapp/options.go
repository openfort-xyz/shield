package shareapp

type options struct {
	encryptionPart    *string
	encryptionSession *string
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
