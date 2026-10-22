-- +goose Up
ALTER TABLE shld_encryption_parts ADD COLUMN key_check VARCHAR(64) DEFAULT NULL;
ALTER TABLE shld_encryption_parts ADD COLUMN pending_part VARCHAR(255) DEFAULT NULL;
ALTER TABLE shld_encryption_parts ADD COLUMN pending_key_check VARCHAR(64) DEFAULT NULL;
ALTER TABLE shld_encryption_parts ADD COLUMN pending_expires_at TIMESTAMP DEFAULT NULL;
ALTER TABLE shld_projects ADD COLUMN pending_api_key VARCHAR(255) DEFAULT NULL;
ALTER TABLE shld_projects ADD COLUMN pending_api_key_expires_at TIMESTAMP DEFAULT NULL;
CREATE UNIQUE INDEX idx_projects_pending_api_key ON shld_projects(pending_api_key) WHERE pending_api_key IS NOT NULL;
-- +goose StatementBegin
-- +goose StatementEnd

-- +goose Down
DROP INDEX IF EXISTS idx_projects_pending_api_key;
ALTER TABLE shld_projects DROP COLUMN IF EXISTS pending_api_key_expires_at;
ALTER TABLE shld_projects DROP COLUMN IF EXISTS pending_api_key;
ALTER TABLE shld_encryption_parts DROP COLUMN IF EXISTS pending_expires_at;
ALTER TABLE shld_encryption_parts DROP COLUMN IF EXISTS pending_key_check;
ALTER TABLE shld_encryption_parts DROP COLUMN IF EXISTS pending_part;
ALTER TABLE shld_encryption_parts DROP COLUMN IF EXISTS key_check;
-- +goose StatementBegin
-- +goose StatementEnd
