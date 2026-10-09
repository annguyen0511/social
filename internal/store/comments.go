package store

import (
	"context"
	"database/sql"

	"github.com/annguyen0511/social/internal/model"
)

type CommentStore struct {
	db *sql.DB
}

func (s *CommentStore) Create(ctx context.Context, comment *model.Comment) error {
	query := `
	INSERT INTO comments (post_id, user_id, content)
	VALUES ($1, $2, $3)
	RETURNING id, created_at, updated_at
	`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	err := s.db.QueryRowContext(
		ctx,
		query,
		comment.PostID,
		comment.UserID,
		comment.Content,
	).
		Scan(
			&comment.ID,
			&comment.CreatedAt,
			&comment.UpdatedAt,
		)

	if err != nil {
		return err
	}

	return nil
}

func (s *CommentStore) GetByPostId(ctx context.Context, postID int64) ([]model.Comment, error) {
	// first_name, last_name and avatar_url come along because a comment is
	// shown with its author's picture, which falls back to their initials.
	// Email is deliberately not selected: nothing on screen uses it, and
	// sending it would hand every reader the address of everyone who ever
	// commented.
	//
	// Thứ tự có thêm c.id vì created_at không phải khoá duy nhất: hai bình
	// luận trùng giây sẽ ra thứ tự khác nhau giữa các lần gọi.
	//
	// first_name, last_name và avatar_url được lấy kèm vì bình luận hiển thị
	// cùng ảnh của tác giả, mà ảnh đó lùi về chữ cái đầu của tên khi không
	// có. Email cố tình không lấy: không chỗ nào trên màn hình dùng tới, mà
	// gửi đi thì mọi người đọc đều có địa chỉ của tất cả những ai từng bình
	// luận.
	query := `
	SELECT c.id, c.post_id, c.user_id, c.content, c.created_at, c.updated_at,
	       u.id, u.username, u.first_name, u.last_name, COALESCE(u.avatar_url, '')
	FROM comments c
	JOIN users u ON c.user_id = u.id
	WHERE c.post_id = $1
	ORDER BY c.created_at DESC, c.id DESC
	`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	rows, err := s.db.QueryContext(ctx, query, postID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []model.Comment
	for rows.Next() {
		var c model.Comment
		if err := rows.Scan(
			&c.ID, &c.PostID, &c.UserID, &c.Content, &c.CreatedAt, &c.UpdatedAt,
			&c.User.ID, &c.User.UserName, &c.User.FirstName, &c.User.LastName, &c.User.AvatarURL,
		); err != nil {
			return nil, err
		}
		comments = append(comments, c)
	}

	return comments, nil
}
