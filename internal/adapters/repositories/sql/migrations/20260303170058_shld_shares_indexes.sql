-- +goose Up
CREATE INDEX idx_shld_shares_reference_deleted_at ON shld_shares(reference, deleted_at);
-- +goose StatementBegin
-- +goose StatementEnd

-- +goose Down
DROP INDEX IF EXISTS idx_shld_shares_reference_deleted_at;
-- +goose StatementBegin
-- +goose StatementEnd
