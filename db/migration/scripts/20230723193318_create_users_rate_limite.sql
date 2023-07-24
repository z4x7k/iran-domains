-- +goose Up
CREATE TABLE users_rate_limit (
	the_user_id BIGINT NOT NULL PRIMARY KEY,
	last_access_ts BIGINT NOT NULL,
	the_count BIGINT NOT NULL
);

-- +goose Down
DROP TABLE users_rate_limit;
