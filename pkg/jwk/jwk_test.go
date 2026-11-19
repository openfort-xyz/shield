package jwk

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
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

func serveJWKS(t *testing.T, jwks string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(jwks))
	}))
	t.Cleanup(server.Close)
	return server
}

func TestValidate_AcceptsRS256Token(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}
	b64 := base64.RawURLEncoding.EncodeToString
	server := serveJWKS(t, fmt.Sprintf(`{"keys":[{"kty":"RSA","kid":"rsa-1","alg":"RS256","use":"sig","n":%q,"e":%q}]}`,
		b64(key.N.Bytes()), b64(big.NewInt(int64(key.E)).Bytes())))
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{"sub": "user-123", "exp": time.Now().Add(time.Hour).Unix()})
	token.Header["kid"] = "rsa-1"
	signed, err := token.SignedString(key)
	if err != nil {
		t.Fatalf("failed to sign: %v", err)
	}

	sub, err := Validate(signed, []string{server.URL})
	if err != nil || sub != "user-123" {
		t.Fatalf("Validate(RS256) = %q, %v; want user-123", sub, err)
	}
}

// A JWK set may publish a symmetric (oct) key, which is a shared secret rather
// than a public key; a token signed with it must not authenticate anyone.
func TestValidate_RejectsHMACTokenSignedWithOctKey(t *testing.T) {
	secret := []byte("0123456789abcdef0123456789abcdef")
	server := serveJWKS(t, fmt.Sprintf(`{"keys":[{"kty":"oct","kid":"oct-1","k":%q}]}`,
		base64.RawURLEncoding.EncodeToString(secret)))
	u, _ := url.Parse(server.URL)
	t.Setenv(AllowedHostsEnv, u.Host)

	for _, method := range []*jwt.SigningMethodHMAC{jwt.SigningMethodHS256, jwt.SigningMethodHS384, jwt.SigningMethodHS512} {
		// Without a kid, keyfunc offers every key in the set, the oct one included.
		for _, kid := range []string{"oct-1", ""} {
			token := jwt.NewWithClaims(method, jwt.MapClaims{"sub": "user", "exp": time.Now().Add(time.Hour).Unix()})
			if kid != "" {
				token.Header["kid"] = kid
			}
			signed, err := token.SignedString(secret)
			if err != nil {
				t.Fatalf("failed to sign: %v", err)
			}

			if sub, err := Validate(signed, []string{server.URL}); err == nil {
				t.Errorf("Validate accepted a %s token (kid %q) for %q", method.Alg(), kid, sub)
			}
			if sub, err := ValidateTenant(signed, server.URL+"/jwks.json"); err == nil {
				t.Errorf("ValidateTenant accepted a %s token (kid %q) for %q", method.Alg(), kid, sub)
			}
		}
	}

	unsigned := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{"sub": "user", "exp": time.Now().Add(time.Hour).Unix()})
	signed, err := unsigned.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("failed to sign: %v", err)
	}
	if sub, err := Validate(signed, []string{server.URL}); err == nil {
		t.Errorf("Validate accepted an unsigned token for %q", sub)
	}
}

func TestValidate_AcceptsEveryAsymmetricMethod(t *testing.T) {
	b64 := base64.RawURLEncoding.EncodeToString
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate RSA key: %v", err)
	}
	rsaJWK := fmt.Sprintf(`{"kty":"RSA","kid":"k","n":%q,"e":%q}`, b64(rsaKey.N.Bytes()), b64(big.NewInt(int64(rsaKey.E)).Bytes()))
	ecJWK := func(curve elliptic.Curve, name string) (*ecdsa.PrivateKey, string) {
		key, err := ecdsa.GenerateKey(curve, rand.Reader)
		if err != nil {
			t.Fatalf("generate %s key: %v", name, err)
		}
		size := (curve.Params().BitSize + 7) / 8
		return key, fmt.Sprintf(`{"kty":"EC","kid":"k","crv":%q,"x":%q,"y":%q}`, name,
			b64(key.X.FillBytes(make([]byte, size))), b64(key.Y.FillBytes(make([]byte, size))))
	}
	p256, p256JWK := ecJWK(elliptic.P256(), "P-256")
	p384, p384JWK := ecJWK(elliptic.P384(), "P-384")
	p521, p521JWK := ecJWK(elliptic.P521(), "P-521")
	edPub, edKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate Ed25519 key: %v", err)
	}
	edJWK := fmt.Sprintf(`{"kty":"OKP","kid":"k","crv":"Ed25519","x":%q}`, b64(edPub))

	cases := []struct {
		method jwt.SigningMethod
		key    crypto.Signer
		jwk    string
	}{
		{jwt.SigningMethodRS256, rsaKey, rsaJWK}, {jwt.SigningMethodRS384, rsaKey, rsaJWK}, {jwt.SigningMethodRS512, rsaKey, rsaJWK},
		{jwt.SigningMethodPS256, rsaKey, rsaJWK}, {jwt.SigningMethodPS384, rsaKey, rsaJWK}, {jwt.SigningMethodPS512, rsaKey, rsaJWK},
		{jwt.SigningMethodES256, p256, p256JWK}, {jwt.SigningMethodES384, p384, p384JWK}, {jwt.SigningMethodES512, p521, p521JWK},
		{jwt.SigningMethodEdDSA, edKey, edJWK},
	}
	if len(cases) != len(asymmetricMethods) {
		t.Fatalf("%d cases for %d allowed methods", len(cases), len(asymmetricMethods))
	}
	for _, tc := range cases {
		t.Run(tc.method.Alg(), func(t *testing.T) {
			server := serveJWKS(t, `{"keys":[`+tc.jwk+`]}`)
			token := jwt.NewWithClaims(tc.method, jwt.MapClaims{"sub": "user-123", "exp": time.Now().Add(time.Hour).Unix()})
			token.Header["kid"] = "k"
			signed, err := token.SignedString(tc.key)
			if err != nil {
				t.Fatalf("sign: %v", err)
			}
			if sub, err := Validate(signed, []string{server.URL}); err != nil || sub != "user-123" {
				t.Errorf("Validate(%s) = %q, %v; want user-123", tc.method.Alg(), sub, err)
			}
		})
	}
}
