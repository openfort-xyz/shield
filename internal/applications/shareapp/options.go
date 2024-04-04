package shareapp

type options struct {
	encryptionPart *string
}

type Option func(*options)

func WithEncryptionPart(encryptionPart string) Option {
	return func(o *options) {
		o.encryptionPart = &encryptionPart
	}
}
