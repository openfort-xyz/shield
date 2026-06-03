-- +goose Up
ALTER TABLE shld_shares ALTER COLUMN data TYPE VARCHAR(500);
-- +goose StatementBegin
-- +goose StatementEnd

-- +goose Down
ALTER TABLE shld_shares ALTER COLUMN data TYPE VARCHAR(255);
-- +goose StatementBegin
-- +goose StatementEnd
