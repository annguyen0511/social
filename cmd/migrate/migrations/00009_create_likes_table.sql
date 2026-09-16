-- +goose Up
CREATE TABLE IF NOT EXISTS likes (
    post_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    created_at timestamp(0) with time zone DEFAULT now(),
    PRIMARY KEY (post_id, user_id),
    FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- +goose Down
DROP TABLE IF EXISTS likes;
