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
	query := `
	SELECT c.id, c.post_id, c.user_id, c.content, c.created_at, c.updated_at, u.id, u.username, u.email
	FROM comments c
	JOIN users u ON c.user_id = u.id
	WHERE c.post_id = $1
	ORDER BY c.created_at DESC
	`
	rows, err := s.db.QueryContext(ctx, query, postID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []model.Comment
	for rows.Next() {
		var c model.Comment
		if err := rows.Scan(&c.ID, &c.PostID, &c.UserID, &c.Content, &c.CreatedAt, &c.UpdatedAt, &c.User.ID, &c.User.UserName, &c.User.Email); err != nil {
			return nil, err
		}
		comments = append(comments, c)
	}

	return comments, nil
}
