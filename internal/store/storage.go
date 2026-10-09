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

	ErrDuplicateEmail    = errors.New("a user with that email already exists")
	ErrDuplicateUsername = errors.New("a user with that username already exists")

	QueryTimeoutDuration = time.Second * 5
)

type Storage struct {
	Post interface {
		Create(context.Context, *model.Post) error
		Update(context.Context, *model.Post) error
		GetById(context.Context, int64) (*model.Post, error)
		Delete(context.Context, int64) error
		GetUserFeed(context.Context, int64, PaginationQuery) ([]model.FeedPost, int64, error)
		GetByUser(ctx context.Context, authorID, viewerID int64, page PaginationQuery) ([]model.FeedPost, int64, error)
	}

	User interface {
		Create(context.Context, *model.User) error
		CreateAndInvited(context.Context, *model.User, string, time.Duration) error
		Activate(context.Context, string) error
		Delete(context.Context, int64) error
		Update(context.Context, *model.User) error
		GetById(context.Context, int64) (*model.User, error)
		GetByEmail(context.Context, string) (*model.User, error)
		Search(ctx context.Context, viewerID int64, q string, page PaginationQuery) ([]model.SearchedUser, int64, error)
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
		Counts(ctx context.Context, userID int64) (followers int64, following int64, err error)
	}
	Block interface {
		Block(ctx context.Context, blockerId int64, blockedId int64) error
		Unblock(ctx context.Context, blockerId int64, blockedId int64) error
		ListBlocking(ctx context.Context, blockerId int64, page PaginationQuery) ([]model.Block, int64, error)
		IsBlocking(ctx context.Context, blockerId int64, blockedId int64) (bool, error)
	}
	Like interface {
		Like(ctx context.Context, postID, userID int64) error
		Unlike(ctx context.Context, postID, userID int64) error
		Stats(ctx context.Context, postID, viewerID int64) (int64, bool, error)
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
		Like:        &LikeStore{db},
		CloseFriend: &CloseFriendStore{db},
	}
}

// withTx runs fn inside a transaction: fn returning nil commits, any error
// rolls back. The deferred Rollback is a no-op after a successful Commit, and
// it is what releases the connection if fn panics — middleware.Recoverer would
// otherwise keep the server alive with a connection stuck "idle in
// transaction", still holding its locks.
//
// withTx chạy fn bên trong một transaction: fn trả về nil thì commit, trả về
// lỗi thì rollback. Lệnh Rollback trong defer không làm gì sau khi Commit
// thành công, và nó chính là thứ trả connection về pool khi fn panic — nếu
// không, middleware.Recoverer sẽ giữ server sống với một connection kẹt ở
// trạng thái "idle in transaction" và vẫn đang giữ khoá.
func withTx(ctx context.Context, db *sql.DB, fn func(*sql.Tx) error) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := fn(tx); err != nil {
		return err
	}

	return tx.Commit()
}
