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
// CountByUser returns how many posts an author has written.
// CountByUser trả về số bài một tác giả đã viết.
func (s *PostStore) CountByUser(ctx context.Context, authorID int64) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	var total int64
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM posts WHERE user_id = $1`, authorID).Scan(&total)
	return total, err
}

func (s *PostStore) GetByUser(ctx context.Context, authorID, viewerID int64, page PaginationQuery) ([]model.FeedPost, int64, error) {
	countQuery := `SELECT COUNT(*) FROM posts p WHERE p.user_id = $1`
	pageQuery := `
	SELECT ` + feedPostColumns("$2") + `
	FROM posts p
	JOIN users u ON u.id = p.user_id
	WHERE p.user_id = $1
	ORDER BY p.created_at DESC, p.id DESC
	LIMIT $3 OFFSET $4
	`

	return paginateWith(ctx, s.db, page, countQuery, []any{authorID}, pageQuery, []any{authorID, viewerID}, scanFeedPost)
}

// feedPostColumns is the SELECT list every list of posts returns, with viewer
// standing for the placeholder holding the id of whoever is reading.
//
// It is one string rather than three copies because the three lists have
// already drifted apart twice: once when the author's avatar was added and
// once when their name was, each time leaving the other queries returning
// blanks that only showed up on screen. A shared list cannot drift, and
// scanFeedPost below is its matching half — change one and the compiler or
// the row scan will point at the other.
//
// feedPostColumns là danh sách cột SELECT mà mọi danh sách bài viết trả về,
// với viewer là chỗ dành cho placeholder chứa id của người đang đọc.
//
// Nó là một chuỗi duy nhất thay vì ba bản sao, vì ba danh sách đó đã lệch
// nhau hai lần rồi: một lần khi thêm avatar tác giả và một lần khi thêm họ
// tên, mỗi lần đều để các truy vấn còn lại trả về giá trị rỗng mà chỉ lòi ra
// khi nhìn màn hình. Một danh sách dùng chung thì không thể lệch, và
// scanFeedPost bên dưới là nửa còn lại của nó — sửa bên này thì trình biên
// dịch hoặc lúc quét dòng sẽ chỉ ngay sang bên kia.
func feedPostColumns(viewer string) string {
	return `
	p.id, p.user_id, u.username, u.first_name, u.last_name, COALESCE(u.avatar_url, ''),
	p.title, p.content, p.tags, p.created_at, p.updated_at, p.version,
	(SELECT COUNT(*) FROM comments c WHERE c.post_id = p.id) AS comment_count,
	(SELECT COUNT(*) FROM likes l WHERE l.post_id = p.id) AS like_count,
	EXISTS (SELECT 1 FROM likes l WHERE l.post_id = p.id AND l.user_id = ` + viewer + `) AS is_liked,
	EXISTS (SELECT 1 FROM saved_posts sp WHERE sp.post_id = p.id AND sp.user_id = ` + viewer + `) AS is_saved
	`
}

// scanFeedPost reads one row of feedPostColumns, in the same order.
// scanFeedPost đọc một dòng của feedPostColumns, theo đúng thứ tự đó.
func scanFeedPost(rows *sql.Rows) (model.FeedPost, error) {
	var post model.FeedPost
	err := rows.Scan(
		&post.ID, &post.UserID, &post.User.UserName, &post.User.FirstName, &post.User.LastName,
		&post.User.AvatarURL, &post.Title, &post.Content, pq.Array(&post.Tags),
		&post.CreatedAt, &post.UpdatedAt, &post.Version,
		&post.CommentCount, &post.LikeCount, &post.IsLiked, &post.IsSaved,
	)
	post.User.ID = post.UserID
	return post, err
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
	SELECT ` + feedPostColumns("$1") + `
	FROM posts p
	JOIN users u ON u.id = p.user_id
	WHERE ` + feedFilter + `
	ORDER BY p.created_at DESC, p.id DESC
	LIMIT $2 OFFSET $3
	`

	return paginate(ctx, s.db, page, countQuery, pageQuery, []any{userID}, scanFeedPost)
}

// GetSaved returns the posts viewerID has saved, most recently saved first.
//
// The order comes from saved_posts.created_at, not the post's: a saved list
// is a reading list, so it belongs in the order things were put on it, not
// the order they were written.
//
// GetSaved trả về những bài viewerID đã lưu, lưu gần đây nhất trước.
//
// Thứ tự lấy theo saved_posts.created_at chứ không phải của bài viết: danh
// sách đã lưu là một danh sách để đọc lại, nên nó phải theo thứ tự được cất
// vào, không phải thứ tự được viết ra.
func (s *PostStore) GetSaved(ctx context.Context, viewerID int64, page PaginationQuery) ([]model.FeedPost, int64, error) {
	countQuery := `SELECT COUNT(*) FROM saved_posts WHERE user_id = $1`
	pageQuery := `
	SELECT ` + feedPostColumns("$1") + `
	FROM saved_posts s
	JOIN posts p ON p.id = s.post_id
	JOIN users u ON u.id = p.user_id
	WHERE s.user_id = $1
	ORDER BY s.created_at DESC, p.id DESC
	LIMIT $2 OFFSET $3
	`

	return paginate(ctx, s.db, page, countQuery, pageQuery, []any{viewerID}, scanFeedPost)
}
