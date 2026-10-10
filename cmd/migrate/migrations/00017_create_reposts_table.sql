-- +goose Up
CREATE TABLE IF NOT EXISTS reposts (
    post_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    created_at timestamp(0) with time zone DEFAULT now(),
    PRIMARY KEY (post_id, user_id),
    -- Xoá bài gốc thì mọi lượt repost biến mất theo. Không có CASCADE thì
    -- danh sách repost sẽ trỏ tới những bài không còn tồn tại.
    --
    -- Deleting the original removes every repost of it. Without CASCADE the
    -- repost lists would point at posts that no longer exist.
    FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Truy vấn là "bài người này đã repost, mới nhất trước", nên index mang cả
-- cột sắp xếp. Chiều ngược lại — đếm repost của một bài — đã có khoá chính lo.
--
-- The query is "posts this person reposted, newest first", so the index
-- carries the sort column. The other direction, counting a post's reposts,
-- is already covered by the primary key.
CREATE INDEX IF NOT EXISTS idx_reposts_user_id ON reposts (user_id, created_at DESC);

-- +goose Down
DROP INDEX IF EXISTS idx_reposts_user_id;
DROP TABLE IF EXISTS reposts;
