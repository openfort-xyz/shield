package cypher

import (
	"errors"
	"testing"

	"github.com/openfort-xyz/shield/pkg/random"
)

func TestDecryptRejectsWrongKey(t *testing.T) {
	key, err := random.GenerateRandomString(32)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	otherKey, err := random.GenerateRandomString(32)
	if err != nil {
		t.Fatalf("generate other key: %v", err)
	}

	encrypted, err := Encrypt("secret", key)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	plaintext, err := Decrypt(encrypted, key)
	if err != nil {
		t.Fatalf("decrypt with the right key: %v", err)
	}
	if plaintext != "secret" {
		t.Fatalf("decrypt returned %q, want %q", plaintext, "secret")
	}

	_, err = Decrypt(encrypted, otherKey)
	if !errors.Is(err, ErrAuthenticationFailed) {
		t.Fatalf("decrypt with the wrong key returned %v, want ErrAuthenticationFailed", err)
	}

	_, err = Decrypt("c2hvcnQ=", key)
	if err == nil || errors.Is(err, ErrAuthenticationFailed) {
		t.Fatalf("truncated ciphertext returned %v, want an error other than ErrAuthenticationFailed", err)
	}
}
