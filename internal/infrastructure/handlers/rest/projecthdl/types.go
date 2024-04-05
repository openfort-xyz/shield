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

type ProvidersRequest struct {
	Openfort *OpenfortProvider `json:"openfort,omitempty"`
	Custom   *CustomProvider   `json:"custom,omitempty"`
}

type OpenfortProvider struct {
	ProviderID     string `json:"provider_id,omitempty"`
	PublishableKey string `json:"publishable_key,omitempty"`
}

type CustomProvider struct {
	ProviderID string `json:"provider_id,omitempty"`
	JWK        string `json:"jwk,omitempty"`
}

type AddProvidersResponse struct {
	Providers []*ProviderResponse `json:"providers"`
}

type ProviderResponse struct {
	ProviderID string `json:"provider_id"`
	Type       string `json:"type"`
}

type GetProvidersResponse struct {
	Providers []*ProviderResponse `json:"providers"`
}

type GetProviderResponse struct {
	ProviderID     string `json:"provider_id"`
	Type           string `json:"type"`
	PublishableKey string `json:"publishable_key,omitempty"`
	JWK            string `json:"jwk,omitempty"`
}

type UpdateProviderRequest struct {
	PublishableKey string `json:"publishable_key,omitempty"`
	JWK            string `json:"jwk,omitempty"`
}

type AddAllowedOriginRequest struct {
	Origin string `json:"origin"`
}

type GetAllowedOriginsResponse struct {
	Origins []string `json:"origins"`
}

type EncryptBodyRequest struct {
	EncryptionPart string `json:"encryption_part"`
}

type RegisterEncryptionKeyResponse struct {
	EncryptionPart string `json:"encryption_part"`
}
