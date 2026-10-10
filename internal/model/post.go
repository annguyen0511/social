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

	// LikeCount and IsLiked live on Post rather than only on FeedPost,
	// because a post read on its own needs them too. Declaring LikeCount in
	// both would leave two fields with the same JSON name at different
	// depths, where the shallower one silently wins.
	//
	// LikeCount và IsLiked nằm ở Post chứ không chỉ ở FeedPost, vì một bài
	// đọc riêng lẻ cũng cần tới chúng. Khai báo LikeCount ở cả hai sẽ tạo ra
	// hai field cùng tên JSON ở hai độ sâu khác nhau, và cái nông hơn âm thầm
	// thắng.
	LikeCount int64 `json:"like_count"`
	// IsLiked is about the person asking, not about the post, so the same row
	// answers differently for two readers.
	//
	// IsLiked nói về người đang hỏi chứ không phải về bài viết, nên cùng một
	// dòng dữ liệu trả lời khác nhau với hai người đọc khác nhau.
	IsLiked bool `json:"is_liked"`

	// IsSaved is also about the person asking. It has no count beside it on
	// purpose: a save is private, so how many people saved a post is nobody's
	// business but theirs.
	//
	// IsSaved cũng nói về người đang hỏi. Nó cố tình không có số đếm đi kèm:
	// việc lưu bài là riêng tư, nên có bao nhiêu người đã lưu một bài là
	// chuyện của riêng họ.
	IsSaved bool `json:"is_saved"`
} //@name PostViewModel

type FeedPost struct {
	Post
	CommentCount int64 `json:"comment_count"`
} //@name FeedPostViewModel
