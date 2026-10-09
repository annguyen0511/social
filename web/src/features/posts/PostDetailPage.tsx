import { Link, Navigate, useLocation, useNavigate, useParams } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { ArrowLeft } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { HttpError, get, isUnauthorized } from '../../api/client'
import type { Post, User } from '../../api/types'
import { formatDateTime } from '../../lib/format'
import { UserAvatar } from '../users/UserAvatar'
import { CommentForm } from './CommentForm'
import { LikeButton } from './LikeButton'
import { PostActions } from './PostActions'

/**
 * Goes back where the reader came from, or home when there is nowhere to go.
 *
 * React Router gives the first entry of a session the key "default", so that
 * is how an arrival by pasted link or a fresh tab is told apart from a click
 * inside the app. Calling history.back() in the first case would leave the
 * site entirely.
 *
 * Quay lại nơi người dùng vừa rời đi, hoặc về trang chủ khi không có chỗ nào
 * để quay về.
 *
 * React Router đặt key "default" cho mục đầu tiên của một phiên, và đó là cách
 * phân biệt việc vào bằng link dán hay tab mới với việc bấm từ trong ứng dụng.
 * Gọi history.back() ở trường hợp đầu sẽ đưa người dùng ra khỏi hẳn trang.
 */
function BackButton() {
  const navigate = useNavigate()
  const location = useLocation()
  const canGoBack = location.key !== 'default'

  if (!canGoBack) {
    return (
      <Button asChild variant="ghost" size="sm" className="-ml-2 mb-4">
        <Link to="/">
          <ArrowLeft />
          Về bảng tin
        </Link>
      </Button>
    )
  }

  return (
    <Button
      type="button"
      variant="ghost"
      size="sm"
      className="-ml-2 mb-4"
      onClick={() => navigate(-1)}
    >
      <ArrowLeft />
      Quay lại
    </Button>
  )
}

export function PostDetailPage() {
  const { postID = '' } = useParams()
  const id = Number(postID)

  const post = useQuery({
    queryKey: ['post', id],
    queryFn: ({ signal }) => get<Post>(`/v1/posts/${postID}`, signal),
    enabled: Number.isFinite(id),
  })

  // Needed to decide whether the "..." menu belongs here at all. The API
  // refuses an edit or delete from anyone else with a 403, so showing the menu
  // to a visitor would only offer them two buttons that cannot work.
  //
  // Cần để biết menu "..." có nên xuất hiện ở đây hay không. API từ chối lệnh
  // sửa hay xoá của người khác bằng 403, nên hiện menu cho khách chỉ là đưa ra
  // hai cái nút chắc chắn không chạy được.
  const me = useQuery({
    queryKey: ['users', 'me'],
    queryFn: ({ signal }) => get<User>('/v1/users/me', signal),
  })

  if (isUnauthorized(post.error) || isUnauthorized(me.error)) {
    return <Navigate to="/login" replace />
  }

  if (post.isPending) {
    return (
      <main className="mx-auto max-w-2xl p-6">
        <BackButton />
        <Skeleton className="mb-3 h-8 w-40" />
        <Skeleton className="mb-3 h-6 w-2/3" />
        <Skeleton className="h-24 w-full" />
      </main>
    )
  }

  if (post.isError) {
    const notFound = post.error instanceof HttpError && post.error.status === 404
    return (
      <main className="mx-auto max-w-2xl p-6">
        <p className="mb-4 text-muted-foreground">
          {notFound ? 'Bài viết không tồn tại hoặc đã bị xoá.' : 'Không tải được bài viết.'}
        </p>
        <Button asChild variant="outline" size="sm">
          <Link to="/">
            <ArrowLeft />
            Về bảng tin
          </Link>
        </Button>
      </main>
    )
  }

  const data = post.data
  const isMine = me.data?.id === data.user_id
  const comments = data.comments ?? []

  return (
    <main className="mx-auto max-w-2xl p-6">
      <BackButton />

      <Card>
        <CardContent>
          <div className="mb-4 flex items-start gap-2">
            <Link to={`/users/${data.user.id}`} className="flex min-w-0 flex-1 items-center gap-2">
              <UserAvatar user={data.user} className="size-10" />
              <span className="min-w-0">
                <span className="block truncate text-sm font-medium hover:underline">
                  @{data.user.username}
                </span>
                <span className="block text-sm text-muted-foreground">
                  {formatDateTime(data.created_at)}
                  {data.updated_at !== data.created_at && ' · đã sửa'}
                </span>
              </span>
            </Link>

            {isMine && <PostActions post={data} />}
          </div>

          <h1 className="text-xl font-semibold">{data.title}</h1>
          <p className="mt-2 whitespace-pre-wrap text-foreground/80">{data.content}</p>

          {data.tags && data.tags.length > 0 && (
            <ul className="mt-4 flex flex-wrap gap-1.5">
              {data.tags.map((tag) => (
                <li key={tag} className="rounded-md bg-muted px-2 py-0.5 text-xs text-muted-foreground">
                  #{tag}
                </li>
              ))}
            </ul>
          )}

          <div className="mt-4 -ml-2">
            <LikeButton postID={data.id} likeCount={data.like_count} isLiked={data.is_liked} />
          </div>
        </CardContent>
      </Card>

      <section className="mt-6">
        <h2 className="mb-3 font-semibold">{comments.length} bình luận</h2>

        <CommentForm postID={data.id} />

        {comments.length === 0 ? (
          <p className="text-muted-foreground">Chưa có bình luận nào.</p>
        ) : (
          <ul className="space-y-3">
            {comments.map((comment) => (
              <li key={comment.id} className="rounded-xl border border-border p-4">
                <Link
                  to={`/users/${comment.user.id}`}
                  className="flex items-center gap-2 text-sm font-medium hover:underline"
                >
                  <UserAvatar user={comment.user} className="size-6 text-[10px]" />@
                  {comment.user.username}
                </Link>
                <p className="mt-2 whitespace-pre-wrap text-foreground/80">{comment.content}</p>
                <p className="mt-2 text-xs text-muted-foreground">
                  {formatDateTime(comment.created_at)}
                </p>
              </li>
            ))}
          </ul>
        )}
      </section>
    </main>
  )
}
