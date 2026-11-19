-- +goose NO TRANSACTION
-- Index the FK columns hit by the ON DELETE CASCADE / SET NULL chain of
-- DELETE FROM shld_projects that 20261118120000_share_owner_indexes does not
-- cover. Postgres does not index FK columns, so each cascaded row scanned
-- the whole child table (hard delete took > 10 s).
--
-- CONCURRENTLY keeps writes open while the indexes build; it cannot run in a
-- transaction, hence NO TRANSACTION. The DROP first clears an INVALID index
-- left by an interrupted build, so a re-run of this migration is safe.

-- +goose Up
DROP INDEX CONCURRENTLY IF EXISTS idx_shld_shares_keychain_id;
CREATE INDEX CONCURRENTLY idx_shld_shares_keychain_id ON shld_shares(keychain_id);

DROP INDEX CONCURRENTLY IF EXISTS idx_shld_external_users_user_id;
CREATE INDEX CONCURRENTLY idx_shld_external_users_user_id ON shld_external_users(user_id);

DROP INDEX CONCURRENTLY IF EXISTS idx_shld_external_users_provider_id;
CREATE INDEX CONCURRENTLY idx_shld_external_users_provider_id ON shld_external_users(provider_id);

DROP INDEX CONCURRENTLY IF EXISTS idx_shld_notifications_project_id;
CREATE INDEX CONCURRENTLY idx_shld_notifications_project_id ON shld_notifications(project_id);

DROP INDEX CONCURRENTLY IF EXISTS idx_shld_encryption_parts_project_id;
CREATE INDEX CONCURRENTLY idx_shld_encryption_parts_project_id ON shld_encryption_parts(project_id);

DROP INDEX CONCURRENTLY IF EXISTS idx_shld_rate_limit_project_id;
CREATE INDEX CONCURRENTLY idx_shld_rate_limit_project_id ON shld_rate_limit(project_id);

-- +goose Down
DROP INDEX CONCURRENTLY IF EXISTS idx_shld_rate_limit_project_id;
DROP INDEX CONCURRENTLY IF EXISTS idx_shld_encryption_parts_project_id;
DROP INDEX CONCURRENTLY IF EXISTS idx_shld_notifications_project_id;
DROP INDEX CONCURRENTLY IF EXISTS idx_shld_external_users_provider_id;
DROP INDEX CONCURRENTLY IF EXISTS idx_shld_external_users_user_id;
DROP INDEX CONCURRENTLY IF EXISTS idx_shld_shares_keychain_id;
