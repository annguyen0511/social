-- +goose Up
CREATE TABLE IF NOT EXISTS saved_posts (
    post_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    created_at timestamp(0) with time zone DEFAULT now(),
    PRIMARY KEY (post_id, user_id),
    -- Xoá bài thì mục đã lưu biến mất theo. Thiếu CASCADE, danh sách "đã lưu"
    -- sẽ đầy những bài không còn tồn tại.
    --
    -- Deleting a post removes the saves of it. Without CASCADE the saved list
    -- fills up with posts that no longer exist.
    FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Truy vấn duy nhất trên bảng này là "bài tôi đã lưu, mới nhất trước", nên
-- index phải mang cả cột sắp xếp chứ không chỉ cột lọc.
--
-- The only query against this table is "posts I saved, newest first", so the
-- index carries the sort column too, not just the filter column.
CREATE INDEX IF NOT EXISTS idx_saved_posts_user_id ON saved_posts (user_id, created_at DESC);

-- +goose Down
DROP INDEX IF EXISTS idx_saved_posts_user_id;
DROP TABLE IF EXISTS saved_posts;
