package store

import (
	"context"
	"database/sql"
	"errors"

	"github.com/annguyen0511/social/internal/model"
	"github.com/lib/pq"
)

type PostStore struct {
	db *sql.DB
}

func (s *PostStore) Create(ctx context.Context, post *model.Post) error {
	query := `
	INSERT INTO posts (content, title, user_id, tags)
	VALUES ($1, $2, $3, $4) RETURNING id, created_at, updated_at
	`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	err := s.db.QueryRowContext(
		ctx,
		query,
		post.Content,
		post.Title,
		post.UserID,
		pq.Array(post.Tags),
	).
		Scan(
			&post.ID,
			&post.CreatedAt,
			&post.UpdatedAt,
		)

	if err != nil {
		return err
	}

	return nil
}

func (s *PostStore) Update(ctx context.Context, post *model.Post) error {
	query := `
	UPDATE posts
	SET content = $1, title = $2, tags = $3, version = version + 1, updated_at = NOW()
	WHERE id = $4 AND version = $5 RETURNING version
	`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	err := s.db.QueryRowContext(
		ctx,
		query,
		post.Content,
		post.Title,
		pq.Array(post.Tags),
		post.ID,
		post.Version,
	).
		Scan(
			&post.Version,
		)

	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return ErrConflict
		default:
			return err
		}
	}

	return nil
}

func (s *PostStore) GetById(ctx context.Context, id int64) (*model.Post, error) {
	var post model.Post
	// The author is joined in because every caller shows the post to somebody,
	// and a post without a name and a face to go with it is not something any
	// screen can render. Leaving User zero-valued pointed the profile link at
	// /users/0.
	//
	// Tác giả được join vào vì mọi nơi gọi đều hiển thị bài cho ai đó xem, mà
	// một bài viết không có tên và khuôn mặt đi kèm thì không màn hình nào vẽ
	// ra được. Để User rỗng khiến link tới trang cá nhân trỏ về /users/0.
	query := `
	SELECT p.id, p.content, p.title, p.user_id, p.tags, p.created_at, p.updated_at, p.version,
	       u.id, u.username, u.first_name, u.last_name, COALESCE(u.avatar_url, '')
	FROM posts p
	JOIN users u ON u.id = p.user_id
	WHERE p.id = $1
	`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&post.ID, &post.Content, &post.Title, &post.UserID, pq.Array(&post.Tags),
		&post.CreatedAt, &post.UpdatedAt, &post.Version,
		&post.User.ID, &post.User.UserName, &post.User.FirstName, &post.User.LastName, &post.User.AvatarURL,
	)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrNotFound
		default:
			return nil, err
		}
	}
	return &post, nil
}

func (s *PostStore) Delete(ctx context.Context, id int64) error {
	query := `
	DELETE FROM posts
	WHERE id = $1
	`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	result, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	if rowsAffected, _ := result.RowsAffected(); rowsAffected == 0 {
		return ErrNotFound
	}
	return nil

}

// GetUserFeed returns the user's own posts plus posts by everyone they follow,
// newest first. Blocked users never appear: blocking removes the follow in both
// directions, so their posts drop out of the follow subquery on their own.
// GetByUser returns one author's posts, newest first.
//
// It shares the counting subqueries with GetUserFeed for the same reason that
// one uses them at all: joining comments and likes would multiply the rows and
// inflate both totals.
//
// GetByUser trả về bài của một tác giả, mới nhất trước.
//
// Nó dùng chung cách đếm bằng truy vấn con với GetUserFeed, vì cùng một lý do
// khiến bên kia phải dùng: join sang comments và likes sẽ nhân số dòng lên và
// làm phồng cả hai con số.
func (s *PostStore) GetByUser(ctx context.Context, authorID int64, page PaginationQuery) ([]model.FeedPost, int64, error) {
	countQuery := `SELECT COUNT(*) FROM posts p WHERE p.user_id = $1`
	pageQuery := `
	SELECT
		p.id, p.user_id, u.username, u.first_name, u.last_name, COALESCE(u.avatar_url, ''), p.title, p.content, p.tags,
		p.created_at, p.updated_at, p.version,
		(SELECT COUNT(*) FROM comments c WHERE c.post_id = p.id) AS comment_count,
		(SELECT COUNT(*) FROM likes l WHERE l.post_id = p.id) AS like_count
	FROM posts p
	JOIN users u ON u.id = p.user_id
	WHERE p.user_id = $1
	ORDER BY p.created_at DESC, p.id DESC
	LIMIT $2 OFFSET $3
	`

	return paginate(ctx, s.db, page, countQuery, pageQuery, []any{authorID}, func(rows *sql.Rows) (model.FeedPost, error) {
		var post model.FeedPost
		err := rows.Scan(
			&post.ID, &post.UserID, &post.User.UserName, &post.User.FirstName, &post.User.LastName, &post.User.AvatarURL, &post.Title, &post.Content, pq.Array(&post.Tags),
			&post.CreatedAt, &post.UpdatedAt, &post.Version,
			&post.CommentCount, &post.LikeCount,
		)
		post.User.ID = post.UserID
		return post, err
	})
}

func (s *PostStore) GetUserFeed(ctx context.Context, userID int64, page PaginationQuery) ([]model.FeedPost, int64, error) {
	// A subquery rather than JOIN follows: an inner join drops the user's own
	// posts when they follow nobody, and joins each own post to every follows
	// row otherwise, which multiplies COUNT(c.id).
	feedFilter := `
	p.user_id = $1
	OR p.user_id IN (SELECT following_id FROM follows WHERE follower_id = $1)
	`

	countQuery := `SELECT COUNT(*) FROM posts p WHERE ` + feedFilter
	pageQuery := `
	SELECT
		p.id, p.user_id, u.username, u.first_name, u.last_name, COALESCE(u.avatar_url, ''), p.title, p.content, p.tags,
		p.created_at, p.updated_at, p.version,
		(SELECT COUNT(*) FROM comments c WHERE c.post_id = p.id) AS comment_count,
		(SELECT COUNT(*) FROM likes l WHERE l.post_id = p.id) AS like_count
	FROM posts p
	JOIN users u ON u.id = p.user_id
	WHERE ` + feedFilter + `
	ORDER BY p.created_at DESC, p.id DESC
	LIMIT $2 OFFSET $3
	`

	return paginate(ctx, s.db, page, countQuery, pageQuery, []any{userID}, func(rows *sql.Rows) (model.FeedPost, error) {
		var post model.FeedPost
		err := rows.Scan(
			&post.ID, &post.UserID, &post.User.UserName, &post.User.FirstName, &post.User.LastName, &post.User.AvatarURL, &post.Title, &post.Content, pq.Array(&post.Tags),
			&post.CreatedAt, &post.UpdatedAt, &post.Version,
			&post.CommentCount, &post.LikeCount,
		)
		post.User.ID = post.UserID
		return post, err
	})
}
