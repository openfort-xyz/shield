package authentication

type Authentication struct {
	UserID         string
	ProjectID      string
	ExternalUserID string
	// AltExternalUserID is Identity.AltExternalUserID.
	AltExternalUserID string
}

// Identity is a user as the auth provider names it.
type Identity struct {
	ExternalUserID string
	// AltExternalUserID is a second ID the provider issues for the same user,
	// empty when it has none. Shares are stored under ExternalUserID only.
	AltExternalUserID string
}
