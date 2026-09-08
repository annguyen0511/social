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
		GetById(context.Context, int64) (*model.Post, error)
	}

	User interface {
		Create(context.Context, *model.User) error
	}
}

func NewStorage(db *sql.DB) Storage {
	return Storage{
		Post: &PostStore{db},
		User: &UserStore{db},
	}
}
