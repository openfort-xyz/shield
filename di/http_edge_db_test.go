package di

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"net/http"
	"strings"
	"testing"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// userProject is a registered project with a custom PEM provider whose users
// authenticate with tokens signed by signer.
type userProject struct {
	httpClient
	apiKey     string
	part       string
	providerID string
	signer     *rsa.PrivateKey
}

func newUserProject(t *testing.T) userProject {
	t.Helper()
	c, apiKey, part := newProject(t)
	signer, pubPEM := newSigner(t)

	res := c.do(http.MethodPost, "/project/providers", apiKey,
		map[string]any{"providers": map[string]any{"custom": map[string]any{"pem": pubPEM, "key_type": "rsa"}}})
	expect(t, "add custom provider", res, http.StatusOK, "")
	providers, _ := res.Body["providers"].([]any)
	if len(providers) != 1 {
		t.Fatalf("add custom provider: %v, want one provider", res.Body)
	}
	providerID, _ := providers[0].(map[string]any)["provider_id"].(string)
	return userProject{httpClient: c, apiKey: apiKey, part: part, providerID: providerID, signer: signer}
}

func newSigner(t *testing.T) (*rsa.PrivateKey, string) {
	t.Helper()
	signer, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	pubDER, err := x509.MarshalPKIXPublicKey(&signer.PublicKey)
	if err != nil {
		t.Fatalf("marshal public key: %v", err)
	}
	return signer, string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER}))
}

func (p userProject) token(t *testing.T, externalUserID string) string {
	t.Helper()
	signed, err := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"sub": externalUserID,
		"exp": time.Now().Add(time.Hour).Unix(),
	}).SignedString(p.signer)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return signed
}

// asUser calls a user-authenticated route with a bearer token for externalUserID.
func (p userProject) asUser(t *testing.T, method, path, externalUserID string, body any, headers ...string) reply {
	t.Helper()
	headers = append([]string{"X-API-Key", p.apiKey, "X-Auth-Provider", "custom",
		"Authorization", "Bearer " + p.token(t, externalUserID)}, headers...)
	return p.do(method, path, "", body, headers...)
}

func encryptionType(t *testing.T, step string, got reply, key string) (string, string) {
	t.Helper()
	expect(t, step, got, http.StatusOK, "")
	types, _ := got.Body["encryption_types"].(map[string]any)
	entry, _ := types[key].(map[string]any)
	status, _ := entry["status"].(string)
	entropy, _ := entry["encryption_type"].(string)
	return status, entropy
}

func TestUpdateProvider_CookieFieldOnly_HTTP(t *testing.T) {
	p := newUserProject(t)
	path := "/project/providers/" + p.providerID

	expect(t, "update cookie field only", p.do(http.MethodPut, path, p.apiKey, map[string]any{"cookie_field_name": "session"}),
		http.StatusOK, "")

	reloaded := p.do(http.MethodGet, path, p.apiKey, nil)
	expect(t, "reload provider", reloaded, http.StatusOK, "")
	if reloaded.str("cookie_field_name") != "session" {
		t.Fatalf("cookie_field_name after update = %q, want %q", reloaded.str("cookie_field_name"), "session")
	}
	if reloaded.str("pem") == "" || reloaded.str("key_type") != "rsa" {
		t.Fatalf("provider after cookie-only update = %v, want the PEM and key type kept", reloaded.Body)
	}

	user := "user-" + uuid.NewString()
	viaCookie := p.do(http.MethodGet, "/shares", "", nil,
		"X-API-Key", p.apiKey, "X-Auth-Provider", "custom", "Cookie", "session="+p.token(t, user))
	expect(t, "authenticate with the new cookie", viaCookie, http.StatusNotFound, "SH_NOT_FOUND")
}

