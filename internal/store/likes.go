package store

import (
	"context"
	"database/sql"
)

type LikeStore struct {
	db *sql.DB
}

// Like records that userID likes postID. The primary key is (post_id,
// user_id), so ON CONFLICT DO NOTHING makes a second press a no-op rather
// than an error: pressing twice should leave the same state as pressing once.
//
// Like ghi nhận userID thích postID. Khoá chính là (post_id, user_id), nên ON
// CONFLICT DO NOTHING khiến lần bấm thứ hai không làm gì thay vì báo lỗi: bấm
// hai lần phải cho ra đúng trạng thái như bấm một lần.
func (s *LikeStore) Like(ctx context.Context, postID, userID int64) error {
	query := `
	INSERT INTO likes (post_id, user_id)
	VALUES ($1, $2)
	ON CONFLICT DO NOTHING
	`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	_, err := s.db.ExecContext(ctx, query, postID, userID)
	return err
}

// Unlike removes the like, and removing one that is not there is not an error.
// Unlike gỡ lượt thích, và gỡ một lượt vốn không tồn tại thì không phải lỗi.
func (s *LikeStore) Unlike(ctx context.Context, postID, userID int64) error {
	query := `DELETE FROM likes WHERE post_id = $1 AND user_id = $2`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	_, err := s.db.ExecContext(ctx, query, postID, userID)
	return err
}

// Stats returns how many people liked the post and whether viewerID is one of
// them, in one round trip. Two queries would be two chances for the numbers to
// disagree with each other.
//
// Stats trả về số người đã thích bài và việc viewerID có nằm trong số đó hay
// không, trong một lượt gọi. Hai truy vấn riêng là hai cơ hội để hai con số
// mâu thuẫn với nhau.
func (s *LikeStore) Stats(ctx context.Context, postID, viewerID int64) (int64, bool, error) {
	query := `
	SELECT
		COUNT(*),
		COUNT(*) FILTER (WHERE user_id = $2) > 0
	FROM likes
	WHERE post_id = $1
	`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	var count int64
	var liked bool
	err := s.db.QueryRowContext(ctx, query, postID, viewerID).Scan(&count, &liked)
	return count, liked, err
}
