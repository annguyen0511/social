import { Link, useNavigate } from 'react-router-dom'
import { useInfiniteQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Loader2, LogOut } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { HttpError, get, post } from '../../api/client'
import type { FeedPost, Pagination } from '../../api/types'
import { UserSearch } from '../users/UserSearch'

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
  const navigate = useNavigate()
  const queryClient = useQueryClient()

  const query = useInfiniteQuery({
    queryKey: ['feed'],
    initialPageParam: 1,
    queryFn: ({ pageParam, signal }) =>
      get<Pagination<FeedPost>>(`/v1/users/feed?page=${pageParam}&page_size=20`, signal),
    getNextPageParam: (last) => (last.page < last.total_pages ? last.page + 1 : undefined),
  })

  const logout = useMutation({
    mutationFn: () => post<null>('/v1/authentication/logout'),
    onSuccess: () => {
      // The cache lives in memory and does not follow the session cookie out.
      // Without this, whoever signs in next on this machine sees the previous
      // person's feed for the moment before the new requests land.
      //
      // Cache nằm trong RAM và không mất đi theo cookie phiên. Thiếu dòng này,
      // người đăng nhập tiếp theo trên cùng máy sẽ thấy feed của người trước
      // trong khoảnh khắc trước khi các request mới về.
      queryClient.clear()
      navigate('/login')
    },
  })

  // A 401 means the session is gone: expired, logged out in another tab, or
  // the account was removed. Send the reader to the login page.
  //
  // 401 nghĩa là phiên không còn: hết hạn, đã đăng xuất ở tab khác, hoặc tài
  // khoản bị xoá. Đưa người dùng về trang đăng nhập.
  if (query.error instanceof HttpError && query.error.status === 401) {
    return (
      <main className="grid min-h-dvh place-items-center p-6">
        <div className="text-center">
          <p className="mb-4 text-muted-foreground">Phiên đăng nhập đã hết.</p>
          <Button asChild>
            <Link to="/login">Đăng nhập lại</Link>
          </Button>
        </div>
      </main>
    )
  }

  const posts = query.data?.pages.flatMap((p) => p.items) ?? []

  return (
    <main className="mx-auto max-w-2xl p-6">
      <header className="mb-6 space-y-4">
        <div className="flex items-center justify-between gap-4">
          <h1 className="text-xl font-semibold">Bảng tin</h1>
          <Button variant="ghost" size="sm" onClick={() => logout.mutate()}>
            <LogOut />
            Đăng xuất
          </Button>
        </div>
        <UserSearch />
      </header>

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
