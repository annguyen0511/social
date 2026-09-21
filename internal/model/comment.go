package model

type Comment struct {
	ID        int64  `json:"id" db:"id"`
	PostID    int64  `json:"post_id" db:"post_id"`
	UserID    int64  `json:"user_id" db:"user_id"`
	Content   string `json:"content" db:"content"`
	CreatedAt string `json:"created_at" db:"created_at"`
	UpdatedAt string `json:"updated_at" db:"updated_at"`
	User      User   `json:"user"`
} //@name CommentViewModel
