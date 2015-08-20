
-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
ALTER TABLE deployments ADD COLUMN branch TEXT;
UPDATE deployments SET branch = "";

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back
-- No Down migration here, since sqlite doesnt allow removing columns
-- http://stackoverflow.com/questions/8442147/how-to-delete-or-add-column-in-sqlite
SELECT 1;
