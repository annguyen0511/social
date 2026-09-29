-- +goose Up
CREATE TABLE IF NOT EXISTS user_invitations (
	token bytea NOT NULL,
	user_id bigint NOT NULL,
	expired_at timestamp(0) with time zone NOT NULL,
	PRIMARY KEY (token, user_id)
);

-- +goose Down
DROP TABLE user_invitations;
