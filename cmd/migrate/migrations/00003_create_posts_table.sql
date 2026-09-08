-- +goose Up
CREATE TABLE posts (
    id bigserial PRIMARY KEY,
    title varchar(255) NOT NULL,
    content text NOT NULL,
    user_id bigserial NOT NULL,
    tags text[],
    created_at timestamp(0) with time zone DEFAULT now(),
    updated_at timestamp(0) with time zone DEFAULT now(),
    CONSTRAINT fk_posts_user_id FOREIGN KEY (user_id) REFERENCES users(id)
);

-- +goose Down
DROP TABLE IF EXISTS posts CASCADE;
