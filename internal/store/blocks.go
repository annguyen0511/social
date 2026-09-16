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
	query := `
	INSERT INTO blocks (blocker_id, blocked_id)
	VALUES ($1, $2)
	`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	_, err := s.db.ExecContext(ctx, query, blockerId, blockedId)
	if err != nil {
		return err
	}
	return nil
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
	SELECT blocked_id
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

	var blocks []model.Block
	for rows.Next() {
		var block model.Block
		if err := rows.Scan(&block.BlockedID); err != nil {
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
