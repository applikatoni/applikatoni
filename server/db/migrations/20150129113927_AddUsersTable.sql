
-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied


CREATE TABLE users (
  id INTEGER PRIMARY KEY,
  name TEXT,
  access_token TEXT,
  avatar_url TEXT
);

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back

DROP TABLE users;
