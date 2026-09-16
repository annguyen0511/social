-- +goose Up
CREATE TABLE IF NOT EXISTS follows (
	follower_id BIGINT NOT NULL,
	following_id BIGINT NOT NULL,
	created_at timestamp(0) with time zone DEFAULT now(),
	PRIMARY KEY (follower_id, following_id), -- Composite Primary Key ensure that one user can only follow another user once
	FOREIGN KEY (follower_id) REFERENCES users(id) ON DELETE CASCADE,
	FOREIGN KEY (following_id) REFERENCES users(id) ON DELETE CASCADE
);

-- +goose Down
DROP TABLE IF EXISTS follows;
