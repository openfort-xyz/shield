package projecthdl

type CreateProjectRequest struct {
	Name                  string `json:"name"`
	GenerateEncryptionKey bool   `json:"generate_encryption_key,omitempty"`
}

type CreateProjectResponse struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	APIKey         string `json:"api_key"`
	APISecret      string `json:"api_secret"`
	EncryptionPart string `json:"encryption_part,omitempty"`
}

type GetProjectResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type AddProvidersRequest struct {
	Providers ProvidersRequest `json:"providers"`
}

type AddProviderV2Request struct {
	Provider CustomProvider `json:"provider"`
}

type ProvidersRequest struct {
	Openfort *OpenfortProvider `json:"openfort,omitempty"`
	Custom   *CustomProvider   `json:"custom,omitempty"`
}

type OpenfortProvider struct {
	ProviderID     string `json:"provider_id,omitempty"`
	PublishableKey string `json:"publishable_key,omitempty"`
}

type CustomProvider struct {
	ProviderID      string  `json:"provider_id,omitempty"`
	JWK             string  `json:"jwk,omitempty"`
	PEM             string  `json:"pem,omitempty"`
	CookieFieldName *string `json:"cookie_field_name,omitempty"`
	KeyType         KeyType `json:"key_type,omitempty"`
}

type KeyType string

const (
	KeyTypeRSA     KeyType = "rsa"
	KeyTypeECDSA   KeyType = "ecdsa"
	KeyTypeEd25519 KeyType = "ed25519"
)

type AddProvidersResponse struct {
	Providers []*ProviderResponse `json:"providers"`
}

type AddProviderV2Response struct {
	ProviderID string `json:"provider_id"`
}

type ProviderResponse struct {
	ProviderID string `json:"provider_id"`
	Type       string `json:"type"`
}

type GetProvidersResponse struct {
	Providers []*ProviderResponse `json:"providers"`
}

type GetProviderResponse struct {
	ProviderID      string  `json:"provider_id"`
	Type            string  `json:"type"`
	PublishableKey  string  `json:"publishable_key,omitempty"`
	JWK             string  `json:"jwk,omitempty"`
	PEM             string  `json:"pem,omitempty"`
	CookieFieldName *string `json:"cookie_field_name,omitempty"`
	KeyType         KeyType `json:"key_type,omitempty"`
}

type UpdateProviderRequest struct {
	PublishableKey  string  `json:"publishable_key,omitempty"`
	JWK             string  `json:"jwk,omitempty"`
	PEM             string  `json:"pem,omitempty"`
	CookieFieldName *string `json:"cookie_field_name,omitempty"`
	KeyType         KeyType `json:"key_type,omitempty"`
}

type EncryptBodyRequest struct {
	EncryptionPart string `json:"encryption_part"`
}

type RegisterEncryptionKeyResponse struct {
	EncryptionPart string `json:"encryption_part"`
}

type RegisterEncryptionSessionRequest struct {
	EncryptionPart string `json:"encryption_part"`
}

type RegisterEncryptionSessionResponse struct {
	SessionID string `json:"session_id"`
}
