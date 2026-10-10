-- +goose Up
-- TEXT kèm CHECK chứ không dùng ENUM của Postgres: thêm một giá trị mới vào
-- ENUM là một thao tác phải cẩn thận và không hoàn tác dễ, còn ở đây chỉ là
-- sửa ràng buộc. Cũng không dùng BOOLEAN is_private, vì "chế độ hiển thị"
-- nhiều khả năng sẽ có mức thứ ba, mà một boolean thì không mở rộng được.
--
-- TEXT with a CHECK rather than a Postgres ENUM: adding a value to an ENUM is
-- a careful, not-easily-undone operation, while this is just editing a
-- constraint. Not a BOOLEAN is_private either, because visibility will likely
-- grow a third level and a boolean cannot.
--
-- DEFAULT 'public' là thứ khiến mọi bài đang có giữ nguyên cách hoạt động cũ.
-- DEFAULT 'public' is what leaves every existing post behaving as before.
ALTER TABLE posts
    ADD COLUMN IF NOT EXISTS visibility TEXT NOT NULL DEFAULT 'public'
    CHECK (visibility IN ('public', 'private'));

-- +goose Down
ALTER TABLE posts DROP COLUMN IF EXISTS visibility;
