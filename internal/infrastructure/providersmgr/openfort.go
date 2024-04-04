package providersmgr

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"go.openfort.xyz/shield/internal/core/domain/provider"
	"go.openfort.xyz/shield/internal/core/ports/providers"
	"go.openfort.xyz/shield/pkg/logger"
)

type openfort struct {
	publishableKey string
	baseURL        string
	providerID     string
	logger         *slog.Logger
}

var _ providers.IdentityProvider = (*openfort)(nil)

func newOpenfortProvider(config *Config, providerConfig *provider.OpenfortConfig) providers.IdentityProvider {
	return &openfort{
		publishableKey: providerConfig.PublishableKey,
		providerID:     providerConfig.ProviderID,
		baseURL:        config.OpenfortBaseURL,
		logger:         logger.New("openfort_provider"),
	}
}

func (o *openfort) GetProviderID() string {
	return o.providerID
}

func (o *openfort) Identify(ctx context.Context, token string, opts ...providers.CustomOption) (string, error) {
	o.logger.InfoContext(ctx, "identifying user")

	userID, err := validateJWKs(token, fmt.Sprintf("%s/iam/v1/%s/jwks.json", o.baseURL, o.publishableKey))
	if err != nil {
		if !errors.Is(err, ErrInvalidToken) {
			o.logger.ErrorContext(ctx, "failed to validate jwks", logger.Error(err))
			return "", err
		}

		return o.identifyOAuth(ctx, token, opts...)
	}

	return userID, nil
}

func (o *openfort) identifyOAuth(ctx context.Context, token string, opts ...providers.CustomOption) (string, error) {
	var opt providers.CustomOptions
	for _, o := range opts {
		o(&opt)
	}

	if opt.OpenfortProvider == nil {
		return "", ErrMissingOpenfortProvider
	}

	if opt.OpenfortTokenType == nil {
		return "", ErrMissingOpenfortTokenType
	}

	url := fmt.Sprintf("%s/iam/v1/oauth/authenticate", o.baseURL)

	reqBody := authenticateOauthRequest{
		Provider:  *opt.OpenfortProvider,
		Token:     token,
		TokenType: *opt.OpenfortTokenType,
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
		return "", ErrUnexpectedStatusCode
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
