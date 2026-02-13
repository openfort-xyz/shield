-- +goose Up
CREATE UNIQUE INDEX idx_projects_api_key ON shld_projects(api_key);
CREATE INDEX idx_providers_project_type ON shld_providers(project_id, type);
-- +goose StatementBegin
-- +goose StatementEnd

-- +goose Down
DROP INDEX idx_projects_api_key ON shld_projects;
DROP INDEX idx_providers_project_type ON shld_providers;
-- +goose StatementBegin
-- +goose StatementEnd
