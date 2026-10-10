-- +goose Up
-- Bảng riêng chứ không phải mấy cột trên posts, dù phiên bản này mỗi bài chỉ
-- một ảnh. Lý do: hình dạng thật của dữ liệu là 0..n, và UNIQUE(post_id) bên
-- dưới chính là thứ ép nó về 1 cho lúc này. Mở ra nhiều ảnh sau này là bỏ một
-- ràng buộc, còn nếu để cột trên posts thì phải di trú dữ liệu.
--
-- A separate table rather than columns on posts, even though this version
-- allows one image per post. The real shape of the data is 0..n, and the
-- UNIQUE(post_id) below is what pins it to 1 for now. Opening it up later is
-- dropping one constraint; columns on posts would mean migrating data.
CREATE TABLE IF NOT EXISTS post_images (
    id BIGSERIAL PRIMARY KEY,
    post_id BIGINT NOT NULL,
    file_name TEXT NOT NULL,
    -- Kích thước được lưu để trình duyệt chừa đúng chỗ trước khi ảnh tải
    -- xong. Thiếu nó, feed sẽ nhảy giật mỗi lần một tấm ảnh load.
    --
    -- The dimensions are stored so the browser can reserve the right space
    -- before the image arrives. Without them the feed jumps every time one
    -- finishes loading.
    width INTEGER NOT NULL,
    height INTEGER NOT NULL,
    position INTEGER NOT NULL DEFAULT 0,
    created_at timestamp(0) with time zone DEFAULT now(),
    CONSTRAINT post_images_one_per_post UNIQUE (post_id),
    FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE CASCADE
);

-- +goose Down
DROP TABLE IF EXISTS post_images;
