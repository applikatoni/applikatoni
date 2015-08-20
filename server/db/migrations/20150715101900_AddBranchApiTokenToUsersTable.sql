
-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
ALTER TABLE users ADD COLUMN api_token TEXT;
UPDATE users SET api_token = "";

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back
SELECT 1;
