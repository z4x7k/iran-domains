-- +goose Up
CREATE TABLE users_rate_limit (
	the_user_id BIGINT NOT NULL UNIQUE,
	last_access_ts BIGINT NOT NULL,
	the_count BIGINT NOT NULL
);
CREATE INDEX users_rate_limit_the_user_id_IDX ON users_rate_limit (the_user_id ASC);

-- +goose Down
DROP TABLE users_rate_limit;
