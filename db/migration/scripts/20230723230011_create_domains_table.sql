-- +goose Up
CREATE TABLE domains (
	domain TEXT NOT NULL PRIMARY KEY,
	created_ts BIGINT NOT NULL,
  created_by_id BIGINT NOT NULL
);

-- +goose Down
DROP TABLE domains;
