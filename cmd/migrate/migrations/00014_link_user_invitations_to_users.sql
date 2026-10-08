-- +goose Up
-- The index comes first: Postgres does not create one for a referencing
-- column, and without it every user delete has to scan user_invitations to
-- find the rows to cascade. It also serves the lookups and cleanups the store
-- already runs by user_id.
--
-- Tạo index trước: Postgres không tự tạo index cho cột tham chiếu, thiếu nó
-- thì mỗi lần xoá user đều phải quét cả bảng user_invitations để tìm dòng cần
-- xoá theo. Index này cũng phục vụ các truy vấn và lệnh dọn theo user_id mà
-- tầng store đang chạy.
CREATE INDEX IF NOT EXISTS idx_user_invitations_user_id ON user_invitations (user_id);

ALTER TABLE user_invitations
    ADD CONSTRAINT fk_user_invitations_user_id
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE;

-- +goose Down
ALTER TABLE user_invitations DROP CONSTRAINT fk_user_invitations_user_id;
DROP INDEX IF EXISTS idx_user_invitations_user_id;
