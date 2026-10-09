import { PostList } from '../posts/PostList'

/**
 * The feed: the reader's own posts plus everyone they follow.
 *
 * The paging, the skeleton and the empty state all live in PostList, which a
 * profile uses too. Only the URL and the cache key differ.
 *
 * Bảng tin: bài của chính người dùng cộng với bài của những người họ theo dõi.
 *
 * Phần phân trang, khung chờ và trạng thái rỗng đều nằm trong PostList, thứ mà
 * trang cá nhân cũng dùng. Chỉ khác URL và key cache.
 */
export function FeedPage() {
  return (
    <main className="mx-auto max-w-2xl p-6">
      <h1 className="mb-6 text-xl font-semibold">Bảng tin</h1>

      <PostList
        queryKey={['feed']}
        path="/v1/users/feed"
        empty="Chưa có bài nào. Dùng ô tìm kiếm ở trên để theo dõi vài người, hoặc bấm Tạo để viết bài đầu tiên."
      />
    </main>
  )
}
