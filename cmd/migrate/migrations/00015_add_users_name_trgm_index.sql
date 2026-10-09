-- +goose Up
-- Trigram index on the full name, mirroring the one on username from 00011.
-- The expression must match the search query character for character, or
-- Postgres cannot use the index and falls back to scanning every row.
--
-- Index trigram cho họ tên đầy đủ, song song với index username ở 00011.
-- Biểu thức phải trùng từng ký tự với biểu thức trong câu truy vấn, nếu lệch
-- thì Postgres không dùng được index và phải quét toàn bảng.
CREATE INDEX IF NOT EXISTS idx_users_fullname_trgm
    ON users USING GIN ((first_name || ' ' || last_name) gin_trgm_ops);

-- +goose Down
DROP INDEX IF EXISTS idx_users_fullname_trgm;
