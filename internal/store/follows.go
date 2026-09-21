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
