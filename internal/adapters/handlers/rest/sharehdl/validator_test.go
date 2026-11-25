package sharehdl

import (
	"fmt"
	"testing"
)

func TestUserEntropyLegacySHALengths(t *testing.T) {
	for _, digest := range []string{"SHA-256", "SHA-512"} {
		for _, length := range []int{16, 32, 64, 128, 256, 512} {
			t.Run(fmt.Sprintf("%s/%d", digest, length), func(t *testing.T) {
				s := &Share{Secret: "ciphertext", Entropy: EntropyUser, Salt: "salt", Iterations: 1000, Length: length, Digest: digest}
				if err := newValidator().validateShare(s); err != nil {
					t.Fatalf("legacy metadata rejected: %v", err)
				}
				if s.Length != length || s.Digest != digest {
					t.Fatal("validation must preserve decryption metadata")
				}
			})
		}
	}
}

func TestUserEntropyRejectsInvalidParameters(t *testing.T) {
	for _, length := range []int{-256, 0, 15, 65, 127, 129, 513} {
		if err := validateUserEntropyParams("salt", 1000, length, "SHA-256"); err == nil {
			t.Errorf("accepted invalid length %d", length)
		}
	}
	for _, iterations := range []int{-1, 0} {
		if err := validateUserEntropyParams("salt", iterations, 256, "SHA-256"); err == nil {
			t.Errorf("accepted iterations %d", iterations)
		}
	}
	if validateUserEntropyParams("", 1000, 256, "SHA-256") == nil ||
		validateUserEntropyParams("salt", 1000, 256, "unsupported") == nil ||
		validateUserEntropyParams("salt", 1000, 256, "argon2") == nil {
		t.Fatal("accepted invalid salt, digest or Argon2 length")
	}
}
