package store

import (
	"context"
	"database/sql"

	"github.com/annguyen0511/social/internal/model"
)

type BlockStore struct {
	db *sql.DB
}

func (s *BlockStore) Block(ctx context.Context, blockerId int64, blockedId int64) error {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// unfollow if exist before block
	dropFollows := `
	DELETE FROM follows
	WHERE (follower_id = $1 AND following_id = $2)
	   OR (follower_id = $2 AND following_id = $1)
	`
	if _, err := tx.ExecContext(ctx, dropFollows, blockerId, blockedId); err != nil {
		return err
	}

	// remove from close friends if exist
	dropCloseFriends := `
	DELETE FROM close_friends
	WHERE (user_id = $1 AND friend_id = $2)
	   OR (user_id = $2 AND friend_id = $1)
	`
	if _, err := tx.ExecContext(ctx, dropCloseFriends, blockerId, blockedId); err != nil {
		return err
	}

	insertBlock := `
	INSERT INTO blocks (blocker_id, blocked_id)
	VALUES ($1, $2)
	ON CONFLICT DO NOTHING
	`
	if _, err := tx.ExecContext(ctx, insertBlock, blockerId, blockedId); err != nil {
		return err
	}

	return tx.Commit()
}

// check block exists in transaction
func blockExistsTx(ctx context.Context, tx *sql.Tx, userId int64, otherId int64) (bool, error) {
	query := `
	SELECT EXISTS(
		SELECT 1 FROM blocks
		WHERE (blocker_id = $1 AND blocked_id = $2)
		   OR (blocker_id = $2 AND blocked_id = $1)
	)
	`

	var exists bool
	err := tx.QueryRowContext(ctx, query, userId, otherId).Scan(&exists)
	return exists, err
}

func (s *BlockStore) Unblock(ctx context.Context, blockerId int64, blockedId int64) error {
	query := `
	DELETE FROM blocks
	WHERE blocker_id = $1 AND blocked_id = $2
	`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	_, err := s.db.ExecContext(ctx, query, blockerId, blockedId)
	if err != nil {
		return err
	}
	return nil
}

func (s *BlockStore) ListBlocking(ctx context.Context, blockerId int64) ([]model.Block, error) {
	query := `
	SELECT blocker_id, blocked_id, created_at
	FROM blocks
	WHERE blocker_id = $1
	`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	rows, err := s.db.QueryContext(ctx, query, blockerId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	blocks := []model.Block{}
	for rows.Next() {
		var block model.Block
		if err := rows.Scan(&block.BlockerID, &block.BlockedID, &block.CreatedAt); err != nil {
			return nil, err
		}
		blocks = append(blocks, block)
	}

	return blocks, nil
}

func (s *BlockStore) IsBlocking(ctx context.Context, blockerId int64, blockedId int64) (bool, error) {
	query := `
	SELECT EXISTS(
		SELECT 1 FROM blocks
		WHERE blocker_id = $1 AND blocked_id = $2
	)
	`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	var exists bool
	if err := s.db.QueryRowContext(ctx, query, blockerId, blockedId).Scan(&exists); err != nil {
		return false, err
	}

	return exists, nil
}
