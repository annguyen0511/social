package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/annguyen0511/social/internal/model"
)

var (
	ErrNotFound = errors.New("resource not found")
	ErrConflict = errors.New("resource conflict")
	ErrBlocked  = errors.New("a block exists between the two users")

	QueryTimeoutDuration = time.Second * 5
)

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
	Follow interface {
		Follow(context.Context, int64, int64) error
		Unfollow(context.Context, int64, int64) error
		GetFollowers(context.Context, int64, PaginationQuery) ([]model.Follow, int64, error)
		GetFollowing(context.Context, int64, PaginationQuery) ([]model.Follow, int64, error)
		IsFollowing(context.Context, int64, int64) (bool, error)
	}
	Block interface {
		Block(ctx context.Context, blockerId int64, blockedId int64) error
		Unblock(ctx context.Context, blockerId int64, blockedId int64) error
		ListBlocking(ctx context.Context, blockerId int64, page PaginationQuery) ([]model.Block, int64, error)
		IsBlocking(ctx context.Context, blockerId int64, blockedId int64) (bool, error)
	}
	CloseFriend interface {
		Add(ctx context.Context, userID int64, friendID int64) error
		Remove(ctx context.Context, userID int64, friendID int64) error
		List(ctx context.Context, userID int64, page PaginationQuery) ([]int64, int64, error)
		IsCloseFriend(ctx context.Context, userID, friendID int64) (bool, error)
	}
}

func NewStorage(db *sql.DB) Storage {
	return Storage{
		Post:        &PostStore{db},
		User:        &UserStore{db},
		Comment:     &CommentStore{db},
		Follow:      &FollowStore{db},
		Block:       &BlockStore{db},
		CloseFriend: &CloseFriendStore{db},
	}
}
