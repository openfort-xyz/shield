-- +goose Up
CREATE INDEX idx_external_user_provider_deleted ON shld_external_users (external_user_id, provider_id, deleted_at);
-- +goose StatementBegin
-- +goose StatementEnd

-- +goose Down
DROP INDEX idx_external_user_provider_deleted ON shld_external_users;
-- +goose StatementBegin
-- +goose StatementEnd
