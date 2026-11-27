package encryption

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	domainErrors "github.com/openfort-xyz/shield/internal/core/domain/errors"
	"github.com/openfort-xyz/shield/pkg/contexter"
	"github.com/tidwall/buntdb"
)

func recoveryChangeBinding(ctx context.Context, shareID, ciphertext string) string {
	binding := strings.Join([]string{
		contexter.GetProjectID(ctx), contexter.GetProviderID(ctx), contexter.GetUserID(ctx), shareID, ciphertext,
	}, "\x00")
	digest := sha256.Sum256([]byte(binding))
	return hex.EncodeToString(digest[:])
}

// GrantRecoveryChange records the share version read with a verified OTP session.
func (e *encryptionFactory) GrantRecoveryChange(ctx context.Context, sessionID, shareID, ciphertext string) error {
	return e.encryptionPartsRepo.Set(ctx, "recovery-change:"+sessionID, recoveryChangeBinding(ctx, shareID, ciphertext),
		&buntdb.SetOptions{Expires: true, TTL: time.Minute})
}

func (e *encryptionFactory) ConsumeRecoveryChange(ctx context.Context, sessionID, shareID, ciphertext string) error {
	key := "recovery-change:" + sessionID
	binding, err := e.encryptionPartsRepo.Get(ctx, key)
	if errors.Is(err, domainErrors.ErrDataInDBNotFound) {
		return domainErrors.ErrOTPVerificationRequired
	}
	if err != nil {
		return err
	}
	if binding != recoveryChangeBinding(ctx, shareID, ciphertext) {
		return domainErrors.ErrInvalidEncryptionSession
	}
	err = e.encryptionPartsRepo.Delete(ctx, key)
	if errors.Is(err, domainErrors.ErrEncryptionPartNotFound) {
		return domainErrors.ErrOTPVerificationRequired
	}
	return err
}
