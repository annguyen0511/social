package store

import (
	"context"
	"database/sql"

	"github.com/annguyen0511/social/internal/model"
)

type CloseFriendStore struct {
	db *sql.DB
}

func (s *CloseFriendStore) Add(ctx context.Context, userID int64, friendID int64) error {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	blocked, err := blockExistsTx(ctx, tx, userID, friendID)
	if err != nil {
		return err
	}
	if blocked {
		return ErrBlocked
	}

	query := `
	INSERT INTO close_friends (user_id, friend_id)
	VALUES ($1, $2)
	ON CONFLICT DO NOTHING
	`
	if _, err := tx.ExecContext(ctx, query, userID, friendID); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *CloseFriendStore) Remove(ctx context.Context, userID int64, friendID int64) error {
	query := `
	DELETE FROM close_friends
	WHERE user_id = $1 AND friend_id = $2
	`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	_, err := s.db.ExecContext(ctx, query, userID, friendID)
	if err != nil {
		return err
	}
	return nil
}

// List returns the people userID has put on their close friends list.
//
// It returns the users themselves rather than their ids, for the same reason
// the blocked list does: a bare id cannot be shown to anyone, and resolving
// each one separately would be a query per row.
//
// No block filter is needed. Blocking severs close friend entries in both
// directions inside the same transaction, so a blocked person cannot still be
// on this list.
//
// List trả về những người mà userID đã đưa vào danh sách bạn thân.
//
// Nó trả về chính các user chứ không phải id, vì cùng lý do với danh sách
// chặn: một id trần thì không hiển thị cho ai được, mà tra từng cái riêng lẻ
// sẽ thành một truy vấn cho mỗi dòng.
//
// Không cần lọc block. Việc chặn đã cắt các mục bạn thân ở cả hai chiều
// trong cùng một transaction, nên một người bị chặn không thể còn nằm trong
// danh sách này.
func (s *CloseFriendStore) List(ctx context.Context, userID int64, page PaginationQuery) ([]model.User, int64, error) {
	countQuery := `SELECT COUNT(*) FROM close_friends WHERE user_id = $1`
	pageQuery := `
	SELECT u.id, u.first_name, u.last_name, COALESCE(u.avatar_url, ''), u.username,
	       u.email, u.is_active, u.created_at, u.updated_at
	FROM close_friends cf
	JOIN users u ON u.id = cf.friend_id
	WHERE cf.user_id = $1
	ORDER BY cf.created_at DESC, u.id DESC
	LIMIT $2 OFFSET $3
	`

	return paginate(ctx, s.db, page, countQuery, pageQuery, []any{userID}, func(rows *sql.Rows) (model.User, error) {
		var user model.User
		err := rows.Scan(
			&user.ID, &user.FirstName, &user.LastName, &user.AvatarURL, &user.UserName,
			&user.Email, &user.IsActive, &user.CreatedAt, &user.UpdatedAt,
		)
		return user, err
	})
}

func (s *CloseFriendStore) IsCloseFriend(ctx context.Context, userID, friendID int64) (bool, error) {
	query := `
	SELECT EXISTS(
		SELECT 1 FROM close_friends
		WHERE user_id = $1 AND friend_id = $2
	)
	`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	var exists bool
	if err := s.db.QueryRowContext(ctx, query, userID, friendID).Scan(&exists); err != nil {
		return false, err
	}

	return exists, nil
}
