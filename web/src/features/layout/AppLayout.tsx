import { Link, Navigate, Outlet, useNavigate } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { LogOut } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { get, isUnauthorized, post } from '../../api/client'
import type { User } from '../../api/types'
import { UserAvatar } from '../users/UserAvatar'
import { UserSearch } from '../users/UserSearch'

/**
 * The frame every signed-in page sits in: one sticky bar carrying the search
 * box, the logout button and the reader's own avatar.
 *
 * It lives in a layout route rather than in each page, so the bar is mounted
 * once and survives navigation. Repeating it per page would remount the search
 * box on every route change, throwing away whatever had been typed.
 *
 * This is also the single place that notices a dead session. Every endpoint
 * behind requireAuth answers 401 together, so checking the one query the frame
 * already needs is enough, and no page has to carry its own "you are logged
 * out" screen.
 *
 * Khung chứa mọi trang của người đã đăng nhập: một thanh cố định mang ô tìm
 * kiếm, nút đăng xuất và avatar của chính người dùng.
 *
 * Nó nằm trong một layout route chứ không nằm trong từng trang, nên thanh này
 * chỉ mount một lần và sống qua các lần chuyển trang. Lặp lại ở từng trang sẽ
 * mount lại ô tìm kiếm mỗi lần đổi route, làm mất những gì đang gõ.
 *
 * Đây cũng là chỗ duy nhất phát hiện phiên đã chết. Mọi endpoint sau
 * requireAuth đều trả 401 cùng lúc, nên kiểm đúng cái query mà khung này vốn
 * đã cần là đủ, và không trang nào phải tự mang màn hình "bạn đã đăng xuất".
 */
export function AppLayout() {
  const navigate = useNavigate()
  const queryClient = useQueryClient()

  const me = useQuery({
    queryKey: ['users', 'me'],
    queryFn: ({ signal }) => get<User>('/v1/users/me', signal),
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

  if (isUnauthorized(me.error)) {
    return <Navigate to="/login" replace />
  }

  return (
    <div className="min-h-dvh">
      <header className="sticky top-0 z-20 border-b border-border bg-background/80 backdrop-blur">
        <div className="mx-auto flex max-w-2xl items-center gap-3 px-6 py-3">
          <Link to="/" className="shrink-0 font-semibold">
            Social
          </Link>

          <div className="min-w-0 flex-1">
            <UserSearch />
          </div>

          <Button
            type="button"
            variant="ghost"
            size="icon-sm"
            title="Đăng xuất"
            aria-label="Đăng xuất"
            onClick={() => logout.mutate()}
            disabled={logout.isPending}
          >
            <LogOut />
          </Button>

          {/* Avatar nằm ngoài cùng bên phải và dẫn tới trang của chính mình.
              Chưa tải xong thì giữ đúng chỗ bằng một khối xám, để thanh header
              không bị giật khi dữ liệu về.

              The avatar is the rightmost element and leads to the reader's own
              page. While it loads, a grey block holds the same space so the bar
              does not jump when the data arrives. */}
          {me.data ? (
            <Link
              to={`/users/${me.data.id}`}
              title={`@${me.data.username}`}
              aria-label="Trang cá nhân của bạn"
              className="shrink-0 rounded-full outline-offset-2 focus-visible:outline-2 focus-visible:outline-ring"
            >
              <UserAvatar user={me.data} className="size-8" />
            </Link>
          ) : (
            <Skeleton className="size-8 shrink-0 rounded-full" />
          )}
        </div>
      </header>

      <Outlet />
    </div>
  )
}
