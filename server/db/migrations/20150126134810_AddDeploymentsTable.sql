
-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
CREATE TABLE deployments (
  id INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
  user_id INTEGER,
  application_name TEXT,
  target_name TEXT,
  commit_sha TEXT,
  comment TEXT,
  state TEXT,
  created_at DATETIME
);

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back
DROP TABLE deployments;
