-- +goose Up
-- Add the storage_method_id column to shld_shares table, defaulting to 0 and not null
ALTER TABLE shld_shares
ADD COLUMN storage_method_id INTEGER NOT NULL DEFAULT 0;

-- Update existing rows to have storage_method_id = 0
UPDATE shld_shares
SET storage_method_id = 0
WHERE storage_method_id IS NULL;

-- Remove the default value (if you don't want new rows to default to 0)
ALTER TABLE shld_shares
ALTER COLUMN storage_method_id DROP DEFAULT;

-- Add a foreign key constraint to shld_share_storage_methods(id)
ALTER TABLE shld_shares
ADD CONSTRAINT fk_shares_storage_method
FOREIGN KEY (storage_method_id) REFERENCES shld_share_storage_methods(id);
-- +goose Down
-- Remove the foreign key constraint
ALTER TABLE shld_shares
DROP CONSTRAINT IF EXISTS fk_shares_storage_method;
-- Remove the storage_method_id column
ALTER TABLE shld_shares
DROP COLUMN IF EXISTS storage_method_id;