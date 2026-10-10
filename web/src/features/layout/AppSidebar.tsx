import type { ComponentType, ReactNode } from 'react'
import { Link, NavLink } from 'react-router-dom'
import {
  Compass,
  Heart,
  House,
  LogOut,
  MessageCircle,
  Settings,
  SquarePlus,
} from 'lucide-react'
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip'
import { Skeleton } from '@/components/ui/skeleton'
import { cn } from 'cn'
import type { User } from '../../api/types'
import { useComposer } from '../posts/ComposerProvider'
import { useSettings } from '../settings/SettingsProvider'
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

const row =
  'flex size-11 items-center justify-center rounded-lg transition-colors'

/**
 * Wraps a rail item so its name appears on hover and on keyboard focus.
 *
 * The rail is icons only, which keeps it narrow but leaves every item unnamed
 * until you already know what it does. A tooltip gives the name back without
 * widening the rail or shifting the page when the pointer moves across it.
 *
 * Bọc một mục trên thanh để tên của nó hiện ra khi rê chuột và khi focus bằng
 * bàn phím.
 *
 * Thanh bên chỉ có icon, nhờ vậy nó hẹp, nhưng đổi lại mọi mục đều không có
 * tên cho tới khi bạn vốn đã biết nó làm gì. Tooltip trả lại cái tên đó mà
 * không làm thanh rộng ra hay làm trang nhảy khi con trỏ đi ngang qua.
 */
function RailItem({ label, children }: { label: string; children: ReactNode }) {
  return (
    <Tooltip>
      <TooltipTrigger asChild>{children}</TooltipTrigger>
      <TooltipContent side="right">{label}</TooltipContent>
    </Tooltip>
  )
}

export function AppSidebar({
  me,
  onLogout,
  loggingOut,
}: {
  me?: User
  onLogout: () => void
  loggingOut: boolean
}) {
  const openComposer = useComposer()
  const openSettings = useSettings()

  return (
    // delayDuration 0 vì thanh này chỉ có icon: chờ nửa giây mới hiện tên thì
    // người chưa quen phải rê từng mục và đợi ở từng mục một.
    //
    // delayDuration 0 because the rail is icons only: waiting half a second
    // for a name means anyone new has to hover each item and wait at each one.
    <TooltipProvider delayDuration={0}>
      <aside className="fixed inset-y-0 left-0 z-30 flex w-16 flex-col items-center gap-1 border-r border-border bg-background py-4">
        <Link to="/" className={cn(row, 'mb-3')} aria-label="Social">
          <span className="grid size-7 place-items-center rounded-md bg-foreground text-xs font-semibold text-background">
            S
          </span>
        </Link>

        <nav className="flex flex-1 flex-col items-center gap-1">
          {items.map(({ label, icon: Icon, to }) =>
            to ? (
              <RailItem key={label} label={label}>
                <NavLink
                  to={to}
                  end
                  aria-label={label}
                  className={({ isActive }) =>
                    cn(row, 'hover:bg-muted', isActive && 'bg-muted')
                  }
                >
                  <Icon className="size-6" />
                </NavLink>
              </RailItem>
            ) : (
              <RailItem key={label} label={`${label} — sắp có`}>
                <button
                  type="button"
                  disabled
                  aria-label={label}
                  className={cn(row, 'cursor-not-allowed text-muted-foreground opacity-50')}
                >
                  <Icon className="size-6" />
                </button>
              </RailItem>
            ),
          )}

          {/* "Tạo" mở hộp thoại chứ không chuyển trang, nên nó không nằm trong
              mảng items vốn chỉ mô tả các đích điều hướng.

              "Tạo" opens a dialog instead of navigating, so it is not in the
              items array, which only describes navigation targets. */}
          <RailItem label="Tạo bài viết">
            <button
              type="button"
              aria-label="Tạo bài viết"
              onClick={openComposer}
              className={cn(row, 'hover:bg-muted')}
            >
              <SquarePlus className="size-6" />
            </button>
          </RailItem>
        </nav>

        <div className="flex flex-col items-center gap-1">
          {me ? (
            <RailItem label={`Trang cá nhân (@${me.username})`}>
              <NavLink
                to={`/users/${me.id}`}
                aria-label="Trang cá nhân"
                className={({ isActive }) => cn(row, 'hover:bg-muted', isActive && 'bg-muted')}
              >
                <UserAvatar user={me} className="size-7 text-[10px]" />
              </NavLink>
            </RailItem>
          ) : (
            <div className={row}>
              <Skeleton className="size-7 rounded-full" />
            </div>
          )}

          <RailItem label="Cài đặt">
            <button
              type="button"
              aria-label="Cài đặt"
              onClick={() => openSettings()}
              className={cn(row, 'hover:bg-muted')}
            >
              <Settings className="size-6" />
            </button>
          </RailItem>

          <RailItem label="Đăng xuất">
            <button
              type="button"
              aria-label="Đăng xuất"
              onClick={onLogout}
              disabled={loggingOut}
              className={cn(row, 'hover:bg-muted disabled:opacity-40')}
            >
              <LogOut className="size-6" />
            </button>
          </RailItem>
        </div>
      </aside>
    </TooltipProvider>
  )
}
