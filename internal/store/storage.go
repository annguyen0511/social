package store

import (
	"context"
	"database/sql"
	"errors"

	"github.com/annguyen0511/social/internal/model"
)

var ErrNotFound = errors.New("resource not found")

type Storage struct {
	Post interface {
		Create(context.Context, *model.Post) error
		Update(context.Context, *model.Post) error
		GetById(context.Context, int64) (*model.Post, error)
		Delete(context.Context, int64) error
	}

	User interface {
		Create(context.Context, *model.User) error
		Update(context.Context, *model.User) error
		GetById(context.Context, int64) (*model.User, error)
	}

	Comment interface {
		Create(context.Context, *model.Comment) error
		GetByPostId(context.Context, int64) ([]model.Comment, error)
	}
}

func NewStorage(db *sql.DB) Storage {
	return Storage{
		Post:    &PostStore{db},
		User:    &UserStore{db},
		Comment: &CommentStore{db},
	}
}
