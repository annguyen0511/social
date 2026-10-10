package model

type Post struct {
	ID      int64    `json:"id" db:"id"`
	Content string   `json:"content" db:"content"`
	Title   string   `json:"title" db:"title"`
	UserID  int64    `json:"user_id" db:"user_id"`
	Tags    []string `json:"tags" db:"tags"`
	// Visibility is "public" or "private". A private post is readable by its
	// author and by whoever the author listed as a close friend.
	//
	// Visibility là "public" hoặc "private". Bài riêng tư chỉ tác giả và
	// những người tác giả đã đưa vào danh sách bạn thân mới đọc được.
	Visibility string    `json:"visibility" db:"visibility" enums:"public,private" example:"public"`
	CreatedAt  string    `json:"created_at" db:"created_at"`
	UpdatedAt  string    `json:"updated_at" db:"updated_at"`
	Comments   []Comment `json:"comments"`
	Version    int       `json:"version" db:"version"`
	User       User      `json:"user"`

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

	// RepostCount and IsReposted follow the same shape as the like pair: a
	// public number plus a flag about the person asking.
	//
	// A repost is a relation, not a post of its own, so it never appears in
	// posts_count — reposting someone does not make you the author of
	// anything.
	//
	// RepostCount và IsReposted theo đúng hình dạng của cặp like: một con số
	// công khai cộng một cờ nói về người đang hỏi.
	//
	// Repost là một quan hệ chứ không phải một bài viết riêng, nên nó không
	// bao giờ được tính vào posts_count — repost bài người khác không khiến
	// bạn thành tác giả của thứ gì cả.
	RepostCount int64 `json:"repost_count"`
	IsReposted  bool  `json:"is_reposted"`

	// Image is nil for a post with no picture. The URL inside it goes through
	// the API rather than straight to a file, because a private post's
	// picture has to be as private as the post.
	//
	// Image là nil với bài không có ảnh. URL bên trong đi qua API chứ không
	// trỏ thẳng tới file, vì ảnh của một bài riêng tư phải riêng tư đúng như
	// chính bài đó.
	Image *PostImage `json:"image"`
} //@name PostViewModel

type FeedPost struct {
	Post
	CommentCount int64 `json:"comment_count"`
} //@name FeedPostViewModel

// PostImage is the picture attached to a post.
//
// Width and Height travel with it so a browser can reserve the right space
// before the bytes arrive; without them a feed jumps as each picture lands.
//
// PostImage là ảnh đính kèm một bài viết.
//
// Width và Height đi kèm để trình duyệt chừa đúng chỗ trước khi dữ liệu ảnh
// về; thiếu chúng thì feed nhảy giật mỗi lần một tấm ảnh hiện ra.
type PostImage struct {
	URL    string `json:"url" example:"/v1/posts/42/image"`
	Width  int    `json:"width" example:"1080"`
	Height int    `json:"height" example:"810"`
} //@name PostImageViewModel
