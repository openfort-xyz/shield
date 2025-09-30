package project

type Project struct {
	ID             string
	Name           string
	APIKey         string
	APISecret      string
	EncryptionPart string
	Enable2FA      bool
	SMSRateLimit   int64
	EmailRateLimit int64
}

type WithRateLimit struct {
	ID             string
	Name           string
	APIKey         string
	APISecret      string
	EncryptionPart string
	Enable2FA      bool
	SMSRateLimit   int64
	EmailRateLimit int64
}

type RateLimit struct {
	ProjectID            string
	SMSRequestsPerHour   int64
	EmailRequestsPerHour int64
}