func TestProvider_CookieFieldNameLength_HTTP(t *testing.T) {
	const maxLength = 64
	p := newUserProject(t)
	path := "/project/providers/" + p.providerID

	expect(t, "update with a cookie field name over the limit",
		p.do(http.MethodPut, path, p.apiKey, map[string]any{"cookie_field_name": strings.Repeat("c", maxLength+1)}),
		http.StatusBadRequest, "PV_CFG_INVALID")
	atLimit := strings.Repeat("é", maxLength)
	expect(t, "update with a cookie field name at the limit",
		p.do(http.MethodPut, path, p.apiKey, map[string]any{"cookie_field_name": atLimit}), http.StatusOK, "")
	if got := p.do(http.MethodGet, path, p.apiKey, nil).str("cookie_field_name"); got != atLimit {
		t.Fatalf("cookie_field_name = %q, want %q", got, atLimit)
	}

	c, apiKey, _ := newProject(t)
	_, pubPEM := newSigner(t)
	expect(t, "add with a cookie field name over the limit", c.do(http.MethodPost, "/project/providers", apiKey,
		map[string]any{"providers": map[string]any{"custom": map[string]any{
			"pem": pubPEM, "key_type": "rsa", "cookie_field_name": strings.Repeat("c", maxLength+1),
		}}}), http.StatusBadRequest, "PV_CFG_INVALID")
	got := c.do(http.MethodGet, "/project/providers", apiKey, nil)
	if providers, _ := got.Body["providers"].([]any); got.Status != http.StatusOK || len(providers) != 0 {
		t.Fatalf("providers after a rejected add = %d %v, want none", got.Status, got.Body)
	}
}

func TestBodyLimit_AuthenticatedRequest_HTTP(t *testing.T) {
	c, apiKey, _ := newProject(t)
	overLimit := strings.Repeat("a", 256<<10)

	for _, path := range []string{"/shares/encryption/user/bulk", "/project/rotate-encryption-key/prepare"} {
		expect(t, "POST "+path+" over the body limit", c.do(http.MethodPost, path, apiKey, overLimit),
			http.StatusRequestEntityTooLarge, "BODY_TOO_LARGE")
	}
}

func TestEncryptionPart_StoredPartCoordinate_HTTP(t *testing.T) {
	const storedPartCoordinate = 1
	p := newUserProject(t)
	user := "user-" + uuid.NewString()

	expect(t, "register project-entropy share", p.asUser(t, http.MethodPost, "/shares", user, map[string]any{
		"secret": "project-secret", "entropy": "project", "encryption_part": p.part,
	}), http.StatusCreated, "")

	raw, err := base64.StdEncoding.DecodeString(p.part)
	if err != nil {
		t.Fatalf("decode encryption part: %v", err)
	}
	raw[0] = storedPartCoordinate
	expect(t, "read with a part on the stored part's coordinate",
		p.asUser(t, http.MethodGet, "/shares", user, nil, "X-Encryption-Part", base64.StdEncoding.EncodeToString(raw)),
		http.StatusBadRequest, "EC_INVALID")

	got := p.asUser(t, http.MethodGet, "/shares", user, nil, "X-Encryption-Part", p.part)
	expect(t, "read with the real part", got, http.StatusOK, "")
	if got.str("secret") != "project-secret" {
		t.Fatalf("secret = %q, want project-secret", got.str("secret"))
	}
}

