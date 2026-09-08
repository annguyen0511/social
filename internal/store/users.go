package store

import (
	"context"
	"database/sql"

	"github.com/annguyen0511/social/internal/model"
)

type UserStore struct {
	db *sql.DB
}

func (u *UserStore) Create(ctx context.Context, user *model.User) error {
	query := `
	INSERT INTO users (username, email, password) 
	VALUES ($1, $2, $3)
	RETURNING id, created_at
	`

	err := u.db.QueryRowContext(
		ctx,
		query,
		user.UserName,
		user.Email,
		user.Password,
	).
		Scan(
			&user.ID,
			&user.CreatedAt,
		)

	if err != nil {
		return err
	}
	return nil
}
