import type { QueryClient } from '@tanstack/react-query'
import type { FeedPost, Pagination, Post } from '../../api/types'

type InfinitePosts = { pages: Pagination<FeedPost>[]; pageParams: unknown[] }

/**
 * Rewrites one post wherever it is already cached.
 *
 * The same post appears in the feed, on its author's profile, in the saved
 * list and on its own page, under four different cache keys. Invalidating all
 * of them would refetch whole lists over one button press; patching them
 * keeps every copy in step without a single request.
 *
 * Ghi lại một bài viết ở mọi nơi nó đang nằm trong cache.
 *
 * Cùng một bài xuất hiện trong bảng tin, trên trang cá nhân của tác giả,
 * trong danh sách đã lưu và ở trang riêng của nó, dưới bốn key cache khác
 * nhau. Invalidate cả bốn sẽ tải lại nguyên các danh sách chỉ vì một cú bấm
 * nút; vá trực tiếp thì mọi bản sao khớp nhau mà không tốn request nào.
 */
export function patchPostLists(
  queryClient: QueryClient,
  postID: number,
  next: Partial<FeedPost>,
): void {
  const patch = (data: InfinitePosts | undefined) => {
    if (!data) return data
    return {
      ...data,
      pages: data.pages.map((page) => ({
        ...page,
        items: page.items.map((item) => (item.id === postID ? { ...item, ...next } : item)),
      })),
    }
  }

  queryClient.setQueryData<Post>(['post', postID], (old) => (old ? { ...old, ...next } : old))
  for (const key of [['feed'], ['userPosts'], ['saved']]) {
    queryClient.setQueriesData({ queryKey: key }, patch)
  }
}
