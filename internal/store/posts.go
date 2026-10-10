package store

import (
	"context"
	"database/sql"
	"encoding/json"
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
	       ` + postImagesColumn + `
	FROM posts p
	JOIN users u ON u.id = p.user_id
	WHERE p.id = $1
	`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	var images []byte

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&post.ID, &post.Content, &post.Title, &post.UserID, pq.Array(&post.Tags), &post.Visibility,
		&post.CreatedAt, &post.UpdatedAt, &post.Version,
		&post.User.ID, &post.User.UserName, &post.User.FirstName, &post.User.LastName, &post.User.AvatarURL,
		&images,
	)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrNotFound
		default:
			return nil, err
		}
	}

	post.Images, err = buildPostImages(post.ID, images)
	if err != nil {
		return nil, err
	}
	return &post, nil
}

// AttachImage records one picture belonging to a post, at a given position.
//
// Kept apart from Create so the handler can write the files first, create the
// post, and only then link them — leaving one clear place to undo if a link
// fails.
//
// AttachImage ghi nhận một ảnh thuộc về bài viết, ở một vị trí nhất định.
//
// Tách khỏi Create để handler ghi file trước, tạo bài, rồi mới nối chúng lại
// — nhờ vậy có đúng một chỗ rõ ràng để hoàn tác nếu bước nối thất bại.
func (s *PostStore) AttachImage(ctx context.Context, postID int64, fileName string, width, height, position int) error {
	query := `
	INSERT INTO post_images (post_id, file_name, width, height, position)
	VALUES ($1, $2, $3, $4, $5)
	`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	_, err := s.db.ExecContext(ctx, query, postID, fileName, width, height, position)
	return err
}

// ImageNames returns the stored file names of a post's pictures.
//
// Deleting a post needs them: ON DELETE CASCADE clears the rows but never
// touches the disk, and once the rows are gone nothing says which files they
// were.
//
// ImageNames trả về tên các file ảnh đã lưu của một bài.
//
// Việc xoá bài cần tới chúng: ON DELETE CASCADE dọn các dòng nhưng không bao
// giờ đụng tới đĩa, mà dòng đã mất rồi thì không còn gì cho biết đó là những
// file nào.
func (s *PostStore) ImageNames(ctx context.Context, postID int64) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	rows, err := s.db.QueryContext(ctx,
		`SELECT file_name FROM post_images WHERE post_id = $1 ORDER BY position, id`, postID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		names = append(names, name)
	}
	return names, rows.Err()
}

// HasImage reports whether a file belongs to a post.
//
// This is what keeps the serving route from becoming a way to read any
// picture on the server: the route is authorised for one post, so the name it
// is handed has to belong to that post and not merely be a real file.
//
// HasImage cho biết một file có thuộc về một bài viết hay không.
//
// Đây là thứ giữ cho route phục vụ ảnh khỏi trở thành đường đọc bất kỳ tấm
// ảnh nào trên server: route được cấp quyền cho một bài, nên cái tên nó nhận
// phải thuộc về chính bài đó chứ không chỉ là một file có thật.
func (s *PostStore) HasImage(ctx context.Context, postID int64, fileName string) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	var exists bool
	err := s.db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM post_images WHERE post_id = $1 AND file_name = $2)`,
		postID, fileName).Scan(&exists)
	return exists, err
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
	` + postImagesColumn + ` AS images
	`
}

// scanFeedPost reads one row of feedPostColumns, in the same order.
// scanFeedPost đọc một dòng của feedPostColumns, theo đúng thứ tự đó.
func scanFeedPost(rows *sql.Rows) (model.FeedPost, error) {
	var post model.FeedPost
	// NULL for a post with no pictures, so it has to be scanned as something
	// that can hold nothing.
	//
	// NULL với bài không có ảnh, nên phải quét vào một kiểu chứa được "không
	// có gì".
	var images []byte

	err := rows.Scan(
		&post.ID, &post.UserID, &post.User.UserName, &post.User.FirstName, &post.User.LastName,
		&post.User.AvatarURL, &post.Title, &post.Content, pq.Array(&post.Tags), &post.Visibility,
		&post.CreatedAt, &post.UpdatedAt, &post.Version,
		&post.CommentCount, &post.LikeCount, &post.IsLiked, &post.IsSaved,
		&post.RepostCount, &post.IsReposted,
		&images,
	)
	if err != nil {
		return post, err
	}

	post.User.ID = post.UserID
	post.Images, err = buildPostImages(post.ID, images)
	return post, err
}

// storedImageRow is the shape json_build_object produces above. It exists
// only to be unmarshalled into, which is why the file name never leaves it:
// the URL handed to a client is built from the post and the name together.
//
// storedImageRow là hình dạng mà json_build_object ở trên tạo ra. Nó tồn tại
// chỉ để được giải mã vào, và đó là lý do tên file không bao giờ rời khỏi
// đây: URL đưa cho client được dựng từ bài viết cộng với tên file.
type storedImageRow struct {
	Name   string `json:"name"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

