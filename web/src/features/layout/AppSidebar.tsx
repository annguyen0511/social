import { useState, type ComponentType } from 'react'
import { Link, NavLink } from 'react-router-dom'
import { Compass, Heart, House, LogOut, MessageCircle, SquarePlus } from 'lucide-react'
import { Skeleton } from '@/components/ui/skeleton'
import { cn } from 'cn'
import type { User } from '../../api/types'
import { CreatePostDialog } from '../posts/CreatePostDialog'
import { UserAvatar } from '../users/UserAvatar'

type NavItem = {
  label: string
  icon: ComponentType<{ className?: string }>
  // A missing `to` is a section that does not exist yet. It is rendered
  // disabled rather than left out, so the shape of the app is visible and
  // adding the page later is one line here plus one route.
  //
  // Thiếu `to` nghĩa là phần đó chưa tồn tại. Nó được vẽ ở trạng thái vô hiệu
  // chứ không bị bỏ đi, để hình dáng của ứng dụng nhìn thấy được, và sau này
  // thêm trang chỉ là một dòng ở đây cộng một route.
  to?: string
}

const items: NavItem[] = [
  { label: 'Trang chủ', icon: House, to: '/' },
  { label: 'Khám phá', icon: Compass },
  { label: 'Tin nhắn', icon: MessageCircle },
  { label: 'Thông báo', icon: Heart },
]

// The label is hidden rather than the whole row on a narrow screen, so the rail
// stays usable as icons instead of disappearing and stranding the reader with
// no way to navigate.
//
// Trên màn hình hẹp thì chỉ ẩn nhãn chứ không ẩn cả dòng, để thanh bên vẫn
// dùng được dưới dạng icon thay vì biến mất và bỏ người dùng lại không còn
// đường nào để chuyển trang.
const row =
  'flex items-center gap-4 rounded-lg px-3 py-2.5 text-sm transition-colors lg:w-full'
const label = 'hidden truncate lg:inline'

export function AppSidebar({
  me,
  onLogout,
  loggingOut,
}: {
  me?: User
  onLogout: () => void
  loggingOut: boolean
}) {
  const [composing, setComposing] = useState(false)

  return (
    <aside className="fixed inset-y-0 left-0 z-30 flex w-16 flex-col border-r border-border bg-background px-2 py-4 lg:w-60 lg:px-3">
      <Link to="/" className={cn(row, 'mb-4 font-semibold')} aria-label="Social">
        <span className="grid size-6 shrink-0 place-items-center rounded-md bg-foreground text-xs text-background">
          S
        </span>
        <span className={label}>Social</span>
      </Link>

      <nav className="flex flex-1 flex-col gap-1">
        {items.map(({ label: text, icon: Icon, to }) =>
          to ? (
            <NavLink
              key={text}
              to={to}
              end
              className={({ isActive }) =>
                cn(row, 'hover:bg-muted', isActive && 'font-semibold')
              }
            >
              <Icon className="size-6 shrink-0" />
              <span className={label}>{text}</span>
            </NavLink>
          ) : (
            <button
              key={text}
              type="button"
              disabled
              title={`${text} — sắp có`}
              className={cn(row, 'cursor-not-allowed text-muted-foreground opacity-50')}
            >
              <Icon className="size-6 shrink-0" />
              <span className={label}>{text}</span>
            </button>
          ),
        )}
        {/* "Tạo" mở hộp thoại chứ không chuyển trang, nên nó không nằm trong
            mảng items vốn chỉ mô tả các đích điều hướng.

            "Tạo" opens a dialog instead of navigating, so it is not in the
            items array, which only describes navigation targets. */}
        <button type="button" onClick={() => setComposing(true)} className={cn(row, 'hover:bg-muted')}>
          <SquarePlus className="size-6 shrink-0" />
          <span className={label}>Tạo</span>
        </button>
      </nav>

      <CreatePostDialog open={composing} onOpenChange={setComposing} />

      <div className="flex flex-col gap-1">
        {/* Avatar của chính mình, dẫn tới trang cá nhân. Chưa tải xong thì giữ
            chỗ bằng khối xám để thanh bên không bị nhảy.

            The reader's own avatar, leading to their profile. A grey block
            holds the space while it loads so the rail does not jump. */}
        {me ? (
          <NavLink
            to={`/users/${me.id}`}
            className={({ isActive }) =>
              cn(row, 'hover:bg-muted', isActive && 'font-semibold')
            }
            title={`@${me.username}`}
          >
            <UserAvatar user={me} className="size-6 shrink-0 text-[10px]" />
            <span className={label}>Trang cá nhân</span>
          </NavLink>
        ) : (
          <div className={row}>
            <Skeleton className="size-6 shrink-0 rounded-full" />
            <Skeleton className={cn(label, 'h-4 w-24')} />
          </div>
        )}

        <button
          type="button"
          onClick={onLogout}
          disabled={loggingOut}
          className={cn(row, 'hover:bg-muted disabled:opacity-40')}
        >
          <LogOut className="size-6 shrink-0" />
          <span className={label}>Đăng xuất</span>
        </button>
      </div>
    </aside>
  )
}
