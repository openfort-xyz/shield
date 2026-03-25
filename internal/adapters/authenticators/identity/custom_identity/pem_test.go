package cstmidty

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/openfort-xyz/shield/internal/core/domain/provider"
)

func generateRSAKeyPEM(t *testing.T) ([]byte, *rsa.PrivateKey) {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	pubBytes, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubBytes})
	return pubPEM, priv
}

func generateECDSAKeyPEM(t *testing.T) ([]byte, *ecdsa.PrivateKey) {
	t.Helper()
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	pubBytes, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubBytes})
	return pubPEM, priv
}

func generateEd25519KeyPEM(t *testing.T) ([]byte, ed25519.PrivateKey) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	pubBytes, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		t.Fatal(err)
	}
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubBytes})
	return pubPEM, priv
}

func signToken(t *testing.T, method jwt.SigningMethod, key interface{}, claims jwt.MapClaims) string {
	t.Helper()
	token := jwt.NewWithClaims(method, claims)
	signed, err := token.SignedString(key)
	if err != nil {
		t.Fatal(err)
	}
	return signed
}

func validClaims() jwt.MapClaims {
	return jwt.MapClaims{
		"sub": "user-123",
		"exp": time.Now().Add(time.Hour).Unix(),
	}
}

func TestGetKeyFuncFromPEM_RejectsUnknownKeyType(t *testing.T) {
	_, _, err := getKeyFuncFromPEM([]byte("anything"), provider.KeyTypeUnknown)
	if err == nil {
		t.Fatal("expected error for unknown key type")
	}
}

func TestValidatePEM_RSA_AcceptsRS256(t *testing.T) {
	pubPEM, priv := generateRSAKeyPEM(t)
	factory := &CustomIdentityFactory{
		config: &provider.CustomConfig{
			PEM:     string(pubPEM),
			KeyType: provider.KeyTypeRSA,
		},
	}

	token := signToken(t, jwt.SigningMethodRS256, priv, validClaims())
	sub, err := factory.validatePEM(token)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if sub != "user-123" {
		t.Fatalf("expected sub=user-123, got: %s", sub)
	}
}

func TestValidatePEM_RSA_AcceptsPS256(t *testing.T) {
	pubPEM, priv := generateRSAKeyPEM(t)
	factory := &CustomIdentityFactory{
		config: &provider.CustomConfig{
			PEM:     string(pubPEM),
			KeyType: provider.KeyTypeRSA,
		},
	}

	token := signToken(t, jwt.SigningMethodPS256, priv, validClaims())
	sub, err := factory.validatePEM(token)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if sub != "user-123" {
		t.Fatalf("expected sub=user-123, got: %s", sub)
	}
}

func TestValidatePEM_RSA_RejectsES256(t *testing.T) {
	pubPEM, _ := generateRSAKeyPEM(t)
	_, ecPriv := generateECDSAKeyPEM(t)
	factory := &CustomIdentityFactory{
		config: &provider.CustomConfig{
			PEM:     string(pubPEM),
			KeyType: provider.KeyTypeRSA,
		},
	}

	token := signToken(t, jwt.SigningMethodES256, ecPriv, validClaims())
	_, err := factory.validatePEM(token)
	if err == nil {
		t.Fatal("expected error when using ES256 against RSA provider")
	}
}

func TestValidatePEM_RSA_RejectsHS256(t *testing.T) {
	pubPEM, _ := generateRSAKeyPEM(t)
	factory := &CustomIdentityFactory{
		config: &provider.CustomConfig{
			PEM:     string(pubPEM),
			KeyType: provider.KeyTypeRSA,
		},
	}

	// Sign with HMAC using the PEM bytes as secret (algorithm confusion attack)
	token := signToken(t, jwt.SigningMethodHS256, pubPEM, validClaims())
	_, err := factory.validatePEM(token)
	if err == nil {
		t.Fatal("expected error when using HS256 against RSA provider (algorithm confusion)")
	}
}

func TestValidatePEM_ECDSA_AcceptsES256(t *testing.T) {
	pubPEM, priv := generateECDSAKeyPEM(t)
	factory := &CustomIdentityFactory{
		config: &provider.CustomConfig{
			PEM:     string(pubPEM),
			KeyType: provider.KeyTypeECDSA,
		},
	}

	token := signToken(t, jwt.SigningMethodES256, priv, validClaims())
	sub, err := factory.validatePEM(token)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if sub != "user-123" {
		t.Fatalf("expected sub=user-123, got: %s", sub)
	}
}

