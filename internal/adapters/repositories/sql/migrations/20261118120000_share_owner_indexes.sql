-- +goose Up
CREATE INDEX IF NOT EXISTS idx_shld_shares_user_id ON shld_shares(user_id);
CREATE INDEX IF NOT EXISTS idx_shld_users_project_id ON shld_users(project_id);
-- +goose StatementBegin
-- +goose StatementEnd

-- +goose Down
DROP INDEX IF EXISTS idx_shld_users_project_id;
DROP INDEX IF EXISTS idx_shld_shares_user_id;
-- +goose StatementBegin
-- +goose StatementEnd
