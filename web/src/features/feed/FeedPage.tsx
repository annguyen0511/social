import { Link, useNavigate } from 'react-router-dom'
import { useInfiniteQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { HttpError, get, post } from '../../api/client'
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
      <main className="min-h-dvh grid place-items-center bg-stone-100 p-6">
        <div className="text-center">
          <p className="mb-4 text-stone-600">Phiên đăng nhập đã hết.</p>
          <Link to="/login" className="rounded-lg bg-stone-900 px-5 py-2.5 font-medium text-white">
            Đăng nhập lại
          </Link>
        </div>
      </main>
    )
  }

  const posts = query.data?.pages.flatMap((p) => p.items) ?? []

  return (
    <main className="mx-auto max-w-2xl p-6">
      <header className="mb-6 flex items-center justify-between">
        <h1 className="text-xl font-semibold">Bảng tin</h1>
        <button onClick={() => logout.mutate()} className="text-sm text-stone-500 underline">
          Đăng xuất
        </button>
      </header>

      {query.isPending && <p className="text-stone-500">Đang tải…</p>}

      {query.isSuccess && posts.length === 0 && (
        <p className="text-stone-500">Chưa có bài nào. Hãy theo dõi vài người để bảng tin có nội dung.</p>
      )}

      <ul className="space-y-4">
        {posts.map((post) => (
          <li key={post.id} className="rounded-xl bg-white p-5 shadow-sm">
            <p className="mb-1 text-sm text-stone-500">@{post.user.username}</p>
            <h2 className="font-semibold">{post.title}</h2>
            <p className="mt-1 text-stone-700">{post.content}</p>
            <p className="mt-3 text-sm text-stone-500">
              {post.comment_count} bình luận · {post.like_count} thích
            </p>
          </li>
        ))}
      </ul>

      {query.hasNextPage && (
        <button
          onClick={() => query.fetchNextPage()}
          disabled={query.isFetchingNextPage}
          className="mt-6 w-full rounded-lg border border-stone-300 py-2.5 disabled:opacity-40"
        >
          {query.isFetchingNextPage ? 'Đang tải…' : 'Tải thêm'}
        </button>
      )}
    </main>
  )
}