func TestShareEncryptionRoutes_HTTP(t *testing.T) {
	p := newUserProject(t)
	user := "user-" + uuid.NewString()
	reference := "ref-" + uuid.NewString()

	expect(t, "register default share", p.asUser(t, http.MethodPost, "/shares", user, map[string]any{
		"secret": "default-secret", "entropy": "user",
		"salt": "default-salt", "iterations": 1000, "length": 32, "digest": "SHA-256",
	}), http.StatusCreated, "")
	expect(t, "register referenced share", p.asUser(t, http.MethodPost, "/shares", user, map[string]any{
		"secret": "user-secret", "entropy": "user", "reference": reference,
		"salt": "salt", "iterations": 1000, "length": 32, "digest": "SHA-256",
	}), http.StatusCreated, "")

	enc := p.asUser(t, http.MethodGet, "/shares/encryption", user, nil)
	expect(t, "GET /shares/encryption as the user", enc, http.StatusOK, "")
	if enc.str("entropy") != "user" || enc.str("salt") != "default-salt" || enc.str("digest") != "SHA-256" {
		t.Fatalf("GET /shares/encryption body = %v, want the default share's encryption parameters", enc.Body)
	}

	byRef := p.asUser(t, http.MethodGet, "/shares/"+reference, user, nil)
	expect(t, "GET /shares/{reference}", byRef, http.StatusOK, "")
	if byRef.str("secret") != "user-secret" || byRef.str("reference") != reference {
		t.Fatalf("GET /shares/{reference} body = %v, want the registered share", byRef.Body)
	}

	encWithSecret := p.do(http.MethodGet, "/shares/encryption", p.apiKey, nil)
	expect(t, "GET /shares/encryption with the API secret only", encWithSecret, http.StatusUnauthorized, "A_MISSING")
	if encWithSecret.str("message") != "Missing auth provider" {
		t.Fatalf("GET /shares/encryption with the API secret only: %v, want Missing auth provider", encWithSecret.Body)
	}

	status, entropy := encryptionType(t, "reference bulk with the API secret",
		p.do(http.MethodPost, "/shares/encryption/reference/bulk", p.apiKey, map[string]any{"references": []string{reference}}), reference)
	if status != "found" || entropy != "user" {
		t.Fatalf("reference bulk: status %q entropy %q, want found user", status, entropy)
	}
	status, entropy = encryptionType(t, "user bulk with the API secret",
		p.do(http.MethodPost, "/shares/encryption/user/bulk", p.apiKey, map[string]any{"user_ids": []string{user}}), user)
	if status != "found" || entropy != "user" {
		t.Fatalf("user bulk: status %q entropy %q, want found user", status, entropy)
	}

	bulkAsUser := p.asUser(t, http.MethodPost, "/shares/encryption/reference/bulk", user, map[string]any{"references": []string{reference}})
	expect(t, "reference bulk with a user token only", bulkAsUser, http.StatusUnauthorized, "A_MISSING")
	if bulkAsUser.str("message") != "Missing API secret" {
		t.Fatalf("reference bulk with a user token only: %v, want Missing API secret", bulkAsUser.Body)
	}
}

func TestEncryptionPart_RejectedOnReadsWith2FA_HTTP(t *testing.T) {
	p := newUserProject(t)
	user := "user-" + uuid.NewString()
	readPaths := []string{"/shares", "/keychain", "/shares/default"}

	expect(t, "register project-entropy share", p.asUser(t, http.MethodPost, "/shares", user, map[string]any{
		"secret": "project-secret", "entropy": "project", "encryption_part": p.part,
	}), http.StatusCreated, "")
	for _, path := range readPaths {
		expect(t, "GET "+path+" with the raw part before 2FA",
			p.asUser(t, http.MethodGet, path, user, nil, "X-Encryption-Part", p.part), http.StatusOK, "")
	}

	expect(t, "enable 2FA", p.do(http.MethodPost, "/project/enable-2fa", p.apiKey, nil), http.StatusOK, "")

	for _, path := range readPaths {
		expect(t, "GET "+path+" with the raw part on a 2FA project",
			p.asUser(t, http.MethodGet, path, user, nil, "X-Encryption-Part", p.part),
			http.StatusPreconditionRequired, "OTP_MISSING")
	}
	expect(t, "session without an OTP code", p.do(http.MethodPost, "/project/encryption-session", p.apiKey,
		map[string]any{"encryption_part": p.part, "user_id": user}), http.StatusPreconditionRequired, "OTP_MISSING")
}
