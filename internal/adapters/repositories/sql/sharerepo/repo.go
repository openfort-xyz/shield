package sharerepo

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	domainErrors "github.com/openfort-xyz/shield/internal/core/domain/errors"

	"github.com/google/uuid"
	"github.com/openfort-xyz/shield/internal/adapters/repositories/sql"
	"github.com/openfort-xyz/shield/internal/core/domain/share"
	"github.com/openfort-xyz/shield/internal/core/ports/repositories"
	"github.com/openfort-xyz/shield/pkg/cypher"
	"github.com/openfort-xyz/shield/pkg/logger"
	"gorm.io/gorm"
)

type repository struct {
	db     *sql.Client
	logger *slog.Logger
	parser *parser
}

var _ repositories.ShareRepository = (*repository)(nil)

func New(db *sql.Client) repositories.ShareRepository {
	return &repository{
		db:     db,
		logger: logger.New("share_repository"),
		parser: newParser(),
	}
}

func (r *repository) Create(ctx context.Context, shr *share.Share) error {
	r.logger.InfoContext(ctx, "creating share")

	if shr.ID == "" {
		shr.ID = uuid.NewString()
	}

	dbShr := r.parser.toDatabase(shr)
	err := r.underGuard(ctx, shr.EncryptionGuard, func(tx *gorm.DB) error {
		return tx.Create(dbShr).Error
	})
	if err != nil {
		r.logError(ctx, "error creating share", err)
		return err
	}

	return nil
}

func (r *repository) Get(ctx context.Context, shareID string) (*share.Share, error) {
	r.logger.InfoContext(ctx, "getting share", slog.String("id", shareID))

	dbShr := &Share{}
	err := r.db.Preload("PasskeyReference").Where("id = ?", shareID).First(dbShr).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainErrors.ErrShareNotFound
		}
		r.logger.ErrorContext(ctx, "error getting share", logger.Error(err))
		return nil, err
	}

	return r.parser.toDomain(dbShr), nil
}

func (r *repository) ListByKeychainID(ctx context.Context, keychainID string) ([]*share.Share, error) {
	r.logger.InfoContext(ctx, "listing shares", slog.String("keychain_id", keychainID))

	var dbShares []*Share
	err := r.db.Preload("PasskeyReference").Where("keychain_id = ?", keychainID).Find(&dbShares).Error
	if err != nil {
		r.logger.ErrorContext(ctx, "error listing shares", logger.Error(err))
		return nil, err
	}

	var shares []*share.Share
	for _, dbShr := range dbShares {
		shares = append(shares, r.parser.toDomain(dbShr))
	}

	return shares, nil
}

func (r *repository) GetByReference(ctx context.Context, reference string) (*share.Share, error) {
	r.logger.InfoContext(ctx, "getting share", slog.String("reference", reference))

	dbShr := &Share{}
	err := r.db.Preload("PasskeyReference").Where("reference = ?", reference).First(dbShr).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainErrors.ErrShareNotFound
		}
		r.logger.ErrorContext(ctx, "error getting share", logger.Error(err))
		return nil, err
	}

	return r.parser.toDomain(dbShr), nil
}

func (r *repository) GetByReferenceAndKeychain(ctx context.Context, reference, keychainID string) (*share.Share, error) {
	r.logger.InfoContext(ctx, "getting share", slog.String("reference", reference))

	dbShr := &Share{}
	err := r.db.Preload("PasskeyReference").Where("reference = ? AND keychain_id = ?", reference, keychainID).First(dbShr).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainErrors.ErrShareNotFound
		}
		r.logger.ErrorContext(ctx, "error getting share", logger.Error(err))
		return nil, err
	}

	return r.parser.toDomain(dbShr), nil
}

func (r *repository) GetByReferenceAndProjectID(ctx context.Context, reference, projectID string) (*share.Share, error) {
	r.logger.InfoContext(ctx, "getting share by reference and project", slog.String("reference", reference), slog.String("project_id", projectID))

	dbShr := &Share{}
	err := r.db.Preload("PasskeyReference").
		Joins("JOIN shld_users ON shld_shares.user_id = shld_users.id").
		Where("shld_shares.reference = ? AND shld_users.project_id = ?", reference, projectID).
		Where("shld_shares.deleted_at IS NULL").
		First(dbShr).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainErrors.ErrShareNotFound
		}
		r.logger.ErrorContext(ctx, "error getting share by reference and project", logger.Error(err))
		return nil, err
	}

	return r.parser.toDomain(dbShr), nil
}