// buildPostImages turns the aggregated JSON into the model's pictures.
//
// The URL is built here rather than stored, so the route can change without
// a data migration — the database keeps only the file name.
//
// buildPostImages biến mảng JSON đã gom thành danh sách ảnh của model.
//
// URL được dựng ở đây chứ không lưu xuống, nên đổi route sau này không cần di
// trú dữ liệu — database chỉ giữ tên file.
func buildPostImages(postID int64, raw []byte) ([]model.PostImage, error) {
	// Always a slice, never nil: a client maps over this without checking,
	// and a null in its place would be one more thing every caller has to
	// remember.
	//
	// Luôn là slice, không bao giờ nil: client duyệt qua nó mà không kiểm
	// trước, mà một null ở đó sẽ thành thêm một thứ nữa mọi nơi gọi phải nhớ.
	images := []model.PostImage{}
	if len(raw) == 0 {
		return images, nil
	}

	var rows []storedImageRow
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil, fmt.Errorf("decode post images: %w", err)
	}

	for _, row := range rows {
		images = append(images, model.PostImage{
			URL:    fmt.Sprintf("/v1/posts/%d/image/%s", postID, row.Name),
			Width:  row.Width,
			Height: row.Height,
		})
	}
	return images, nil
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
	WHERE ` + feedFilter + visibleToViewer("$1") + `
	ORDER BY p.created_at DESC, p.id DESC
	LIMIT $2 OFFSET $3
	`

	return paginate(ctx, s.db, page, countQuery, pageQuery, []any{userID}, scanFeedPost)
}

// postImagesColumn gathers a post's pictures into one JSON array.
//
// A join cannot do this any more. With one image per post a LEFT JOIN added
// one row and nothing else; with several it multiplies the post across them,
// and every count beside it — comments, likes, reposts — would be counted
// once per picture. Aggregating in a subquery keeps one row per post.
//
// NULL comes back for a post with no pictures, which the scan turns into an
// empty slice rather than a null in the JSON a client reads.
//
// postImagesColumn gom các ảnh của một bài vào một mảng JSON.
//
// Phép join không làm được việc này nữa. Với mỗi bài một ảnh thì LEFT JOIN
// chỉ thêm một dòng và không gì khác; với nhiều ảnh thì nó nhân bài viết lên
// theo số ảnh, và mọi con số đếm bên cạnh — bình luận, thích, repost — sẽ bị
// đếm lặp lại một lần cho mỗi tấm. Gom bằng truy vấn con thì vẫn giữ mỗi bài
// một dòng.
//
// Bài không có ảnh trả về NULL, và bước quét biến nó thành một slice rỗng
// chứ không phải một null trong JSON mà client đọc.
const postImagesColumn = `
	(
		SELECT json_agg(
			json_build_object('name', i.file_name, 'width', i.width, 'height', i.height)
			ORDER BY i.position, i.id
		)
		FROM post_images i WHERE i.post_id = p.id
	)`

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
	WHERE s.user_id = $1 ` + notBlockedByViewer("$1") + visibleToViewer("$1") + `
	ORDER BY s.created_at DESC, p.id DESC
	LIMIT $2 OFFSET $3
	`

	return paginate(ctx, s.db, page, countQuery, pageQuery, []any{viewerID}, scanFeedPost)
}