func TestValidatePEM_ECDSA_RejectsRS256(t *testing.T) {
	pubPEM, _ := generateECDSAKeyPEM(t)
	_, rsaPriv := generateRSAKeyPEM(t)
	factory := &CustomIdentityFactory{
		config: &provider.CustomConfig{
			PEM:     string(pubPEM),
			KeyType: provider.KeyTypeECDSA,
		},
	}

	token := signToken(t, jwt.SigningMethodRS256, rsaPriv, validClaims())
	_, err := factory.validatePEM(token)
	if err == nil {
		t.Fatal("expected error when using RS256 against ECDSA provider")
	}
}

func TestValidatePEM_Ed25519_AcceptsEdDSA(t *testing.T) {
	pubPEM, priv := generateEd25519KeyPEM(t)
	factory := &CustomIdentityFactory{
		config: &provider.CustomConfig{
			PEM:     string(pubPEM),
			KeyType: provider.KeyTypeEd25519,
		},
	}

	token := signToken(t, jwt.SigningMethodEdDSA, priv, validClaims())
	sub, err := factory.validatePEM(token)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if sub != "user-123" {
		t.Fatalf("expected sub=user-123, got: %s", sub)
	}
}

func TestValidatePEM_MissingSub_ReturnsError(t *testing.T) {
	pubPEM, priv := generateRSAKeyPEM(t)
	factory := &CustomIdentityFactory{
		config: &provider.CustomConfig{
			PEM:     string(pubPEM),
			KeyType: provider.KeyTypeRSA,
		},
	}

	claims := jwt.MapClaims{
		"exp": time.Now().Add(time.Hour).Unix(),
	}
	token := signToken(t, jwt.SigningMethodRS256, priv, claims)
	_, err := factory.validatePEM(token)
	if err == nil {
		t.Fatal("expected error for missing sub claim")
	}
}

func TestValidatePEM_EmptySub_ReturnsError(t *testing.T) {
	pubPEM, priv := generateRSAKeyPEM(t)
	factory := &CustomIdentityFactory{
		config: &provider.CustomConfig{
			PEM:     string(pubPEM),
			KeyType: provider.KeyTypeRSA,
		},
	}

	claims := jwt.MapClaims{
		"sub": "",
		"exp": time.Now().Add(time.Hour).Unix(),
	}
	token := signToken(t, jwt.SigningMethodRS256, priv, claims)
	_, err := factory.validatePEM(token)
	if err == nil {
		t.Fatal("expected error for empty sub claim")
	}
}

func TestValidatePEM_NonStringSub_ReturnsError(t *testing.T) {
	pubPEM, priv := generateRSAKeyPEM(t)
	factory := &CustomIdentityFactory{
		config: &provider.CustomConfig{
			PEM:     string(pubPEM),
			KeyType: provider.KeyTypeRSA,
		},
	}

	claims := jwt.MapClaims{
		"sub": 12345,
		"exp": time.Now().Add(time.Hour).Unix(),
	}
	token := signToken(t, jwt.SigningMethodRS256, priv, claims)
	_, err := factory.validatePEM(token)
	if err == nil {
		t.Fatal("expected error for non-string sub claim")
	}
}

func TestValidatePEM_ExpiredToken_ReturnsError(t *testing.T) {
	pubPEM, priv := generateRSAKeyPEM(t)
	factory := &CustomIdentityFactory{
		config: &provider.CustomConfig{
			PEM:     string(pubPEM),
			KeyType: provider.KeyTypeRSA,
		},
	}

	claims := jwt.MapClaims{
		"sub": "user-123",
		"exp": time.Now().Add(-time.Hour).Unix(),
	}
	token := signToken(t, jwt.SigningMethodRS256, priv, claims)
	_, err := factory.validatePEM(token)
	if err == nil {
		t.Fatal("expected error for expired token")
	}
}

func TestValidMethodsForKeyType(t *testing.T) {
	tests := []struct {
		name    string
		keyType provider.KeyType
		want    []string
	}{
		{"RSA", provider.KeyTypeRSA, []string{"RS256", "RS384", "RS512", "PS256", "PS384", "PS512"}},
		{"ECDSA", provider.KeyTypeECDSA, []string{"ES256", "ES384", "ES512"}},
		{"Ed25519", provider.KeyTypeEd25519, []string{"EdDSA"}},
		{"Unknown", provider.KeyTypeUnknown, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validMethodsForKeyType(tt.keyType)
			if len(got) != len(tt.want) {
				t.Fatalf("expected %v, got %v", tt.want, got)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("expected %v, got %v", tt.want, got)
				}
			}
		})
	}
}
