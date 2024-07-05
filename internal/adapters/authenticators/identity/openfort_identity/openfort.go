package openfort_identity

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	domainErrors "go.openfort.xyz/shield/internal/core/domain/errors"
	"go.openfort.xyz/shield/internal/core/ports/factories"
	"go.openfort.xyz/shield/pkg/jwk"
	"io"
	"log/slog"
	"net/http"
	"time"

	"go.openfort.xyz/shield/internal/core/domain/provider"
	"go.openfort.xyz/shield/pkg/logger"
)

type OpenfortIdentityFactory struct {
	publishableKey string
	baseURL        string
	providerID     string
	logger         *slog.Logger

	authenticationProvider *string
	tokenType              *string
}

var _ factories.Identity = (*OpenfortIdentityFactory)(nil)

func NewOpenfortIdentityFactory(config *Config, providerConfig *provider.OpenfortConfig, authenticationProvider, tokenType *string) factories.Identity {
	return &OpenfortIdentityFactory{
		publishableKey:         providerConfig.PublishableKey,
		providerID:             providerConfig.ProviderID,
		baseURL:                config.OpenfortBaseURL,
		logger:                 logger.New("openfort_provider"),
		authenticationProvider: authenticationProvider,
		tokenType:              tokenType,
	}
}

func (o *OpenfortIdentityFactory) GetProviderID() string {
	return o.providerID
}

func (o *OpenfortIdentityFactory) Identify(ctx context.Context, token string) (string, error) {
	o.logger.InfoContext(ctx, "identifying user")

	if o.authenticationProvider != nil && o.tokenType != nil {
		return o.thirdParty(ctx, token, *o.authenticationProvider, *o.tokenType)
	}

	return o.accessToken(ctx, token)
}

func (o *OpenfortIdentityFactory) accessToken(ctx context.Context, token string) (string, error) {
	return jwk.Validate(token, fmt.Sprintf("%s/iam/v1/%s/jwks.json", o.baseURL, o.publishableKey)) // TODO parse error
}

func (o *OpenfortIdentityFactory) thirdParty(ctx context.Context, token, authenticationProvider, tokenType string) (string, error) {
	url := fmt.Sprintf("%s/iam/v1/oauth/authenticate", o.baseURL)

	reqBody := authenticateOauthRequest{
		Provider:  authenticationProvider,
		Token:     token,
		TokenType: tokenType,
	}

	rawReqBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(rawReqBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", o.publishableKey))
	client := http.Client{Timeout: time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		o.logger.ErrorContext(ctx, "unexpected status code", slog.Int("status_code", resp.StatusCode))
		return "", domainErrors.ErrUnexpectedStatusCode
	}

	rawResponse, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var response authenticateOauthResponse
	if err := json.Unmarshal(rawResponse, &response); err != nil {
		return "", err
	}

	return response.Player.ID, nil
}

type authenticateOauthRequest struct {
	Provider  string `json:"provider"`
	Token     string `json:"token"`
	TokenType string `json:"tokenType"`
}

type authenticateOauthResponse struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refreshToken"`
	Player       player `json:"player"`
}

type player struct {
	ID             string          `json:"id"`
	Object         string          `json:"object"`
	CreatedAt      int64           `json:"created_at"`
	LinkedAccounts []linkedAccount `json:"linked_accounts"`
}

type linkedAccount struct {
	Provider       string `json:"provider"`
	Email          string `json:"email,omitempty"`
	ExternalUserID string `json:"external_user_id,omitempty"`
	Disabled       bool   `json:"disabled"`
	UpdatedAt      int64  `json:"updated_at,omitempty"`
	Address        string `json:"address,omitempty"`
	Metadata       string `json:"metadata,omitempty"`
}
