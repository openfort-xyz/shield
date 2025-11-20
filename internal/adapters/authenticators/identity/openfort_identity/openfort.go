package ofidty

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"go.openfort.xyz/shield/pkg/contexter"

	domainErrors "go.openfort.xyz/shield/internal/core/domain/errors"
	"go.openfort.xyz/shield/internal/core/ports/factories"
	"go.openfort.xyz/shield/pkg/jwk"

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

	isJwt := jwk.IsJWT(token)

	if !isJwt {
		return o.accessToken(ctx, token)
	} else {
		return o.jwtToken(ctx, token)
	}
}

func (a *OpenfortIdentityFactory) GetCookieFieldName() string {
	return ""
}

func (o *OpenfortIdentityFactory) accessToken(ctx context.Context, token string) (string, error) {
	url := fmt.Sprintf("%s/iam/v2/auth/get-session", o.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
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

	var response SessionResponse
	if err := json.Unmarshal(rawResponse, &response); err != nil {
		return "", err
	}

	expiresAt, err := time.Parse(time.RFC3339, response.Session.ExpiresAt)
	if err != nil {
		return "", fmt.Errorf("failed to parse expires_at: %w", err)
	}

	if time.Now().After(expiresAt) {
		return "", domainErrors.ErrSessionExpired
	}

	return response.User.Id, nil
}

func (o *OpenfortIdentityFactory) jwtToken(_ context.Context, token string) (string, error) {
	jwksUrls := []string{fmt.Sprintf("%s/iam/v1/%s/jwks.json", o.baseURL, o.publishableKey)}

	return jwk.Validate(token, jwksUrls)
}

func (o *OpenfortIdentityFactory) thirdParty(ctx context.Context, token, authenticationProvider, tokenType string) (string, error) {
	url := fmt.Sprintf("%s/iam/v1/oauth/third_party", o.baseURL)

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
	req.Header.Set("X-Request-ID", contexter.GetRequestID(ctx))
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

	return response.ID, nil
}

type authenticateOauthRequest struct {
	Provider  string `json:"provider"`
	Token     string `json:"token"`
	TokenType string `json:"tokenType"`
}

type authenticateOauthResponse struct {
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

type AuthSession struct {
	ExpiresAt string `json:"expiresAt"`
	Token     string `json:"token"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
	IpAddress string `json:"ipAddress"`
	UserAgent string `json:"userAgent"`
	UserId    string `json:"userId"`
	Id        string `json:"id"`
}

type AuthUser struct {
	Name                string `json:"name"`
	Email               string `json:"email"`
	EmailVerified       bool   `json:"emailVerified"`
	Image               string `json:"image"`
	CreatedAt           string `json:"createdAt"`
	UpdatedAt           string `json:"updatedAt"`
	IsAnonymous         bool   `json:"isAnonymous"`
	PhoneNumber         string `json:"phoneNumber"`
	PhoneNumberVerified bool   `json:"phoneNumberVerified"`
	Id                  string `json:"id"`
}

type SessionResponse struct {
	Session AuthSession
	User    AuthUser
}
