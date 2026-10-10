-- +goose Up
-- Bỏ ràng buộc từng ép mỗi bài chỉ một ảnh. Đúng như ghi chú trong 00019:
-- mở ra nhiều ảnh chỉ là bỏ một ràng buộc, không phải di trú dữ liệu.
--
-- Drops the constraint that pinned each post to one image. Exactly as the
-- note in 00019 said: opening it up is dropping one constraint, not
-- migrating data.
ALTER TABLE post_images DROP CONSTRAINT IF EXISTS post_images_one_per_post;

-- Truy vấn luôn gom ảnh của một bài theo đúng thứ tự người dùng sắp, nên
-- index mang cả cột sắp xếp.
--
-- Queries always gather a post's images in the order the user arranged them,
-- so the index carries the sort column too.
CREATE INDEX IF NOT EXISTS idx_post_images_post_id ON post_images (post_id, position);

-- Một người không thể gửi cùng một file hai lần trong một bài, và tên file do
-- server sinh ngẫu nhiên nên trùng là dấu hiệu có lỗi chứ không phải ý muốn.
--
-- The same file cannot appear twice in one post, and since the server invents
-- the names at random a duplicate means a bug rather than an intention.
CREATE UNIQUE INDEX IF NOT EXISTS idx_post_images_file_name ON post_images (file_name);

-- +goose Down
DROP INDEX IF EXISTS idx_post_images_file_name;
DROP INDEX IF EXISTS idx_post_images_post_id;
ALTER TABLE post_images ADD CONSTRAINT post_images_one_per_post UNIQUE (post_id);
