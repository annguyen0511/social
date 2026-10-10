package store

import (
	"context"
	"database/sql"
)

type SaveStore struct {
	db *sql.DB
}

// Save records that userID saved postID. The primary key is (post_id,
// user_id), so ON CONFLICT DO NOTHING makes a second press a no-op rather
// than an error: pressing twice should leave the same state as pressing once.
//
// Save ghi nhận userID đã lưu postID. Khoá chính là (post_id, user_id), nên
// ON CONFLICT DO NOTHING khiến lần bấm thứ hai không làm gì thay vì báo lỗi:
// bấm hai lần phải cho ra đúng trạng thái như bấm một lần.
func (s *SaveStore) Save(ctx context.Context, postID, userID int64) error {
	query := `
	INSERT INTO saved_posts (post_id, user_id)
	VALUES ($1, $2)
	ON CONFLICT DO NOTHING
	`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	_, err := s.db.ExecContext(ctx, query, postID, userID)
	return err
}

// Unsave removes the save, and removing one that is not there is not an error.
// Unsave gỡ mục đã lưu, và gỡ một mục vốn không tồn tại thì không phải lỗi.
func (s *SaveStore) Unsave(ctx context.Context, postID, userID int64) error {
	query := `DELETE FROM saved_posts WHERE post_id = $1 AND user_id = $2`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	_, err := s.db.ExecContext(ctx, query, postID, userID)
	return err
}

// IsSaved reports whether viewerID saved the post.
//
// There is no count to go with it, and that is the point: unlike a like, a
// save is private. Publishing how many people saved a post would tell its
// author about behaviour those people never chose to share.
//
// IsSaved cho biết viewerID đã lưu bài này chưa.
//
// Không có số đếm đi kèm, và đó là chủ đích: khác với lượt thích, việc lưu
// bài là riêng tư. Công bố số người đã lưu một bài là kể cho tác giả nghe một
// hành vi mà những người đó chưa bao giờ chọn chia sẻ.
func (s *SaveStore) IsSaved(ctx context.Context, postID, viewerID int64) (bool, error) {
	query := `SELECT EXISTS (SELECT 1 FROM saved_posts WHERE post_id = $1 AND user_id = $2)`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	var saved bool
	err := s.db.QueryRowContext(ctx, query, postID, viewerID).Scan(&saved)
	return saved, err
}
