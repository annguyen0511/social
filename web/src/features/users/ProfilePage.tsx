import { useState } from 'react'
import { Link, Navigate, useParams } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { ArrowLeft, Ban, Loader2, Pencil, ShieldOff, Star, StarOff } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { HttpError, del, get, isUnauthorized, put } from '../../api/client'
import type { FriendshipStatus, User } from '../../api/types'
import { PostList } from '../posts/PostList'
import { EditProfileDialog } from './EditProfileDialog'
import { FollowButton } from './FollowButton'
import { UserAvatar } from './UserAvatar'

/**
 * Toggles one relationship — a block, or a close friend entry — by sending PUT
 * to add it and DELETE to remove it.
 *
 * Bật tắt một quan hệ — chặn, hoặc bạn thân — bằng cách gửi PUT để thêm và
 * DELETE để bỏ.
 */
function RelationButton({
  userID,
  path,
  active,
  addLabel,
  removeLabel,
  addIcon,
  removeIcon,
  destructive = false,
}: {
  userID: number
  path: 'block' | 'close-friend'
  active: boolean
  addLabel: string
  removeLabel: string
  addIcon: React.ReactNode
  removeIcon: React.ReactNode
  destructive?: boolean
}) {
  const queryClient = useQueryClient()

  const mutation = useMutation({
    mutationFn: () =>
      active
        ? del<null>(`/v1/friend-ship/${userID}/${path}`)
        : put<null>(`/v1/friend-ship/${userID}/${path}`),
    onSuccess: () => {
      // A block does more than record itself: the API removes follows and
      // close friend entries in both directions in the same transaction, and
      // a blocked user drops out of search. So the whole relationship is
      // stale, not just the one flag this button owns.
      //
      // Chặn không chỉ ghi lại việc chặn: API xoá luôn follow và bạn thân ở cả
      // hai chiều trong cùng một transaction, và người bị chặn biến khỏi kết
      // quả tìm kiếm. Nên toàn bộ quan hệ đã cũ, không riêng cái cờ mà nút này
      // phụ trách.
      queryClient.invalidateQueries({ queryKey: ['friendship', userID] })
      queryClient.invalidateQueries({ queryKey: ['feed'] })
      queryClient.invalidateQueries({ queryKey: ['users', 'search'] })
    },
  })

  return (
    <Button
      type="button"
      size="sm"
      variant={destructive && !active ? 'destructive' : 'outline'}
      onClick={() => mutation.mutate()}
      disabled={mutation.isPending}
    >
      {mutation.isPending ? <Loader2 className="animate-spin" /> : active ? removeIcon : addIcon}
      {mutation.isPending ? 'Đang lưu' : active ? removeLabel : addLabel}
    </Button>
  )
}

