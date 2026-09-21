package model

type Post struct {
	ID        int64     `json:"id" db:"id"`
	Content   string    `json:"content" db:"content"`
	Title     string    `json:"title" db:"title"`
	UserID    int64     `json:"user_id" db:"user_id"`
	Tags      []string  `json:"tags" db:"tags"`
	CreatedAt string    `json:"created_at" db:"created_at"`
	UpdatedAt string    `json:"updated_at" db:"updated_at"`
	Comments  []Comment `json:"comments"`
	Version   int       `json:"version" db:"version"`
	User      User      `json:"user"`
}

type FeedPost struct {
	Post
	CommentCount int64 `json:"comment_count"`
	LikeCount    int64 `json:"like_count"`
}
