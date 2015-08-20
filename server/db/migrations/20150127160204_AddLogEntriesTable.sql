
-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied

CREATE TABLE log_entries (
  id INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
  deployment_id INTEGER,
  entry_type TEXT,
  origin TEXT,
  message TEXT,
  timestamp DATETIME,
  created_at DATETIME
);


-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back

DROP TABLE log_entries;
