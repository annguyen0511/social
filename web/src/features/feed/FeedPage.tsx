import { Link, Navigate } from 'react-router-dom'
import { useInfiniteQuery } from '@tanstack/react-query'
import { Loader2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { get, isUnauthorized } from '../../api/client'
import type { FeedPost, Pagination } from '../../api/types'

/**
 * The feed, paged with useInfiniteQuery.
 *
 * The API pages with LIMIT/OFFSET, so a post published while the reader is
 * scrolling shifts everything down a slot and the next page repeats a row.
 * Switching the API to cursor pagination later only changes getNextPageParam.
 *
 * Feed, phân trang bằng useInfiniteQuery.
 *
 * API phân trang bằng LIMIT/OFFSET, nên nếu có bài mới đăng trong lúc người
 * dùng đang cuộn thì mọi thứ bị đẩy xuống một bậc và trang sau lặp lại một
 * dòng. Sau này API đổi sang cursor thì chỉ phải sửa getNextPageParam.
 */
export function FeedPage() {
  const query = useInfiniteQuery({
    queryKey: ['feed'],
    initialPageParam: 1,
    queryFn: ({ pageParam, signal }) =>
      get<Pagination<FeedPost>>(`/v1/users/feed?page=${pageParam}&page_size=20`, signal),
    getNextPageParam: (last) => (last.page < last.total_pages ? last.page + 1 : undefined),
  })

  // The layout watches the same thing, but a session can die between its check
  // and this request, and a 401 here must not be rendered as a broken feed.
  //
  // Layout cũng theo dõi điều này, nhưng phiên có thể chết trong khoảng giữa
  // lần kiểm của nó và request này, và 401 ở đây không được hiện ra thành một
  // feed hỏng.
  if (isUnauthorized(query.error)) {
    return <Navigate to="/login" replace />
  }

  const posts = query.data?.pages.flatMap((p) => p.items) ?? []

  return (
    <main className="mx-auto max-w-2xl p-6">
      <h1 className="mb-6 text-xl font-semibold">Bảng tin</h1>

      {query.isPending && (
        <div className="space-y-4">
          {[0, 1, 2].map((i) => (
            <Card key={i}>
              <CardContent className="space-y-2">
                <Skeleton className="h-4 w-24" />
                <Skeleton className="h-5 w-2/3" />
                <Skeleton className="h-4 w-full" />
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      {query.isSuccess && posts.length === 0 && (
        <p className="text-muted-foreground">
          Chưa có bài nào. Dùng ô tìm kiếm ở trên để theo dõi vài người.
        </p>
      )}

      <ul className="space-y-4">
        {posts.map((post) => (
          <li key={post.id}>
            <Card>
              <CardContent>
                <Link
                  to={`/users/${post.user.id}`}
                  className="mb-1 block text-sm text-muted-foreground hover:underline"
                >
                  @{post.user.username}
                </Link>
                <h2 className="font-semibold">{post.title}</h2>
                <p className="mt-1 text-foreground/80">{post.content}</p>
                <p className="mt-3 text-sm text-muted-foreground">
                  {post.comment_count} bình luận · {post.like_count} thích
                </p>
              </CardContent>
            </Card>
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
    </main>
  )
}
