package di

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var octSecret = []byte("0123456789abcdef0123456789abcdef")

// jwksServer publishes an oct key next to an RSA and an EC key, at the root and
// at the Openfort per-project path, and returns the asymmetric signing keys.
func jwksServer(t *testing.T) (*httptest.Server, *rsa.PrivateKey, *ecdsa.PrivateKey) {
	t.Helper()
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate rsa key: %v", err)
	}
	ecKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate ec key: %v", err)
	}
	b64 := base64.RawURLEncoding.EncodeToString
	jwks := fmt.Sprintf(`{"keys":[`+
		`{"kty":"oct","kid":"oct-1","k":%q},`+
		`{"kty":"RSA","kid":"rsa-1","alg":"RS256","use":"sig","n":%q,"e":%q},`+
		`{"kty":"EC","kid":"ec-1","alg":"ES256","use":"sig","crv":"P-256","x":%q,"y":%q}]}`,
		b64(octSecret), b64(rsaKey.N.Bytes()), b64(big.NewInt(int64(rsaKey.E)).Bytes()),
		b64(ecKey.X.FillBytes(make([]byte, 32))), b64(ecKey.Y.FillBytes(make([]byte, 32))))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(jwks))
	}))
	t.Cleanup(srv.Close)
	return srv, rsaKey, ecKey
}

func signJWT(t *testing.T, method jwt.SigningMethod, kid string, key any, sub string) string {
	t.Helper()
	token := jwt.NewWithClaims(method, jwt.MapClaims{"sub": sub, "exp": time.Now().Add(time.Hour).Unix()})
	token.Header["kid"] = kid
	signed, err := token.SignedString(key)
	if err != nil {
		t.Fatalf("sign %s token: %v", method.Alg(), err)
	}
	return signed
}

func TestCustomJWKProvider_RejectsHMACToken_HTTP(t *testing.T) {
	srv, rsaKey, _ := jwksServer(t)
	u, _ := url.Parse(srv.URL)
	t.Setenv("JWK_ALLOWED_HOSTS", u.Host)
	c, apiKey, _ := newProject(t)
	expect(t, "add custom jwk provider", c.do(http.MethodPost, "/project/providers", apiKey,
		map[string]any{"providers": map[string]any{"custom": map[string]any{"jwk": srv.URL + "/jwks.json"}}}),
		http.StatusOK, "")
	getShares := func(token string) reply {
		return c.do(http.MethodGet, "/shares", "", nil,
			"X-API-Key", apiKey, "X-Auth-Provider", "custom", "Authorization", "Bearer "+token)
	}

	expect(t, "HS256 token signed with the published oct key",
		getShares(signJWT(t, jwt.SigningMethodHS256, "oct-1", octSecret, "user-"+uuid.NewString())),
		http.StatusUnauthorized, "A_INVALID")
	expect(t, "RS256 token from the provider",
		getShares(signJWT(t, jwt.SigningMethodRS256, "rsa-1", rsaKey, "user-"+uuid.NewString())),
		http.StatusNotFound, "SH_NOT_FOUND")
}

func TestOpenfortJWT_RejectsHMACToken_HTTP(t *testing.T) {
	srv, _, ecKey := jwksServer(t)
	t.Setenv("OPENFORT_BASE_URL", srv.URL)
	c, apiKey, _ := newProject(t)
	expect(t, "add openfort provider", c.do(http.MethodPost, "/project/providers", apiKey,
		map[string]any{"providers": map[string]any{"openfort": map[string]any{"publishable_key": "pk_test_" + uuid.NewString()}}}),
		http.StatusOK, "")
	getShares := func(token string) reply {
		return c.do(http.MethodGet, "/shares", "", nil,
			"X-API-Key", apiKey, "X-Auth-Provider", "openfort", "Authorization", "Bearer "+token)
	}

	expect(t, "HS256 token signed with the published oct key",
		getShares(signJWT(t, jwt.SigningMethodHS256, "oct-1", octSecret, "user-"+uuid.NewString())),
		http.StatusUnauthorized, "A_INVALID")
	expect(t, "ES256 token from Openfort",
		getShares(signJWT(t, jwt.SigningMethodES256, "ec-1", ecKey, "user-"+uuid.NewString())),
		http.StatusNotFound, "SH_NOT_FOUND")
}
