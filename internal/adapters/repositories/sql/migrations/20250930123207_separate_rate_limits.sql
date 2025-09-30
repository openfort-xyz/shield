-- +goose Up
ALTER TABLE shld_rate_limit
ADD COLUMN sms_requests_per_hour INT NOT NULL;
ALTER TABLE shld_rate_limit RENAME COLUMN requests_per_minute TO email_requests_per_hour;

UPDATE shld_rate_limit SET sms_requests_per_hour = 2;
UPDATE shld_rate_limit SET email_requests_per_hour = 120;

-- +goose StatementBegin
-- +goose StatementEnd

-- +goose Down
ALTER TABLE shld_rate_limit DROP COLUMN IF EXISTS sms_requests_per_hour;
ALTER TABLE shld_rate_limit DROP COLUMN IF EXISTS email_requests_per_hour;

-- +goose StatementBegin
-- +goose StatementEnd
