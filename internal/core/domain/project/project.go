package project

type Project struct {
	ID             string
	Name           string
	APIKey         string
	APISecret      string
	EncryptionPart string
	Enable2FA      bool
	RateLimit      int64
}

type ProjectWithRateLimit struct {
	ID             string
	Name           string
	APIKey         string
	APISecret      string
	EncryptionPart string
	Enable2FA      bool
	RateLimit      int64
}

type RateLimit struct {
	ProjectID         string
	RequestsPerMinute int64
}
