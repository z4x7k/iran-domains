-- +goose Up
CREATE TABLE users_rate_limit (
	peer CHARACTER(44) NOT NULL,
	upload UNSIGNED BIG INT NOT NULL,
	download UNSIGNED BIG INT NOT NULL,
	ts UNSIGNED BIG INT NOT NULL
);
CREATE INDEX peers_usage_peer_IDX ON peers_usage (peer ASC);
CREATE INDEX peers_usage_ts_IDX ON peers_usage (ts DESC);

-- +goose Down
DROP TABLE users_rate_limit;
