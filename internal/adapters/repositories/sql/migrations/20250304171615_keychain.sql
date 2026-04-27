-- +goose Up
CREATE TABLE IF NOT EXISTS shld_keychains (
    id VARCHAR(36) PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP DEFAULT NULL
);
CREATE INDEX idx_user_id ON shld_keychains(user_id);

ALTER TABLE shld_keychains ADD CONSTRAINT fk_keychain_user FOREIGN KEY (user_id) REFERENCES shld_users(id) ON DELETE CASCADE;
ALTER TABLE shld_shares ALTER COLUMN user_id DROP NOT NULL;
ALTER TABLE shld_shares ADD COLUMN keychain_id VARCHAR(36) DEFAULT NULL;
ALTER TABLE shld_shares ADD COLUMN reference VARCHAR(255) DEFAULT NULL;
ALTER TABLE shld_shares ADD CONSTRAINT fk_share_keychain FOREIGN KEY (keychain_id) REFERENCES shld_keychains(id) ON DELETE SET NULL;
-- +goose StatementBegin
-- +goose StatementEnd

-- +goose Down
ALTER TABLE shld_shares DROP CONSTRAINT IF EXISTS fk_share_keychain;
ALTER TABLE shld_shares DROP COLUMN reference;
ALTER TABLE shld_shares DROP COLUMN keychain_id;
ALTER TABLE shld_shares ALTER COLUMN user_id SET NOT NULL;
ALTER TABLE shld_keychains DROP CONSTRAINT fk_keychain_user;
DROP TABLE IF EXISTS shld_keychains;
-- +goose StatementBegin
-- +goose StatementEnd
