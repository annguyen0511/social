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

func (s *FollowStore) GetFollowers(ctx context.Context, followingId int64) ([]model.Follow, error) {
	query := `
	SELECT follower_id, following_id, created_at
	FROM follows
	WHERE following_id = $1
	`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	rows, err := s.db.QueryContext(ctx, query, followingId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	follows := []model.Follow{}
	for rows.Next() {
		var follow model.Follow
		if err := rows.Scan(&follow.FollowerID, &follow.FollowingID, &follow.CreatedAt); err != nil {
			return nil, err
		}
		follows = append(follows, follow)
	}

	return follows, nil
}

func (s *FollowStore) GetFollowing(ctx context.Context, followerId int64) ([]model.Follow, error) {
	query := `
	SELECT follower_id, following_id, created_at
	FROM follows
	WHERE follower_id = $1
	`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	rows, err := s.db.QueryContext(ctx, query, followerId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	follows := []model.Follow{}
	for rows.Next() {
		var follow model.Follow
		if err := rows.Scan(&follow.FollowerID, &follow.FollowingID, &follow.CreatedAt); err != nil {
			return nil, err
		}
		follows = append(follows, follow)
	}

	return follows, nil
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