func (r *repository) GetByUserID(ctx context.Context, userID string) (*share.Share, error) {
	r.logger.InfoContext(ctx, "getting share", slog.String("user_id", userID))

	dbShr := &Share{}
	err := r.db.Preload("PasskeyReference").Where("user_id = ? AND keychain_id is NULL", userID).First(dbShr).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainErrors.ErrShareNotFound
		}
		r.logger.ErrorContext(ctx, "error getting share", logger.Error(err))
		return nil, err
	}

	return r.parser.toDomain(dbShr), nil
}

func (r *repository) Delete(ctx context.Context, shareID string) error {
	r.logger.InfoContext(ctx, "deleting share", slog.String("id", shareID))

	err := r.db.Where("id = ?", shareID).Delete(&Share{}).Error
	if err != nil {
		r.logger.ErrorContext(ctx, "error deleting share", logger.Error(err))
		return err
	}

	return nil
}

func (r *repository) ListProjectIDAndEntropy(ctx context.Context, projectID string, entropy share.Entropy) ([]*share.Share, error) {
	r.logger.InfoContext(ctx, "listing shares", slog.String("project_id", projectID))

	var dbShares []*Share
	err := r.db.Preload("PasskeyReference").Joins("JOIN shld_users ON shld_shares.user_id = shld_users.id").
		Joins("JOIN shld_projects ON shld_users.project_id = shld_projects.id").
		Where("shld_projects.id = ?", projectID).
		Where("shld_shares.entropy = ?", r.parser.mapDomainEntropy[entropy]).
		Find(&dbShares).Error
	if err != nil {
		r.logger.ErrorContext(ctx, "error listing shares", logger.Error(err))
		return nil, err
	}

	var shares []*share.Share
	for _, dbShr := range dbShares {
		shares = append(shares, r.parser.toDomain(dbShr))
	}

	return shares, nil
}

func (r *repository) UpdateProjectEncryption(ctx context.Context, shareID, plain, encrypted string, guard *share.EncryptionGuard) error {
	r.logger.InfoContext(ctx, "updating share", slog.String("id", shareID))

	err := r.underGuard(ctx, guard, func(tx *gorm.DB) error {
		res := tx.Model(&Share{}).Where("id = ? AND entropy = ? AND data = ?", shareID, EntropyNone, plain).
			Updates(map[string]any{"data": encrypted, "entropy": EntropyProject})
		if res.Error == nil && res.RowsAffected == 0 {
			r.logger.WarnContext(ctx, "share changed since it was read; left as is", slog.String("id", shareID))
		}
		return res.Error
	})
	if err != nil {
		r.logError(ctx, "error updating share", err)
		return err
	}

	return nil
}

// Intentionally left out of ShareRepository interface
// since usage is only internal
func updatePasskeyReference(db *gorm.DB, passkeyReference *PasskeyReference) error {
	if passkeyReference != nil {
		// WHERE clause is necessary for GORM >= v2 (it won't figure out which field to change even if PK is provided otherwise)
		return db.Model(&PasskeyReference{}).Where("share_reference = ?", passkeyReference.ShareReference).Save(passkeyReference).Error
	}
	return nil
}

// Intentionally left out of ShareRepository interface
// since usage is only internal
func updateShare(db *gorm.DB, dbShr *Share) error {
	err := updatePasskeyReference(db, dbShr.PasskeyReference)

	if err != nil {
		return err
	}

	return db.
		Session(&gorm.Session{FullSaveAssociations: true}).
		Model(&Share{}).
		Where("id = ?", dbShr.ID).
		Updates(dbShr).Error
}

func (r *repository) Update(ctx context.Context, shr *share.Share) error {
	r.logger.InfoContext(ctx, "updating share", slog.String("id", shr.ID))

	dbShr := r.parser.toDatabase(shr)
	err := r.underGuard(ctx, shr.EncryptionGuard, func(tx *gorm.DB) error {
		return updateShare(tx, dbShr)
	})
	if err != nil {
		r.logError(ctx, "error updating share", err)
		return err
	}

	return nil
}

func (r *repository) GetShareStorageMethods(ctx context.Context) ([]*share.StorageMethod, error) {
	r.logger.InfoContext(ctx, "getting share storage methods")

	var dbMethods []ShareStorageMethod
	err := r.db.Find(&dbMethods).Error
	if err != nil {
		r.logger.ErrorContext(ctx, "error getting share storage methods", logger.Error(err))
		return nil, err
	}

	var methods []*share.StorageMethod
	for _, dbMethod := range dbMethods {
		methods = append(methods, r.parser.toDomainShareStorageMethod(&dbMethod))
	}

	return methods, nil
}

