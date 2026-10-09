import { useInfiniteQuery } from '@tanstack/react-query'
import { Loader2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { get } from '../../api/client'
import type { FeedPost, Pagination } from '../../api/types'
import { PostCard } from './PostCard'

/**
 * A paged list of posts, used for both the feed and a profile.
 *
 * The two differ only in which URL they read and which cache key they live
 * under, so they share everything else: the skeleton, the empty state and the
 * "load more" button.
 *
 * Danh sách bài có phân trang, dùng cho cả bảng tin lẫn trang cá nhân.
 *
 * Hai nơi chỉ khác nhau ở URL đọc và key cache, nên dùng chung mọi thứ còn
 * lại: khung chờ, trạng thái rỗng và nút tải thêm.
 */
export function PostList({
  queryKey,
  path,
  empty,
}: {
  queryKey: unknown[]
  path: string
  empty: string
}) {
  const query = useInfiniteQuery({
    queryKey,
    initialPageParam: 1,
    queryFn: ({ pageParam, signal }) =>
      get<Pagination<FeedPost>>(`${path}?page=${pageParam}&page_size=20`, signal),
    getNextPageParam: (last) => (last.page < last.total_pages ? last.page + 1 : undefined),
  })

  if (query.isPending) {
    return (
      <div className="space-y-4">
        {[0, 1, 2].map((i) => (
          <Card key={i}>
            <CardContent className="space-y-2">
              <Skeleton className="h-8 w-40" />
              <Skeleton className="h-5 w-2/3" />
              <Skeleton className="h-4 w-full" />
            </CardContent>
          </Card>
        ))}
      </div>
    )
  }

  if (query.isError) {
    return <p className="text-sm text-destructive">Không tải được danh sách bài.</p>
  }

  const posts = query.data.pages.flatMap((page) => page.items)

  if (posts.length === 0) {
    return <p className="text-muted-foreground">{empty}</p>
  }

  return (
    <>
      <ul className="space-y-4">
        {posts.map((post) => (
          <li key={post.id}>
            <PostCard post={post} />
          </li>
        ))}
      </ul>

      {query.hasNextPage && (
        <Button
          variant="outline"
          onClick={() => query.fetchNextPage()}
          disabled={query.isFetchingNextPage}
          className="mt-6 w-full"
        >
          {query.isFetchingNextPage && <Loader2 className="animate-spin" />}
          {query.isFetchingNextPage ? 'Đang tải' : 'Tải thêm'}
        </Button>
      )}
    </>
  )
}
