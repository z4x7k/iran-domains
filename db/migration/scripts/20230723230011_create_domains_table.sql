-- +goose Up
CREATE TABLE domains (
	domain TEXT NOT NULL UNIQUE,
	created_ts BIGINT NOT NULL
);
CREATE INDEX domains_domain_IDX ON domains (domain ASC);

-- +goose Down
DROP TABLE domains;
