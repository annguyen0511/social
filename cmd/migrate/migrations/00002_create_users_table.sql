-- +goose Up
CREATE TABLE users (
    id bigserial PRIMARY KEY,
    first_name varchar(255) NOT NULL,
    last_name varchar(255) NOT NULL,
    avatar_url varchar(255),
    username varchar(255) NOT NULL UNIQUE,
    email citext NOT NULL UNIQUE,
    password bytea NOT NULL,
    created_at timestamp(0) with time zone DEFAULT now(),
    updated_at timestamp(0) with time zone DEFAULT now()
);

-- +goose Down
DROP TABLE IF EXISTS users CASCADE;
