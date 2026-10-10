package store

import (
	"context"
	"database/sql"
	"errors"
)

type RepostStore struct {
	db *sql.DB
}

// Repost records that userID reposted postID.
//
// It runs in a transaction because the block check and the insert have to see
// the same world: without one, somebody could be blocked between the check
// and the write and the repost would still land.
//
// The author is read inside that transaction rather than passed in, so the
// block is always checked against whoever actually owns the post — a caller
// cannot hand in the wrong id, by mistake or otherwise.
//
// Repost ghi nhận userID đã repost postID.
//
// Nó chạy trong một transaction vì phép kiểm block và lệnh ghi phải nhìn thấy
// cùng một thế giới: thiếu nó, một người có thể bị chặn ở khoảng giữa hai
// bước mà lượt repost vẫn lọt xuống.
//
// Tác giả được đọc ngay trong transaction đó chứ không nhận từ tham số, để
// phép kiểm block luôn đối chiếu với đúng người sở hữu bài viết — phía gọi
// không thể đưa nhầm id, dù vô tình hay không.
func (s *RepostStore) Repost(ctx context.Context, postID, userID int64) error {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	return withTx(ctx, s.db, func(tx *sql.Tx) error {
		var authorID int64
		err := tx.QueryRowContext(ctx, `SELECT user_id FROM posts WHERE id = $1`, postID).Scan(&authorID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrNotFound
			}
			return err
		}

		blocked, err := blockExistsTx(ctx, tx, userID, authorID)
		if err != nil {
			return err
		}
		if blocked {
			return ErrBlocked
		}

		// The primary key is (post_id, user_id), so a second press is a no-op
		// rather than an error.
		//
		// Khoá chính là (post_id, user_id), nên lần bấm thứ hai không làm gì
		// thay vì báo lỗi.
		query := `
		INSERT INTO reposts (post_id, user_id)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
		`
		_, err = tx.ExecContext(ctx, query, postID, userID)
		return err
	})
}

// Unrepost removes the repost, and removing one that is not there is not an
// error. No block check: taking something back is always allowed, and
// refusing would leave a repost nobody can undo.
//
// Unrepost gỡ lượt repost, và gỡ một lượt vốn không tồn tại thì không phải
// lỗi. Không kiểm block: rút lại thứ mình đã làm thì luôn được phép, từ chối
// sẽ để lại một lượt repost không ai gỡ nổi.
func (s *RepostStore) Unrepost(ctx context.Context, postID, userID int64) error {
	query := `DELETE FROM reposts WHERE post_id = $1 AND user_id = $2`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	_, err := s.db.ExecContext(ctx, query, postID, userID)
	return err
}

// Stats returns how many people reposted the post and whether viewerID is one
// of them, in one round trip. Two queries would be two chances for the
// numbers to disagree with each other.
//
// Stats trả về số người đã repost bài và việc viewerID có nằm trong số đó hay
// không, trong một lượt gọi. Hai truy vấn riêng là hai cơ hội để hai con số
// mâu thuẫn với nhau.
func (s *RepostStore) Stats(ctx context.Context, postID, viewerID int64) (int64, bool, error) {
	query := `
	SELECT
		COUNT(*),
		COUNT(*) FILTER (WHERE user_id = $2) > 0
	FROM reposts
	WHERE post_id = $1
	`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	var count int64
	var reposted bool
	err := s.db.QueryRowContext(ctx, query, postID, viewerID).Scan(&count, &reposted)
	return count, reposted, err
}
