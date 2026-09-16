-- +goose Up
CREATE TABLE IF NOT EXISTS close_friends (
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    friend_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at timestamp(0) with time zone DEFAULT now(),
    PRIMARY KEY (user_id, friend_id)
);

-- +goose Down
DROP TABLE IF EXISTS close_friends;
