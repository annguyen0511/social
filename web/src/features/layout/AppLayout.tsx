import { Navigate, Outlet, useNavigate } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { get, isUnauthorized, post } from '../../api/client'
import type { User } from '../../api/types'
import { ComposerProvider } from '../posts/ComposerProvider'
import { UserSearch } from '../users/UserSearch'
import { AppSidebar } from './AppSidebar'

/**
 * The frame every signed-in page sits in: a navigation rail down the left, the
 * search box across the top, and the page itself in the middle.
 *
 * It lives in a layout route rather than in each page, so the rail and the
 * search box mount once and survive navigation. Repeating them per page would
 * remount the search box on every route change, throwing away whatever had
 * been typed.
 *
 * This is also the single place that notices a dead session. Every endpoint
 * behind requireAuth answers 401 together, so checking the one query the frame
 * already needs is enough, and no page has to carry its own "you are logged
 * out" screen.
 *
 * Khung chứa mọi trang của người đã đăng nhập: thanh điều hướng dọc bên trái,
 * ô tìm kiếm trên cùng, và bản thân trang nằm ở giữa.
 *
 * Nó nằm trong một layout route chứ không nằm trong từng trang, nên thanh bên
 * và ô tìm kiếm chỉ mount một lần và sống qua các lần chuyển trang. Lặp lại ở
 * từng trang sẽ mount lại ô tìm kiếm mỗi lần đổi route, làm mất những gì đang
 * gõ.
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
    <ComposerProvider>
      <div className="min-h-dvh">
        <AppSidebar me={me.data} onLogout={() => logout.mutate()} loggingOut={logout.isPending} />

        {/* Chừa đúng bề rộng của thanh bên. Thanh bên dùng position: fixed nên
            nó không tự đẩy nội dung sang.

            Leaves exactly the rail's width. The rail is position: fixed, so it
            does not push the content across on its own. */}
        <div className="pl-16 lg:pl-60">
          <header className="sticky top-0 z-20 border-b border-border bg-background/80 backdrop-blur">
            <div className="mx-auto max-w-2xl px-6 py-3">
              <UserSearch />
            </div>
          </header>

          <Outlet />
        </div>
      </div>
    </ComposerProvider>
  )
}
