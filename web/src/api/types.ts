// Shapes the Go API returns. Hand-written for now; once a client is generated
// from docs/swagger.json these can be replaced by the generated types.
//
// Các kiểu dữ liệu API Go trả về. Hiện viết tay; khi nào sinh client từ
// docs/swagger.json thì thay bằng kiểu sinh tự động.

export type Envelope<T> = {
  status: string
  data: T
  is_success: boolean
  message?: string
}

export type ApiError = {
  error: boolean
  message: string
}

export type Pagination<T> = {
  items: T[]
  page: number
  page_size: number
  total_items: number
  total_pages: number
}

export type User = {
  id: number
  first_name: string
  last_name: string
  avatar_url: string
  username: string
  email: string
  is_active: boolean
  created_at: string
  updated_at: string
}

// Trang cá nhân trả về thêm hai con số mà chỉ nó cần. Chúng không nằm trong
// User vì kiểu đó còn là tác giả bài viết, tác giả bình luận và kết quả tìm
// kiếm — những chỗ không hề đếm follow.
//
// A profile carries two extra figures only it needs. They are not on User
// because that type is also a post's author, a comment's author and a search
// result, none of which count follows.
export type UserProfile = User & {
  followers_count: number
  following_count: number
}

// A search result: the user, plus whether the person searching already
// follows them. The flag comes down with the row so each result can draw its
// follow button without one extra request per row.
//
// Một dòng kết quả tìm kiếm: thông tin user, kèm việc người tìm đã theo dõi
// họ chưa. Cờ này về cùng dòng dữ liệu nên mỗi kết quả vẽ được nút theo dõi
// mà không cần thêm một request cho từng dòng.
export type SearchedUser = User & {
  is_following: boolean
}

// Mỗi field là optional: field không gửi thì giữ nguyên giá trị cũ. Gửi
// avatar_url là chuỗi rỗng nghĩa là xoá ảnh.
//
// Every field is optional: one left out keeps its current value. Sending
// avatar_url as an empty string removes the picture.
export type ProfileUpdate = {
  first_name?: string
  last_name?: string
  username?: string
  avatar_url?: string
}

export type FriendshipStatus = {
  user_id: number
  is_following: boolean
  is_blocking: boolean
  is_close_friend: boolean
}

export type Comment = {
  id: number
  post_id: number
  user_id: number
  content: string
  created_at: string
  updated_at: string
  user: User
}

// Chi tiết một bài: giống bài trong feed nhưng kèm danh sách bình luận, và
// không có hai con số đếm vì endpoint chi tiết trả về bình luận thật.
//
// One post in detail: the same shape as a feed post but carrying its comments,
// and without the two counts, because the detail endpoint returns the real
// comments instead.
export type Post = {
  id: number
  title: string
  content: string
  user_id: number
  tags: string[] | null
  user: User
  comments: Comment[] | null
  created_at: string
  updated_at: string
  version: number
  like_count: number
  is_liked: boolean
}

// Trạng thái thích sau khi bấm, do server trả về. Client không tự cộng trừ:
// một cái nút tự cộng thêm một sẽ sai ngay khi có người khác cũng vừa thích
// bài đó.
//
// The like state after a press, as the server reports it. The client does not
// do its own arithmetic: a button that adds one locally is wrong the moment
// somebody else likes the same post.
export type LikeState = {
  like_count: number
  is_liked: boolean
}

export type PostCreate = {
  title: string
  content: string
  tags?: string[]
}

export type PostUpdate = {
  title?: string
  content?: string
  tags?: string[]
}

export type FeedPost = {
  id: number
  title: string
  content: string
  user_id: number
  tags: string[] | null
  user: User
  comment_count: number
  like_count: number
  is_liked: boolean
  created_at: string
  updated_at: string
  version: number
}

export type RegisteredUser = {
  user: User
  // Only present outside production, where the API has no mailer to send it.
  // Chỉ có khi chạy ngoài production, nơi API chưa có mailer để gửi đi.
  token?: string
}
