package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/annguyen0511/social/internal/model"
	"github.com/lib/pq"
)

type PostStore struct {
	db *sql.DB
}

func (s *PostStore) Create(ctx context.Context, post *model.Post) error {
	query := `
	INSERT INTO posts (content, title, user_id, tags, visibility)
	VALUES ($1, $2, $3, $4, $5) RETURNING id, created_at, updated_at
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
		post.Visibility,
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
	SET content = $1, title = $2, tags = $3, visibility = $4, version = version + 1, updated_at = NOW()
	WHERE id = $5 AND version = $6 RETURNING version
	`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	err := s.db.QueryRowContext(
		ctx,
		query,
		post.Content,
		post.Title,
		pq.Array(post.Tags),
		post.Visibility,
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
	SELECT p.id, p.content, p.title, p.user_id, p.tags, p.visibility, p.created_at, p.updated_at, p.version,
	       u.id, u.username, u.first_name, u.last_name, COALESCE(u.avatar_url, ''),
	       pi.file_name, pi.width, pi.height
	FROM posts p
	JOIN users u ON u.id = p.user_id
	` + postImageJoin + `
	WHERE p.id = $1
	`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	var imageName sql.NullString
	var imageWidth, imageHeight sql.NullInt64

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&post.ID, &post.Content, &post.Title, &post.UserID, pq.Array(&post.Tags), &post.Visibility,
		&post.CreatedAt, &post.UpdatedAt, &post.Version,
		&post.User.ID, &post.User.UserName, &post.User.FirstName, &post.User.LastName, &post.User.AvatarURL,
		&imageName, &imageWidth, &imageHeight,
	)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrNotFound
		default:
			return nil, err
		}
	}

	post.Image = buildPostImage(post.ID, imageName, imageWidth, imageHeight)
	return &post, nil
}

// AttachImage records the picture belonging to a post.
//
// Kept apart from Create so the handler can write the file first, create the
// post, and only then link the two — leaving one clear place to undo if the
// link fails.
//
// AttachImage ghi nhận ảnh thuộc về một bài viết.
//
// Tách khỏi Create để handler ghi file trước, tạo bài, rồi mới nối hai thứ
// lại — nhờ vậy có đúng một chỗ rõ ràng để hoàn tác nếu bước nối thất bại.
func (s *PostStore) AttachImage(ctx context.Context, postID int64, fileName string, width, height int) error {
	query := `
	INSERT INTO post_images (post_id, file_name, width, height)
	VALUES ($1, $2, $3, $4)
	`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	_, err := s.db.ExecContext(ctx, query, postID, fileName, width, height)
	return err
}

// ImageName returns the stored file name of a post's picture, or "" when it
// has none.
//
// Two callers need it and neither can use the URL in the model: the route
// that serves the bytes has to open the file, and deleting a post has to
// remove it. ON DELETE CASCADE clears the row but never touches the disk.
//
// ImageName trả về tên file ảnh đã lưu của một bài, hoặc "" nếu bài không có
// ảnh.
//
// Hai nơi cần tới nó mà không nơi nào dùng được URL trong model: route phục
// vụ nội dung phải mở file, còn việc xoá bài thì phải xoá file đi. ON DELETE
// CASCADE dọn dòng dữ liệu nhưng không bao giờ đụng tới đĩa.
func (s *PostStore) ImageName(ctx context.Context, postID int64) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	var name string
	err := s.db.QueryRowContext(ctx, `SELECT file_name FROM post_images WHERE post_id = $1`, postID).Scan(&name)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return name, err
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
// CountByUser returns how many of an author's posts the viewer may see.
//
// It takes the viewer for the same reason the listing does: a count that
// includes posts the reader cannot open tells them those posts exist.
//
// CountByUser trả về số bài của một tác giả mà người xem được phép thấy.
//
// Nó nhận người xem vì đúng lý do mà danh sách cũng nhận: một con số bao gồm
// cả những bài người đọc không mở được chính là đang nói cho họ biết những
// bài đó tồn tại.
func (s *PostStore) CountByUser(ctx context.Context, authorID, viewerID int64) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	query := `SELECT COUNT(*) FROM posts p WHERE p.user_id = $1` + visibleToViewer("$2")

	var total int64
	err := s.db.QueryRowContext(ctx, query, authorID, viewerID).Scan(&total)
	return total, err
}

func (s *PostStore) GetByUser(ctx context.Context, authorID, viewerID int64, page PaginationQuery) ([]model.FeedPost, int64, error) {
	// Both queries take the viewer now, so the two argument lists are the
	// same again and plain paginate is enough.
	//
	// Cả hai truy vấn giờ đều nhận người xem, nên hai bộ tham số lại giống
	// nhau và paginate thường là đủ.
	countQuery := `SELECT COUNT(*) FROM posts p WHERE p.user_id = $1` + visibleToViewer("$2")
	pageQuery := `
	SELECT ` + feedPostColumns("$2") + `
	FROM posts p
	JOIN users u ON u.id = p.user_id
	` + postImageJoin + `
	WHERE p.user_id = $1` + visibleToViewer("$2") + `
	ORDER BY p.created_at DESC, p.id DESC
	LIMIT $3 OFFSET $4
	`

	return paginate(ctx, s.db, page, countQuery, pageQuery, []any{authorID, viewerID}, scanFeedPost)
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
	p.title, p.content, p.tags, p.visibility, p.created_at, p.updated_at, p.version,
	(SELECT COUNT(*) FROM comments c WHERE c.post_id = p.id) AS comment_count,
	(SELECT COUNT(*) FROM likes l WHERE l.post_id = p.id) AS like_count,
	EXISTS (SELECT 1 FROM likes l WHERE l.post_id = p.id AND l.user_id = ` + viewer + `) AS is_liked,
	EXISTS (SELECT 1 FROM saved_posts sp WHERE sp.post_id = p.id AND sp.user_id = ` + viewer + `) AS is_saved,
	(SELECT COUNT(*) FROM reposts rp WHERE rp.post_id = p.id) AS repost_count,
	EXISTS (SELECT 1 FROM reposts rp WHERE rp.post_id = p.id AND rp.user_id = ` + viewer + `) AS is_reposted,
	pi.file_name, pi.width, pi.height
	`
}

// scanFeedPost reads one row of feedPostColumns, in the same order.
// scanFeedPost đọc một dòng của feedPostColumns, theo đúng thứ tự đó.
func scanFeedPost(rows *sql.Rows) (model.FeedPost, error) {
	var post model.FeedPost
	// A post with no picture gives NULL for all three, which only a pointer
	// or a sql.Null* can hold. Scanning straight into int would fail the row.
	//
	// Bài không có ảnh trả về NULL cho cả ba cột, mà chỉ con trỏ hoặc
	// sql.Null* mới chứa được. Quét thẳng vào int sẽ làm hỏng cả dòng.
	var imageName sql.NullString
	var imageWidth, imageHeight sql.NullInt64

	err := rows.Scan(
		&post.ID, &post.UserID, &post.User.UserName, &post.User.FirstName, &post.User.LastName,
		&post.User.AvatarURL, &post.Title, &post.Content, pq.Array(&post.Tags), &post.Visibility,
		&post.CreatedAt, &post.UpdatedAt, &post.Version,
		&post.CommentCount, &post.LikeCount, &post.IsLiked, &post.IsSaved,
		&post.RepostCount, &post.IsReposted,
		&imageName, &imageWidth, &imageHeight,
	)
	post.User.ID = post.UserID
	post.Image = buildPostImage(post.ID, imageName, imageWidth, imageHeight)
	return post, err
}

// buildPostImage turns the three nullable columns into the model's optional
// image, and nothing when the post has none.
//
// The URL is built here rather than stored, so the route can change without
// a data migration — the database keeps only the file name.
//
// buildPostImage biến ba cột nullable thành ảnh tuỳ chọn của model, và không
// tạo gì khi bài không có ảnh.
//
// URL được dựng ở đây chứ không lưu xuống, nên đổi route sau này không cần di
// trú dữ liệu — database chỉ giữ tên file.
func buildPostImage(postID int64, name sql.NullString, width, height sql.NullInt64) *model.PostImage {
	if !name.Valid {
		return nil
	}
	return &model.PostImage{
		URL:    fmt.Sprintf("/v1/posts/%d/image", postID),
		Width:  int(width.Int64),
		Height: int(height.Int64),
	}
}

func (s *PostStore) GetUserFeed(ctx context.Context, userID int64, page PaginationQuery) ([]model.FeedPost, int64, error) {
	// A subquery rather than JOIN follows: an inner join drops the user's own
	// posts when they follow nobody, and joins each own post to every follows
	// row otherwise, which multiplies COUNT(c.id).
	feedFilter := `
	p.user_id = $1
	OR p.user_id IN (SELECT following_id FROM follows WHERE follower_id = $1)
	`

	countQuery := `SELECT COUNT(*) FROM posts p WHERE ` + feedFilter + visibleToViewer("$1")
	pageQuery := `
	SELECT ` + feedPostColumns("$1") + `
	FROM posts p
	JOIN users u ON u.id = p.user_id
	` + postImageJoin + `
	WHERE ` + feedFilter + visibleToViewer("$1") + `
	ORDER BY p.created_at DESC, p.id DESC
	LIMIT $2 OFFSET $3
	`

	return paginate(ctx, s.db, page, countQuery, pageQuery, []any{userID}, scanFeedPost)
}

// postImageJoin brings the attached picture along, if there is one.
//
// A LEFT JOIN and not an inner one: most posts have no image, and an inner
// join would silently drop every one of them. It is a constant so the four
// list queries cannot end up joining differently.
//
// There is at most one row per post today — post_images has a UNIQUE on
// post_id — which is what lets a plain join work here. Allowing several
// pictures later means dropping that constraint and gathering them with
// json_agg instead, because the join would then multiply the post rows.
//
// postImageJoin kéo theo ảnh đính kèm, nếu có.
//
// Dùng LEFT JOIN chứ không phải inner: phần lớn bài không có ảnh, mà inner
// join sẽ âm thầm loại sạch chúng. Nó là hằng số để bốn truy vấn danh sách
// không thể join theo những cách khác nhau.
//
// Hiện mỗi bài nhiều nhất một dòng — post_images có UNIQUE trên post_id — và
// chính điều đó khiến một phép join thường là đủ. Sau này cho phép nhiều ảnh
// thì phải bỏ ràng buộc đó và gom bằng json_agg, vì lúc ấy phép join sẽ nhân
// số dòng của bài lên.
const postImageJoin = `LEFT JOIN post_images pi ON pi.post_id = p.id`

// Visibility values a post can carry. They are strings rather than a Go enum
// because they travel to the database and to the browser unchanged.
//
// Các giá trị hiển thị mà một bài viết có thể mang. Chúng là chuỗi chứ không
// phải enum của Go, vì chúng đi xuống database và lên trình duyệt y nguyên.
const (
	VisibilityPublic  = "public"
	VisibilityPrivate = "private"
)

// visibleToViewer drops posts the reader is not allowed to see.
//
// A private post reaches three kinds of people and nobody else: its author,
// and whoever the author has put on their close friends list. Note the
// direction — close_friends.user_id is the person who owns the list, so the
// author's list is the one that matters, not the reader's.
//
// Every query that returns posts needs this, including the ones that only
// count them: a profile saying "12 bài viết" above a list of 9 is a leak of
// exactly the three the reader was not meant to know about.
//
// visibleToViewer loại những bài mà người đọc không được phép xem.
//
// Một bài riêng tư chỉ đến được ba loại người và không ai khác: tác giả của
// nó, và những ai tác giả đã đưa vào danh sách bạn thân. Chú ý chiều —
// close_friends.user_id là người sở hữu danh sách, nên danh sách của tác giả
// mới là thứ có ý nghĩa, không phải của người đọc.
//
// Mọi truy vấn trả về bài viết đều cần mệnh đề này, kể cả những truy vấn chỉ
// đếm: một trang cá nhân ghi "12 bài viết" bên trên một danh sách 9 bài chính
// là đang tiết lộ đúng ba bài mà người đọc lẽ ra không được biết.
func visibleToViewer(viewer string) string {
	return `
	AND (
		p.visibility = 'public'
		OR p.user_id = ` + viewer + `
		OR EXISTS (
			SELECT 1 FROM close_friends cf
			WHERE cf.user_id = p.user_id AND cf.friend_id = ` + viewer + `
		)
	)
	`
}

// notBlockedByViewer drops rows whose author has a block with the reader, in
// either direction.
//
// The route-level gate cannot do this job: a saved list and a repost list
// hold posts by many different authors, so there is no single person to check
// before the query runs. The filter has to live inside it.
//
// A feed needs no such clause, because blocking already removes the follows
// in both directions and a feed only holds posts by people you follow.
//
// notBlockedByViewer loại các dòng mà tác giả của nó đang có lệnh chặn với
// người đọc, ở bất kỳ chiều nào.
//
// Cổng chặn ở tầng route không làm được việc này: danh sách đã lưu và danh
// sách đã repost chứa bài của nhiều tác giả khác nhau, nên không có một người
// duy nhất nào để kiểm trước khi chạy truy vấn. Bộ lọc buộc phải nằm bên
// trong nó.
//
// Feed thì không cần mệnh đề này, vì việc chặn vốn đã gỡ follow cả hai chiều
// mà feed chỉ chứa bài của những người bạn đang theo dõi.
func notBlockedByViewer(viewer string) string {
	return `
	AND NOT EXISTS (
		SELECT 1 FROM blocks b
		WHERE (b.blocker_id = ` + viewer + ` AND b.blocked_id = p.user_id)
		   OR (b.blocker_id = p.user_id AND b.blocked_id = ` + viewer + `)
	)
	`
}

// GetReposted returns the posts authorID has reposted, most recently first.
//
// Unlike a saved list this one is public, so it takes two ids: whose reposts
// to list, and who is looking at them — the second is what decides the
// is_liked, is_saved and is_reposted flags on each row.
//
// GetReposted trả về những bài authorID đã repost, mới nhất trước.
//
// Khác với danh sách đã lưu, danh sách này là công khai nên nhận hai id: liệt
// kê repost của ai, và ai đang xem — cái thứ hai mới là thứ quyết định các cờ
// is_liked, is_saved và is_reposted trên từng dòng.
func (s *PostStore) GetReposted(ctx context.Context, authorID, viewerID int64, page PaginationQuery) ([]model.FeedPost, int64, error) {
	// Both queries carry the filter, or the total would count rows the page
	// never returns and the last page would come back empty.
	//
	// Cả hai truy vấn đều mang bộ lọc, nếu không thì tổng số sẽ đếm cả những
	// dòng mà trang không bao giờ trả về, và trang cuối sẽ ra rỗng.
	countQuery := `
	SELECT COUNT(*) FROM reposts r JOIN posts p ON p.id = r.post_id
	WHERE r.user_id = $1 ` + notBlockedByViewer("$2") + visibleToViewer("$2")
	pageQuery := `
	SELECT ` + feedPostColumns("$2") + `
	FROM reposts r
	JOIN posts p ON p.id = r.post_id
	JOIN users u ON u.id = p.user_id
	` + postImageJoin + `
	WHERE r.user_id = $1 ` + notBlockedByViewer("$2") + visibleToViewer("$2") + `
	ORDER BY r.created_at DESC, p.id DESC
	LIMIT $3 OFFSET $4
	`

	return paginate(ctx, s.db, page, countQuery, pageQuery, []any{authorID, viewerID}, scanFeedPost)
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
	countQuery := `
	SELECT COUNT(*) FROM saved_posts s JOIN posts p ON p.id = s.post_id
	WHERE s.user_id = $1 ` + notBlockedByViewer("$1") + visibleToViewer("$1")
	pageQuery := `
	SELECT ` + feedPostColumns("$1") + `
	FROM saved_posts s
	JOIN posts p ON p.id = s.post_id
	JOIN users u ON u.id = p.user_id
	` + postImageJoin + `
	WHERE s.user_id = $1 ` + notBlockedByViewer("$1") + visibleToViewer("$1") + `
	ORDER BY s.created_at DESC, p.id DESC
	LIMIT $2 OFFSET $3
	`

	return paginate(ctx, s.db, page, countQuery, pageQuery, []any{viewerID}, scanFeedPost)
}
