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

export type FriendshipStatus = {
  user_id: number
  is_following: boolean
  is_blocking: boolean
  is_close_friend: boolean
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
  created_at: string
  version: number
}

export type RegisteredUser = {
  user: User
  // Only present outside production, where the API has no mailer to send it.
  // Chỉ có khi chạy ngoài production, nơi API chưa có mailer để gửi đi.
  token?: string
}
