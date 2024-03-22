-- +goose Up
CREATE TABLE IF NOT EXISTS shld_projects (
    id VARCHAR(36) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    api_key VARCHAR(255) NOT NULL,
    api_secret VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP DEFAULT NULL
);

CREATE TABLE IF NOT EXISTS shld_providers (
    id VARCHAR(36) PRIMARY KEY,
    project_id VARCHAR(36) NOT NULL,
    type ENUM('OPENFORT', 'SUPABASE', 'CUSTOM') NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP DEFAULT NULL
);
ALTER TABLE shld_providers ADD CONSTRAINT fk_provider_project FOREIGN KEY (project_id) REFERENCES shld_projects(id) ON DELETE CASCADE;

CREATE TABLE IF NOT EXISTS shld_openfort_providers (
    provider_id VARCHAR(36) PRIMARY KEY,
    publishable_key VARCHAR(255) NOT NULL
);
ALTER TABLE shld_openfort_providers ADD CONSTRAINT fk_openfort_provider FOREIGN KEY (provider_id) REFERENCES shld_providers(id) ON DELETE CASCADE;

CREATE TABLE IF NOT EXISTS shld_supabase_providers (
    provider_id VARCHAR(36) PRIMARY KEY,
    supabase_project VARCHAR(255) NOT NULL
);
ALTER TABLE shld_supabase_providers ADD CONSTRAINT fk_supabase_provider FOREIGN KEY (provider_id) REFERENCES shld_providers(id) ON DELETE CASCADE;

CREATE TABLE IF NOT EXISTS shld_custom_providers (
    provider_id VARCHAR(36) PRIMARY KEY,
    jwk_url VARCHAR(255) NOT NULL
);
ALTER TABLE shld_custom_providers ADD CONSTRAINT fk_custom_provider FOREIGN KEY (provider_id) REFERENCES shld_providers(id) ON DELETE CASCADE;

CREATE TABLE IF NOT EXISTS shld_users (
    id VARCHAR(36) PRIMARY KEY,
    project_id VARCHAR(36) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP DEFAULT NULL
);
ALTER TABLE shld_users ADD CONSTRAINT fk_user_project FOREIGN KEY (project_id) REFERENCES shld_projects(id) ON DELETE CASCADE;

CREATE TABLE IF NOT EXISTS shld_external_users (
    id VARCHAR(36) PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL,
    external_user_id VARCHAR(255) NOT NULL,
    provider_id VARCHAR(36) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP DEFAULT NULL
);
ALTER TABLE shld_external_users ADD CONSTRAINT fk_external_user_user FOREIGN KEY (user_id) REFERENCES shld_users(id) ON DELETE CASCADE;
ALTER TABLE shld_external_users ADD CONSTRAINT fk_external_user_provider FOREIGN KEY (provider_id) REFERENCES shld_providers(id) ON DELETE CASCADE;

CREATE TABLE IF NOT EXISTS shld_shares (
    id VARCHAR(36) PRIMARY KEY,
    data VARCHAR(255) NOT NULL,
    user_id VARCHAR(36) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP DEFAULT NULL
);
ALTER TABLE shld_shares ADD CONSTRAINT fk_share_user FOREIGN KEY (user_id) REFERENCES shld_users(id) ON DELETE CASCADE;

-- +goose StatementBegin
-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS shld_shares;
DROP TABLE IF EXISTS shld_external_users;
DROP TABLE IF EXISTS shld_users;
DROP TABLE IF EXISTS shld_custom_providers;
DROP TABLE IF EXISTS shld_supabase_providers;
DROP TABLE IF EXISTS shld_openfort_providers;
DROP TABLE IF EXISTS shld_providers;
DROP TABLE IF EXISTS shld_projects;
-- +goose StatementBegin
-- +goose StatementEnd
