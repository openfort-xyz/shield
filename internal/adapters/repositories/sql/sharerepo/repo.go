package sharerepo

import (
	"context"
	"errors"
	"log/slog"

	domainErrors "github.com/openfort-xyz/shield/internal/core/domain/errors"

	"github.com/google/uuid"
	"github.com/openfort-xyz/shield/internal/adapters/repositories/sql"
	"github.com/openfort-xyz/shield/internal/core/domain/share"
	"github.com/openfort-xyz/shield/internal/core/ports/repositories"
	"github.com/openfort-xyz/shield/pkg/cypher"
	"github.com/openfort-xyz/shield/pkg/logger"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type repository struct {
	db     *sql.Client
	logger *slog.Logger
	parser *parser
}

var _ repositories.ShareRepository = (*repository)(nil)

const normalizedReference = "COALESCE(NULLIF(shld_shares.reference, ''), '" + share.DefaultReference + "')"
const oldestShareFirst = "shld_shares.created_at ASC NULLS LAST, shld_shares.id ASC"

func shareReference(reference *string) string {
	if reference == nil || *reference == "" {
		return share.DefaultReference
	}
	return *reference
}

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
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := checkGuard(tx, shr.EncryptionGuard); err != nil {
			return err
		}
		if dbShr.UserID != nil {
			ref := shareReference(dbShr.Reference)
			// Serialize the existence check and insert for this owner and reference.
			if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtext(?), hashtext(?))", "shield-share:"+*dbShr.UserID, ref).Error; err != nil {
				return err
			}
			var exists bool
			if err := tx.Raw(`SELECT EXISTS (
				SELECT 1 FROM shld_shares WHERE user_id = ? AND deleted_at IS NULL AND `+normalizedReference+` = ?
			)`, *dbShr.UserID, ref).Scan(&exists).Error; err != nil {
				return err
			}
			if exists {
				return domainErrors.ErrShareAlreadyExists
			}
		}
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

func (r *repository) sharesForKeychain(ctx context.Context, keychainID string) *gorm.DB {
	return r.db.WithContext(ctx).Preload("PasskeyReference").
		Joins("JOIN shld_keychains ON shld_keychains.id = shld_shares.keychain_id AND shld_keychains.user_id = shld_shares.user_id").
		Where("shld_shares.keychain_id = ? AND shld_keychains.deleted_at IS NULL", keychainID).
		Order(oldestShareFirst)
}

func (r *repository) ListByKeychainID(ctx context.Context, keychainID string) ([]*share.Share, error) {
	r.logger.InfoContext(ctx, "listing shares", slog.String("keychain_id", keychainID))

	var dbShares []*Share
	err := r.sharesForKeychain(ctx, keychainID).Find(&dbShares).Error
	if err != nil {
		r.logger.ErrorContext(ctx, "error listing shares", logger.Error(err))
		return nil, err
	}

	var shares []*share.Share
	seen := make(map[string]bool, len(dbShares))
	for _, dbShr := range dbShares {
		ref := shareReference(dbShr.Reference)
		if seen[ref] {
			continue
		}
		seen[ref] = true
		shares = append(shares, r.parser.toDomain(dbShr))
	}

	return shares, nil
}

func (r *repository) GetByReferenceAndKeychain(ctx context.Context, reference, keychainID string) (*share.Share, error) {
	r.logger.InfoContext(ctx, "getting share", slog.String("reference", reference))

	dbShr := &Share{}
	err := r.sharesForKeychain(ctx, keychainID).Where(normalizedReference+" = ?", shareReference(&reference)).First(dbShr).Error
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

	// A project-wide reference is ambiguous when different users own it.
	var owners []string
	err := r.db.WithContext(ctx).Model(&Share{}).
		Joins("JOIN shld_users ON shld_shares.user_id = shld_users.id").
		Where(normalizedReference+" = ? AND shld_users.project_id = ?", shareReference(&reference), projectID).
		Distinct("shld_shares.user_id").Limit(2).Pluck("shld_shares.user_id", &owners).Error
	if err != nil {
		r.logger.ErrorContext(ctx, "error getting share by reference and project", logger.Error(err))
		return nil, err
	}
	if len(owners) == 0 {
		return nil, domainErrors.ErrShareNotFound
	}
	if len(owners) > 1 {
		return nil, domainErrors.ErrAmbiguousShareReference
	}
	return r.GetByReferenceForProjectAndUserID(ctx, reference, projectID, owners[0])
}

func (r *repository) GetByReferenceForProjectAndUserID(ctx context.Context, reference, projectID, userID string) (*share.Share, error) {
	r.logger.InfoContext(ctx, "getting share by reference for user", slog.String("reference", reference), slog.String("project_id", projectID))

	dbShr := &Share{}
	err := r.db.Preload("PasskeyReference").
		Joins("JOIN shld_users ON shld_shares.user_id = shld_users.id").
		Where(normalizedReference+" = ? AND shld_users.project_id = ? AND shld_shares.user_id = ?", shareReference(&reference), projectID, userID).
		Where("shld_shares.deleted_at IS NULL").
		Order(oldestShareFirst).
		First(dbShr).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainErrors.ErrShareNotFound
		}
		r.logger.ErrorContext(ctx, "error getting share by reference for user", logger.Error(err))
		return nil, err
	}

	return r.parser.toDomain(dbShr), nil
}

func (r *repository) GetByUserID(ctx context.Context, userID string) (*share.Share, error) {
	r.logger.InfoContext(ctx, "getting share", slog.String("user_id", userID))

	dbShr := &Share{}
	err := r.db.Preload("PasskeyReference").Where("user_id = ? AND keychain_id is NULL", userID).Order(oldestShareFirst).First(dbShr).Error
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
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := checkGuard(tx, shr.EncryptionGuard); err != nil {
			return err
		}
		if state := shr.ExpectedUpdate; state != nil {
			var current Share
			err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Select("id").
				Where("id = ? AND entropy = ? AND data = ?", dbShr.ID, r.parser.mapDomainEntropy[state.Entropy], state.Secret).
				Take(&current).Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return domainErrors.ErrShareChanged
			}
			if err != nil {
				return err
			}
		}
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

// underGuard runs fn, and when guard is set runs it in a transaction that
// first passes checkGuard.
func (r *repository) underGuard(ctx context.Context, guard *share.EncryptionGuard, fn func(tx *gorm.DB) error) error {
	if guard == nil {
		return fn(r.db.DB)
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := checkGuard(tx, guard); err != nil {
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
// does not order writes against it. A key check the row holds then must match
// the guard's key; it is compared after the part, so a write that raced a
// rotation reports the rotation. A nil guard checks nothing.
func checkGuard(tx *gorm.DB, guard *share.EncryptionGuard) error {
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
	if parts[0].KeyCheck != nil && !cypher.VerifyKeyCheck(guard.Key, *parts[0].KeyCheck) {
		return domainErrors.ErrInvalidEncryptionPart
	}
	return nil
}

func hasMigrated(tx *gorm.DB, projectID string) (bool, error) {
	var migrations int64
	err := tx.Raw(`SELECT count(*) FROM shld_shamir_migrations WHERE project_id = ? AND success`, projectID).Scan(&migrations).Error
	return migrations > 0, err
}

func (r *repository) logError(ctx context.Context, msg string, err error) {
	if errors.Is(err, domainErrors.ErrEncryptionPartChanged) || errors.Is(err, domainErrors.ErrInvalidEncryptionPart) {
		r.logger.WarnContext(ctx, msg, logger.Error(err))
		return
	}
	r.logger.ErrorContext(ctx, msg, logger.Error(err))
}
