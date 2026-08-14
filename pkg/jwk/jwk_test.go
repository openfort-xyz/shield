package jwk

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
)

func startJWKS(t *testing.T) (*httptest.Server, *ecdsa.PrivateKey) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}
	b64 := base64.RawURLEncoding.EncodeToString
	jwks := fmt.Sprintf(
		`{"keys":[{"kty":"EC","crv":"P-256","kid":"test-key-1","alg":"ES256","use":"sig","x":%q,"y":%q}]}`,
		b64(key.PublicKey.X.FillBytes(make([]byte, 32))),
		b64(key.PublicKey.Y.FillBytes(make([]byte, 32))),
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(jwks))
	}))
	t.Cleanup(server.Close)
	return server, key
}

func signES256(t *testing.T, key *ecdsa.PrivateKey, claims jwt.MapClaims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	token.Header["kid"] = "test-key-1"
	signed, err := token.SignedString(key)
	if err != nil {
		t.Fatalf("failed to sign: %v", err)
	}
	return signed
}

func TestValidate_AcceptsUnexpiredToken(t *testing.T) {
	server, key := startJWKS(t)
	token := signES256(t, key, jwt.MapClaims{
		"sub": "user-123",
		"exp": time.Now().Add(time.Hour).Unix(),
	})

	sub, err := Validate(token, []string{server.URL})
	if err != nil {
		t.Fatalf("valid token was rejected: %v", err)
	}
	if sub != "user-123" {
		t.Fatalf("got sub %q, want %q", sub, "user-123")
	}
}

func TestValidate_RejectsExpiredToken(t *testing.T) {
	server, key := startJWKS(t)
	token := signES256(t, key, jwt.MapClaims{
		"sub": "user-123",
		"exp": time.Now().Add(-time.Hour).Unix(),
	})

	if _, err := Validate(token, []string{server.URL}); err == nil {
		t.Fatal("expected error for expired token")
	}
}

func TestValidate_RejectsMissingExp(t *testing.T) {
	server, key := startJWKS(t)
	token := signES256(t, key, jwt.MapClaims{
		"sub": "user-123",
	})

	if _, err := Validate(token, []string{server.URL}); err == nil {
		t.Fatal("expected error for token with no exp claim")
	}
}
