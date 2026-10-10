-- +goose Up
-- A table of its own rather than reusing user_invitations. The two tokens
-- unlock different things, and one table would mean an activation token
-- could be posted to the reset endpoint and silently accepted. Separate
-- tables make that impossible rather than merely unlikely.
--
-- Bảng riêng chứ không dùng lại user_invitations. Hai loại token mở hai thứ
-- khác nhau, mà gộp một bảng thì một token kích hoạt có thể gửi vào endpoint
-- đặt lại mật khẩu và được chấp nhận lặng lẽ. Tách bảng khiến chuyện đó là
-- không thể, chứ không chỉ là khó xảy ra.
CREATE TABLE IF NOT EXISTS password_resets (
	token bytea PRIMARY KEY,
	user_id bigint NOT NULL,
	-- created_at is what the cooldown is measured from: without it, asking
	-- for a link over and over would mail someone's inbox into the ground.
	--
	-- created_at là mốc để tính thời gian chờ giữa hai lần xin link: thiếu
	-- nó thì bấm "quên mật khẩu" liên tục sẽ dội bom hòm thư của người khác.
	created_at timestamp(0) with time zone NOT NULL DEFAULT NOW(),
	expired_at timestamp(0) with time zone NOT NULL,
	CONSTRAINT fk_password_resets_user_id
		FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);

-- Postgres creates no index for a referencing column, so without this every
-- user delete scans the table to find rows to cascade. It also serves the
-- lookup by user_id that the cooldown check and the cleanup both run.
--
-- Postgres không tự tạo index cho cột tham chiếu, nên thiếu nó thì mỗi lần
-- xoá user đều phải quét cả bảng để tìm dòng xoá theo. Index này cũng phục vụ
-- việc tra theo user_id mà cả phép kiểm thời gian chờ lẫn lệnh dọn đều chạy.
CREATE INDEX IF NOT EXISTS idx_password_resets_user_id ON password_resets (user_id);

-- token_version is what lets a password change end sessions that are already
-- running. A JWT is stateless: once signed it stays good until it expires, so
-- changing the password would otherwise leave a stolen cookie working for
-- days — and a stolen cookie is the main reason people change a password at
-- all. Every token carries the version it was signed under; bumping this
-- number makes every older one stop matching.
--
-- NOT NULL DEFAULT 0 means existing accounts are already consistent with the
-- tokens their browsers are holding, so nobody is logged out by this
-- migration.
--
-- token_version là thứ cho phép việc đổi mật khẩu kết thúc được những phiên
-- đang chạy. JWT là stateless: ký xong thì có giá trị tới lúc hết hạn, nên
-- nếu không có cột này thì đổi mật khẩu xong một cookie bị trộm vẫn dùng được
-- thêm nhiều ngày — mà cookie bị trộm lại chính là lý do chủ yếu khiến người
-- ta đổi mật khẩu. Mỗi token mang theo số phiên bản lúc nó được ký; tăng số
-- này lên là mọi token cũ thôi khớp.
--
-- NOT NULL DEFAULT 0 nghĩa là các tài khoản sẵn có đã khớp ngay với token mà
-- trình duyệt của họ đang giữ, nên migration này không đá ai ra ngoài.
ALTER TABLE users ADD COLUMN IF NOT EXISTS token_version integer NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE users DROP COLUMN IF EXISTS token_version;
DROP TABLE IF EXISTS password_resets;
