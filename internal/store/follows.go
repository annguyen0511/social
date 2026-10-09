package store

import (
	"context"
	"database/sql"

	"github.com/annguyen0511/social/internal/model"
)

type FollowStore struct {
	db *sql.DB
}

func (s *FollowStore) Follow(ctx context.Context, followerId int64, followingId int64) error {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	blocked, err := blockExistsTx(ctx, tx, followerId, followingId)
	if err != nil {
		return err
	}
	if blocked {
		return ErrBlocked
	}

	query := `
	INSERT INTO follows (follower_id, following_id)
	VALUES ($1, $2)
	ON CONFLICT DO NOTHING
	`
	if _, err := tx.ExecContext(ctx, query, followerId, followingId); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *FollowStore) Unfollow(ctx context.Context, followerId int64, followingId int64) error {
	query := `
	DELETE FROM follows
	WHERE follower_id = $1 AND following_id = $2
	`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	_, err := s.db.ExecContext(ctx, query, followerId, followingId)
	if err != nil {
		return err
	}
	return nil
}

// Counts returns how many people follow userID and how many userID follows.
//
// The follows table stores one direction per row, so the two numbers are the
// same table read from opposite ends: a row is a follower of userID when
// following_id is theirs, and someone they follow when follower_id is.
// Getting that backwards swaps the two figures on every profile, which is the
// kind of mistake nothing crashes on.
//
// Counts trả về có bao nhiêu người theo dõi userID và userID đang theo dõi bao
// nhiêu người.
//
// Bảng follows lưu mỗi dòng một chiều, nên hai con số là cùng một bảng đọc từ
// hai đầu ngược nhau: một dòng là người theo dõi userID khi following_id là
// của họ, và là người mà họ theo dõi khi follower_id là của họ. Nhầm chiều sẽ
// đảo hai con số trên mọi trang cá nhân, kiểu lỗi mà không có gì vỡ ra cả.
func (s *FollowStore) Counts(ctx context.Context, userID int64) (int64, int64, error) {
	query := `
	SELECT
		(SELECT COUNT(*) FROM follows WHERE following_id = $1) AS followers,
		(SELECT COUNT(*) FROM follows WHERE follower_id = $1) AS following
	`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	var followers, following int64
	err := s.db.QueryRowContext(ctx, query, userID).Scan(&followers, &following)
	return followers, following, err
}

// Followers lists the people who follow userID, and Following the people
// userID follows. Both report, for each row, whether viewerID already follows
// that person, so a list can draw its follow buttons without one request per
// row.
//
// The two differ by a single column, and that column is the whole meaning:
// swap following_id and follower_id and each list quietly becomes the other.
//
// Followers liệt kê những người theo dõi userID, còn Following là những người
// userID đang theo dõi. Cả hai đều báo kèm, cho từng dòng, việc viewerID đã
// theo dõi người đó hay chưa, để danh sách vẽ được nút theo dõi mà không cần
// một request cho mỗi dòng.
//
// Hai hàm chỉ khác nhau đúng một cột, mà chính cột đó là toàn bộ ý nghĩa:
// đổi chỗ following_id với follower_id thì mỗi danh sách lặng lẽ biến thành
// cái kia.
func (s *FollowStore) Followers(ctx context.Context, userID, viewerID int64, page PaginationQuery) ([]model.UserSummary, int64, error) {
	return s.listPeople(ctx, "f.following_id", "f.follower_id", userID, viewerID, page)
}

func (s *FollowStore) Following(ctx context.Context, userID, viewerID int64, page PaginationQuery) ([]model.UserSummary, int64, error) {
	return s.listPeople(ctx, "f.follower_id", "f.following_id", userID, viewerID, page)
}

// listPeople is the body both lists share. matchColumn is the side that equals
// userID, and personColumn is the side holding the person to return.
//
// The two column names are interpolated rather than bound as parameters
// because a placeholder cannot name a column. They are constants from the two
// callers above and never come from a request, which is what keeps this from
// being an injection.
//
// listPeople là phần thân chung của hai danh sách. matchColumn là phía bằng
// userID, còn personColumn là phía chứa người cần trả về.
//
// Hai tên cột được nối thẳng vào chuỗi chứ không truyền như tham số, vì một
// placeholder không thể đặt tên cột. Chúng là hằng số đến từ hai hàm ngay
// phía trên và không bao giờ đến từ request — đó chính là thứ khiến đây không
// phải một lỗ hổng tiêm nhiễm.
func (s *FollowStore) listPeople(
	ctx context.Context,
	matchColumn, personColumn string,
	userID, viewerID int64,
	page PaginationQuery,
) ([]model.UserSummary, int64, error) {
	countQuery := `SELECT COUNT(*) FROM follows f WHERE ` + matchColumn + ` = $1`
	pageQuery := `
	SELECT u.id, u.first_name, u.last_name, COALESCE(u.avatar_url, ''), u.username,
	       u.email, u.is_active, u.created_at, u.updated_at,
	       EXISTS (
	         SELECT 1 FROM follows v
	         WHERE v.follower_id = $2 AND v.following_id = u.id
	       ) AS is_following
	FROM follows f
	JOIN users u ON u.id = ` + personColumn + `
	WHERE ` + matchColumn + ` = $1
	ORDER BY f.created_at DESC, u.id DESC
	LIMIT $3 OFFSET $4
	`

	return paginateWith(ctx, s.db, page, countQuery, []any{userID}, pageQuery, []any{userID, viewerID},
		func(rows *sql.Rows) (model.UserSummary, error) {
			var user model.UserSummary
			err := rows.Scan(
				&user.ID, &user.FirstName, &user.LastName, &user.AvatarURL, &user.UserName,
				&user.Email, &user.IsActive, &user.CreatedAt, &user.UpdatedAt,
				&user.IsFollowing,
			)
			return user, err
		})
}

func (s *FollowStore) GetFollowers(ctx context.Context, followingId int64, page PaginationQuery) ([]model.Follow, int64, error) {
	countQuery := `SELECT COUNT(*) FROM follows WHERE following_id = $1`
	pageQuery := `
	SELECT follower_id, following_id, created_at
	FROM follows
	WHERE following_id = $1
	ORDER BY created_at DESC, follower_id DESC
	LIMIT $2 OFFSET $3
	`

	return paginate(ctx, s.db, page, countQuery, pageQuery, []any{followingId}, scanFollow)
}

func (s *FollowStore) GetFollowing(ctx context.Context, followerId int64, page PaginationQuery) ([]model.Follow, int64, error) {
	countQuery := `SELECT COUNT(*) FROM follows WHERE follower_id = $1`
	pageQuery := `
	SELECT follower_id, following_id, created_at
	FROM follows
	WHERE follower_id = $1
	ORDER BY created_at DESC, following_id DESC
	LIMIT $2 OFFSET $3
	`

	return paginate(ctx, s.db, page, countQuery, pageQuery, []any{followerId}, scanFollow)
}

func scanFollow(rows *sql.Rows) (model.Follow, error) {
	var follow model.Follow
	err := rows.Scan(&follow.FollowerID, &follow.FollowingID, &follow.CreatedAt)
	return follow, err
}

func (s *FollowStore) IsFollowing(ctx context.Context, followerId int64, followingId int64) (bool, error) {
	query := `
	SELECT EXISTS(
		SELECT 1
		FROM follows
		WHERE follower_id = $1 AND following_id = $2
	)
	`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	var exists bool
	if err := s.db.QueryRowContext(ctx, query, followerId, followingId).Scan(&exists); err != nil {
		return false, err
	}

	return exists, nil
}