export function ProfilePage() {
  const { userID = '' } = useParams()
  const id = Number(userID)
  const [editing, setEditing] = useState(false)

  const user = useQuery({
    queryKey: ['users', userID],
    queryFn: ({ signal }) => get<User>(`/v1/users/${userID}`, signal),
  })

  // The client has no other way to know its own ID, and it needs it: the
  // friendship endpoints answer 400 for an action aimed at yourself, so the
  // buttons must not be drawn on your own profile in the first place.
  //
  // Client không có cách nào khác để biết ID của chính mình, mà nó cần biết:
  // các endpoint friendship trả về 400 cho hành động nhắm vào chính mình, nên
  // ngay từ đầu đã không được vẽ các nút đó trên profile của bản thân.
  const me = useQuery({
    queryKey: ['users', 'me'],
    queryFn: ({ signal }) => get<User>('/v1/users/me', signal),
  })

  const isSelf = me.data?.id === id

  const friendship = useQuery({
    queryKey: ['friendship', id],
    queryFn: ({ signal }) => get<FriendshipStatus>(`/v1/friend-ship/${userID}`, signal),
    enabled: me.isSuccess && !isSelf && Number.isFinite(id),
  })

  // Layout đã lo việc đưa về trang đăng nhập, nhưng phiên có thể chết giữa
  // lần kiểm của nó và các request ở đây.
  //
  // The layout already handles sending the reader to login, but the session
  // can die between its check and the requests on this page.
  if (isUnauthorized(user.error) || isUnauthorized(me.error)) {
    return <Navigate to="/login" replace />
  }

  if (user.isPending) {
    return (
      <main className="mx-auto max-w-2xl p-6">
        <div className="flex items-start gap-4 rounded-xl border border-border p-6">
          <Skeleton className="size-18 rounded-full" />
          <div className="flex-1 space-y-2">
            <Skeleton className="h-6 w-40" />
            <Skeleton className="h-4 w-24" />
          </div>
        </div>
      </main>
    )
  }

  if (user.isError) {
    const notFound = user.error instanceof HttpError && user.error.status === 404
    return (
      <main className="mx-auto max-w-2xl p-6">
        <p className="mb-4 text-muted-foreground">
          {notFound ? 'Không có người dùng này.' : 'Không tải được trang.'}
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

  return (
    <main className="mx-auto max-w-2xl p-6">
      <header className="flex items-start gap-4 rounded-xl border border-border p-6">
        <UserAvatar user={user.data} className="size-18 text-lg" />

        <div className="min-w-0 flex-1">
          <h1 className="truncate text-xl font-semibold">
            {user.data.first_name} {user.data.last_name}
          </h1>
          <p className="truncate text-muted-foreground">@{user.data.username}</p>

          {isSelf && (
            <div className="mt-4">
              <Button type="button" variant="outline" size="sm" onClick={() => setEditing(true)}>
                <Pencil />
                Chỉnh sửa
              </Button>
              <EditProfileDialog user={user.data} open={editing} onOpenChange={setEditing} />
            </div>
          )}

          {/* Chỉ vẽ các nút khi đã biết quan hệ. Vẽ sớm hơn thì nút sẽ hiện
              trạng thái đoán, rồi tự nhảy sang trạng thái thật.

              The buttons only appear once the relationship is known. Drawing
              them sooner would show a guessed state that then jumps to the
              real one. */}
          {!isSelf && friendship.isSuccess && (
            <div className="mt-4 flex flex-wrap gap-2">
              <FollowButton userID={id} isFollowing={friendship.data.is_following} />

              <RelationButton
                userID={id}
                path="close-friend"
                active={friendship.data.is_close_friend}
                addLabel="Thêm bạn thân"
                removeLabel="Bỏ bạn thân"
                addIcon={<Star />}
                removeIcon={<StarOff />}
              />

              <RelationButton
                userID={id}
                path="block"
                active={friendship.data.is_blocking}
                addLabel="Chặn"
                removeLabel="Bỏ chặn"
                addIcon={<Ban />}
                removeIcon={<ShieldOff />}
                destructive
              />
            </div>
          )}

          {friendship.data?.is_blocking && (
            <p className="mt-3 text-sm text-muted-foreground">
              Bạn đang chặn người này. Bỏ chặn không khôi phục lại việc theo dõi trước đó.
            </p>
          )}
        </div>
      </header>

      <section className="mt-6">
        <h2 className="mb-3 font-semibold">Bài viết</h2>
        {/* queryKey mang userID để hai trang cá nhân khác nhau không dùng
            chung cache, nhưng vẫn bắt đầu bằng 'userPosts' để một bài mới
            đăng làm mới được mọi danh sách bằng một lần invalidate theo tiền
            tố.

            The queryKey carries the userID so two profiles do not share a
            cache, while still starting with 'userPosts' so a new post can
            refresh every list with one prefix invalidation. */}
        <PostList
          queryKey={['userPosts', userID]}
          path={`/v1/users/${userID}/posts`}
          empty={isSelf ? 'Bạn chưa đăng bài nào.' : 'Người này chưa đăng bài nào.'}
        />
      </section>
    </main>
  )
}
