package store

import (
	"context"
	"database/sql"
)

type CloseFriendStore struct {
	db *sql.DB
}

func (s *CloseFriendStore) Add(ctx context.Context, userID int64, friendID int64) error {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	blocked, err := blockExistsTx(ctx, tx, userID, friendID)
	if err != nil {
		return err
	}
	if blocked {
		return ErrBlocked
	}

	query := `
	INSERT INTO close_friends (user_id, friend_id)
	VALUES ($1, $2)
	ON CONFLICT DO NOTHING
	`
	if _, err := tx.ExecContext(ctx, query, userID, friendID); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *CloseFriendStore) Remove(ctx context.Context, userID int64, friendID int64) error {
	query := `
	DELETE FROM close_friends
	WHERE user_id = $1 AND friend_id = $2
	`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	_, err := s.db.ExecContext(ctx, query, userID, friendID)
	if err != nil {
		return err
	}
	return nil
}

func (s *CloseFriendStore) List(ctx context.Context, userID int64) ([]int64, error) {
	query := `
	SELECT friend_id
	FROM close_friends
	WHERE user_id = $1
	`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	friendIDs := []int64{}
	for rows.Next() {
		var friendID int64
		if err := rows.Scan(&friendID); err != nil {
			return nil, err
		}
		friendIDs = append(friendIDs, friendID)
	}

	return friendIDs, nil
}

func (s *CloseFriendStore) IsCloseFriend(ctx context.Context, userID, friendID int64) (bool, error) {
	query := `
	SELECT EXISTS(
		SELECT 1 FROM close_friends
		WHERE user_id = $1 AND friend_id = $2
	)
	`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	var exists bool
	if err := s.db.QueryRowContext(ctx, query, userID, friendID).Scan(&exists); err != nil {
		return false, err
	}

	return exists, nil
}