func (r *repository) GetSharesEncryptionForProjectAndReferences(ctx context.Context, projectID string, references []string) (map[string]share.RecoveryInfo, error) {
	if len(references) == 0 {
		return map[string]share.RecoveryInfo{}, nil
	}

	var queryResult []InfoByReference
	err := r.db.Table("shld_shares").
		Select("shld_shares.reference AS reference, shld_shares.entropy AS entropy, shld_passkey_references.passkey_id AS passkey_id, shld_passkey_references.passkey_env AS passkey_env").
		Joins("LEFT JOIN shld_passkey_references ON shld_shares.id = shld_passkey_references.share_reference").
		Joins("JOIN shld_users ON shld_shares.user_id = shld_users.id").
		Where("shld_shares.reference IN ?", references).
		Where("shld_users.project_id = ?", projectID).
		Where("shld_shares.deleted_at IS NULL").
		Find(&queryResult).Error

	if err != nil {
		r.logger.ErrorContext(ctx, "error getting encryption methods from references", logger.Error(err))
		return nil, err
	}

	result := map[string]share.RecoveryInfo{}

	for _, row := range queryResult {
		result[row.Reference] = share.RecoveryInfo{
			Entropy:    r.parser.mapEntropyDomain[row.Entropy],
			PasskeyID:  row.PasskeyID,
			PasskeyEnv: row.PasskeyEnv,
		}
	}

	return result, nil
}

func (r *repository) GetSharesEncryptionForProjectAndExternalUserIDs(ctx context.Context, projectID string, userIDs []string, reference *string) (map[string]share.RecoveryInfo, error) {
	if len(userIDs) == 0 {
		return map[string]share.RecoveryInfo{}, nil
	}

	var queryResult []InfoByUserID
	req := r.db.Table("shld_shares").
		Select("shld_external_users.external_user_id AS user_id, shld_shares.entropy AS entropy, shld_passkey_references.passkey_id AS passkey_id, shld_passkey_references.passkey_env AS passkey_env").
		Joins("LEFT JOIN shld_passkey_references ON shld_shares.id = shld_passkey_references.share_reference").
		Joins("JOIN shld_users ON shld_shares.user_id = shld_users.id").
		Joins("JOIN shld_external_users ON shld_external_users.user_id = shld_users.id").
		Where("shld_external_users.external_user_id IN ?", userIDs).
		Where("shld_users.project_id = ?", projectID).
		Where("shld_shares.deleted_at IS NULL").
		Limit(len(userIDs)) // Prevent misuse: non-legacy users might have multiple accounts

	if reference != nil {
		req = req.Where("shld_shares.reference = ?", *reference)
	}

	err := req.Find(&queryResult).Error

	if err != nil {
		r.logger.ErrorContext(ctx, "error getting encryption methods from users", logger.Error(err))
		return nil, err
	}

	result := map[string]share.RecoveryInfo{}

	for _, row := range queryResult {
		result[row.UserID] = share.RecoveryInfo{
			Entropy:    r.parser.mapEntropyDomain[row.Entropy],
			PasskeyID:  row.PasskeyID,
			PasskeyEnv: row.PasskeyEnv,
		}
	}

	return result, nil
}

// proofSampleSize bounds how many shares proveKey tries a key against.
const proofSampleSize = 1000

// proofFirstBatch is how many shares proveKey reads before the full sample.
// Postgres computes every row a query asks for whatever the client reads, so
// a key that decrypts the first shares read stops early only with a small
// first query.
const proofFirstBatch = 10

// underGuard runs fn, and when guard is set runs it in a transaction that
// first passes checkGuard. A key without a key check is proven once per
// guard, before the transaction and holding no lock; checkGuard applies the
// result.
func (r *repository) underGuard(ctx context.Context, guard *share.EncryptionGuard, fn func(tx *gorm.DB) error) error {
	if guard == nil {
		return fn(r.db.DB)
	}
	var proof error
	if guard.UnverifiedKey != "" && !guard.KeyProven {
		proof = r.proveKey(ctx, guard)
		guard.KeyProven = proof == nil
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := checkGuard(tx, guard, proof); err != nil {
			return err
		}
		return fn(tx)
	})
}

