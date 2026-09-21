package model

type Follow struct {
	FollowerID  int64  `json:"follower_id" db:"follower_id"`
	FollowingID int64  `json:"following_id" db:"following_id"`
	CreatedAt   string `json:"created_at" db:"created_at"`
} //@name FollowViewModel
