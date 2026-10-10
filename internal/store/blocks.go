package store

import (
	"context"
	"database/sql"

	"github.com/annguyen0511/social/internal/model"
)

type BlockStore struct {
	db *sql.DB
}

func (s *BlockStore) Block(ctx context.Context, blockerId int64, blockedId int64) error {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// unfollow if exist before block
	dropFollows := `
	DELETE FROM follows
	WHERE (follower_id = $1 AND following_id = $2)
	   OR (follower_id = $2 AND following_id = $1)
	`
	if _, err := tx.ExecContext(ctx, dropFollows, blockerId, blockedId); err != nil {
		return err
	}

	// remove from close friends if exist
	dropCloseFriends := `
	DELETE FROM close_friends
	WHERE (user_id = $1 AND friend_id = $2)
	   OR (user_id = $2 AND friend_id = $1)
	`
	if _, err := tx.ExecContext(ctx, dropCloseFriends, blockerId, blockedId); err != nil {
		return err
	}

	insertBlock := `
	INSERT INTO blocks (blocker_id, blocked_id)
	VALUES ($1, $2)
	ON CONFLICT DO NOTHING
	`
	if _, err := tx.ExecContext(ctx, insertBlock, blockerId, blockedId); err != nil {
		return err
	}

	return tx.Commit()
}

// check block exists in transaction
// blockExistsSQL asks whether a block stands between two people, in either
// direction. One constant so the transactional and plain versions below can
// never answer differently.
//
// A block is one row in one direction, but it means both: the blocked person
// loses access to the blocker's side as much as the other way round. Checking
// only `blocker_id = me` would let someone who blocked you keep reading
// everything of yours.
//
// blockExistsSQL hỏi xem giữa hai người có lệnh chặn nào không, ở bất kỳ
// chiều nào. Một hằng số duy nhất để bản chạy trong transaction và bản chạy
// thường bên dưới không bao giờ trả lời khác nhau.
//
// Một lệnh chặn là một dòng theo một chiều, nhưng ý nghĩa của nó là hai
// chiều: người bị chặn mất quyền truy cập phía người chặn, và ngược lại cũng
// vậy. Chỉ kiểm `blocker_id = tôi` sẽ để người đã chặn bạn vẫn đọc được mọi
// thứ của bạn.
const blockExistsSQL = `
	SELECT EXISTS(
		SELECT 1 FROM blocks
		WHERE (blocker_id = $1 AND blocked_id = $2)
		   OR (blocker_id = $2 AND blocked_id = $1)
	)
	`

func blockExistsTx(ctx context.Context, tx *sql.Tx, userId int64, otherId int64) (bool, error) {
	var exists bool
	err := tx.QueryRowContext(ctx, blockExistsSQL, userId, otherId).Scan(&exists)
	return exists, err
}

// Exists is blockExistsTx outside a transaction, for the read paths that only
// need to know whether to show something at all.
//
// Exists là blockExistsTx nhưng ngoài transaction, dành cho các đường đọc chỉ
// cần biết có nên hiển thị thứ gì đó hay không.
func (s *BlockStore) Exists(ctx context.Context, userID, otherID int64) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	var exists bool
	err := s.db.QueryRowContext(ctx, blockExistsSQL, userID, otherID).Scan(&exists)
	return exists, err
}

func (s *BlockStore) Unblock(ctx context.Context, blockerId int64, blockedId int64) error {
	query := `
	DELETE FROM blocks
	WHERE blocker_id = $1 AND blocked_id = $2
	`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	_, err := s.db.ExecContext(ctx, query, blockerId, blockedId)
	if err != nil {
		return err
	}
	return nil
}

// ListBlocking returns the people blockerId has blocked.
//
// It returns the users themselves rather than the rows of the blocks table.
// A pair of ids cannot be shown to anyone: a list of blocked people has to
// say who they are, and resolving each id separately would be one query per
// row.
//
// is_active is not filtered on, unlike everywhere else. Someone whose account
// was never activated can still have been blocked, and hiding them would
// leave a block the owner can neither see nor lift.
//
// ListBlocking trả về những người mà blockerId đã chặn.
//
// Nó trả về chính các user chứ không phải các dòng của bảng blocks. Một cặp
// id thì không hiển thị cho ai được: danh sách người bị chặn phải nói rõ họ
// là ai, mà tra từng id riêng lẻ sẽ thành một truy vấn cho mỗi dòng.
//
// Ở đây không lọc theo is_active, khác với mọi nơi khác. Một tài khoản chưa
// từng kích hoạt vẫn có thể đã bị chặn, và giấu họ đi sẽ để lại một lệnh chặn
// mà chính chủ không nhìn thấy cũng không gỡ được.
func (s *BlockStore) ListBlocking(ctx context.Context, blockerId int64, page PaginationQuery) ([]model.User, int64, error) {
	countQuery := `SELECT COUNT(*) FROM blocks WHERE blocker_id = $1`
	pageQuery := `
	SELECT u.id, u.first_name, u.last_name, COALESCE(u.avatar_url, ''), u.username,
	       u.email, u.is_active, u.created_at, u.updated_at
	FROM blocks b
	JOIN users u ON u.id = b.blocked_id
	WHERE b.blocker_id = $1
	ORDER BY b.created_at DESC, u.id DESC
	LIMIT $2 OFFSET $3
	`

	return paginate(ctx, s.db, page, countQuery, pageQuery, []any{blockerId}, func(rows *sql.Rows) (model.User, error) {
		var user model.User
		err := rows.Scan(
			&user.ID, &user.FirstName, &user.LastName, &user.AvatarURL, &user.UserName,
			&user.Email, &user.IsActive, &user.CreatedAt, &user.UpdatedAt,
		)
		return user, err
	})
}

func (s *BlockStore) IsBlocking(ctx context.Context, blockerId int64, blockedId int64) (bool, error) {
	query := `
	SELECT EXISTS(
		SELECT 1 FROM blocks
		WHERE blocker_id = $1 AND blocked_id = $2
	)
	`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	var exists bool
	if err := s.db.QueryRowContext(ctx, query, blockerId, blockedId).Scan(&exists); err != nil {
		return false, err
	}

	return exists, nil
}
