-- +goose Up
ALTER TABLE shld_shares ADD COLUMN entropy VARCHAR(255) DEFAULT 'none';
UPDATE shld_shares SET entropy = CASE WHEN user_entropy = TRUE THEN 'user' ELSE 'none' END;
ALTER TABLE shld_shares DROP COLUMN user_entropy;
-- +goose StatementBegin
-- +goose StatementEnd

-- +goose Down
ALTER TABLE shld_shares ADD COLUMN user_entropy BOOLEAN DEFAULT FALSE;
UPDATE shld_shares SET user_entropy = CASE WHEN entropy = 'user' THEN TRUE ELSE FALSE END;
ALTER TABLE shld_shares DROP COLUMN entropy;
-- +goose StatementBegin
-- +goose StatementEnd
