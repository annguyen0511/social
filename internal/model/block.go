package model

type Block struct {
	BlockerID int64  `json:"blocker_id" db:"blocker_id"`
	BlockedID int64  `json:"blocked_id" db:"blocked_id"`
	CreatedAt string `json:"created_at" db:"created_at"`
}
