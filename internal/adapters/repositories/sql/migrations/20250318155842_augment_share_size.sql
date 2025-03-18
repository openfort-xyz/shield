-- +goose Up
ALTER TABLE shld_shares MODIFY COLUMN data VARCHAR(500) NOT NULL;
-- +goose StatementBegin
-- +goose StatementEnd

-- +goose Down
ALTER TABLE shld_shares MODIFY COLUMN data VARCHAR(255) NOT NULL;
-- +goose StatementBegin
-- +goose StatementEnd