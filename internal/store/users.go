package store

import (
	"context"
	"database/sql"
	"errors"

	"github.com/annguyen0511/social/internal/model"
)

type UserStore struct {
	db *sql.DB
}

func (u *UserStore) Create(ctx context.Context, user *model.User) error {
	query := `
	INSERT INTO users (first_name, last_name, avatar_url, username, email, password)
	VALUES ($1, $2, $3, $4, $5, $6)
	RETURNING id, created_at, updated_at
	`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	err := u.db.QueryRowContext(
		ctx,
		query,
		user.FirstName,
		user.LastName,
		user.AvatarURL,
		user.UserName,
		user.Email,
		user.Password,
	).
		Scan(
			&user.ID,
			&user.CreatedAt,
			&user.UpdatedAt,
		)

	if err != nil {
		return err
	}
	return nil
}

func (u *UserStore) Update(ctx context.Context, user *model.User) error {
	query := `
	UPDATE users
	SET first_name = $1, last_name = $2, avatar_url = $3, updated_at = NOW()
	WHERE id = $4 RETURNING updated_at
	`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	err := u.db.QueryRowContext(ctx, query, user.FirstName, user.LastName, user.AvatarURL, user.ID).Scan(&user.UpdatedAt)
	if err != nil {
		return err
	}
	return nil
}

func (u *UserStore) GetById(ctx context.Context, id int64) (*model.User, error) {
	var user model.User
	query := `
	SELECT id, first_name, last_name, COALESCE(avatar_url, ''), username, email, password, created_at, updated_at
	FROM users
	WHERE id = $1
	`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	err := u.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID,
		&user.FirstName,
		&user.LastName,
		&user.AvatarURL,
		&user.UserName,
		&user.Email,
		&user.Password,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &user, nil
}