// checkGuard takes FOR SHARE on the project's encryption part row and fails
// with ErrEncryptionPartChanged unless it still holds the part the caller
// encrypted under. It reads the row key reconstruction reads (the first by
// id; a project can hold duplicates). A rotation commit holds FOR UPDATE on
// the part rows and replaces the part in place, so a write that read the part
// before the commit blocks here, then sees the new part and aborts; zero rows
// also abort (TestPartRowLockSemantics). A guard for a pre-migration key also
// aborts if a successful migration row exists once the part lock is granted.
// That keeps the write out of a migration only if the migration takes FOR
// UPDATE on the part rows before it lists shares and inserts its success row
// in the same transaction. The v0.4.2 migration job does neither, so this
// does not order writes against it. For a guard with an UnverifiedKey, a key
// check stored since the key was rebuilt decides, else proof, proveKey's
// result, does; both come after the part comparison, so a write that raced a
// rotation reports the rotation. A nil guard checks nothing.
func checkGuard(tx *gorm.DB, guard *share.EncryptionGuard, proof error) error {
	if guard == nil {
		return nil
	}
	var parts []struct {
		Part     string
		KeyCheck *string
	}
	err := tx.Raw(`SELECT part, key_check FROM shld_encryption_parts WHERE project_id = ? ORDER BY id LIMIT 1 FOR SHARE`, guard.ProjectID).Scan(&parts).Error
	if err != nil {
		return err
	}
	if len(parts) == 0 || parts[0].Part != string(guard.StoredPart) {
		return domainErrors.ErrEncryptionPartChanged
	}
	if !guard.Migrated {
		migrated, err := hasMigrated(tx, guard.ProjectID)
		if err != nil {
			return err
		}
		if migrated {
			return domainErrors.ErrEncryptionPartChanged
		}
	}
	if guard.UnverifiedKey == "" {
		return nil
	}
	if parts[0].KeyCheck != nil {
		if !cypher.VerifyKeyCheck(guard.UnverifiedKey, *parts[0].KeyCheck) {
			return domainErrors.ErrInvalidEncryptionPart
		}
		return nil
	}
	return proof
}

func hasMigrated(tx *gorm.DB, projectID string) (bool, error) {
	var migrations int64
	err := tx.Raw(`SELECT count(*) FROM shld_shamir_migrations WHERE project_id = ? AND success`, projectID).Scan(&migrations).Error
	return migrations > 0, err
}

// proveKey tries a key that was rebuilt without a key check against up to
// proofSampleSize of the project's project-entropy shares, soft-deleted ones
// included as in rotation, in whatever order the index walk yields them, and
// passes at the first that decrypts. A share that does not, malformed or
// under another key, is skipped, so one stray share does not lock out the
// project's part. A project with no such share has nothing to prove a key
// against, and its writes proceed as before key checks existed.
func (r *repository) proveKey(ctx context.Context, guard *share.EncryptionGuard) error {
	tried := 0
	for _, limit := range []int{proofFirstBatch, proofSampleSize} {
		proven, n, err := r.tryKey(ctx, guard, limit)
		if err != nil || proven {
			return err
		}
		tried += n
		if n < limit {
			break
		}
	}
	if tried == 0 {
		return nil
	}
	return fmt.Errorf("project %s: none of %d project shares tried decrypts under the key: %w",
		guard.ProjectID, tried, domainErrors.ErrEncryptionPartUnproven)
}

// tryKey reads up to limit of the project's project-entropy shares and
// reports whether one decrypts under the key, and how many it tried.
func (r *repository) tryKey(ctx context.Context, guard *share.EncryptionGuard, limit int) (bool, int, error) {
	rows, err := r.db.WithContext(ctx).Raw(`SELECT s.data FROM shld_shares s
		JOIN shld_users u ON s.user_id = u.id
		WHERE u.project_id = ? AND s.entropy = 'project'
		LIMIT ?`, guard.ProjectID, limit).Rows()
	if err != nil {
		return false, 0, err
	}
	defer rows.Close()

	tried := 0
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return false, tried, err
		}
		if _, err := cypher.Decrypt(data, guard.UnverifiedKey); err == nil {
			return true, tried, nil
		}
		tried++
	}
	return false, tried, rows.Err()
}

func (r *repository) logError(ctx context.Context, msg string, err error) {
	if errors.Is(err, domainErrors.ErrEncryptionPartChanged) || errors.Is(err, domainErrors.ErrInvalidEncryptionPart) {
		r.logger.WarnContext(ctx, msg, logger.Error(err))
		return
	}
	r.logger.ErrorContext(ctx, msg, logger.Error(err))
}
