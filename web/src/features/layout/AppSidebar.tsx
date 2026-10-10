import type { ComponentType } from 'react'
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

const row = 'flex h-11 w-full items-center gap-4 rounded-lg px-2.5 transition-colors'

// The label stays in the document at all times and only fades, rather than
// being added on hover. Two reasons: a screen reader can read it whether or
// not a pointer is anywhere near, and there is nothing to lay out when it
// appears, so the icons do not twitch as the rail opens.
//
// Nhãn luôn nằm trong tài liệu và chỉ mờ đi, chứ không phải tới lúc hover mới
// được thêm vào. Hai lý do: trình đọc màn hình đọc được nó bất kể con trỏ có
// ở gần hay không, và khi nó hiện ra thì không có gì phải dàn lại, nên các
// icon không bị giật lúc thanh mở.
const labelText =
  'truncate text-sm whitespace-nowrap opacity-0 transition-opacity duration-200 group-hover/rail:opacity-100 group-focus-within/rail:opacity-100'

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
    // The rail widens on hover, and on focus-within as well so that reaching
    // it with Tab opens it too — otherwise the names would be reachable only
    // with a pointer.
    //
    // It is position: fixed and the page reserves only the collapsed width,
    // so opening lays the rail over the content instead of pushing it. Making
    // room for the open width would reflow the whole page every time the
    // pointer crossed the edge of the screen.
    //
    // Thanh nở ra khi rê chuột, và cả khi focus-within, để đi bằng Tab cũng
    // mở được — nếu không thì chỉ có chuột mới đọc được tên các mục.
    //
    // Nó dùng position: fixed và trang chỉ chừa đúng bề rộng lúc thu gọn, nên
    // khi mở thanh đè lên nội dung chứ không đẩy nội dung đi. Chừa sẵn chỗ
    // cho bề rộng lúc mở sẽ khiến cả trang dàn lại mỗi lần con trỏ đi ngang
    // qua mép màn hình.
    <aside
      className={cn(
        'group/rail fixed inset-y-0 left-0 z-30 flex w-16 flex-col gap-1 overflow-hidden',
        'border-r border-border bg-background px-2 py-4',
        'transition-[width,box-shadow] duration-200 hover:w-60 hover:shadow-lg',
        'focus-within:w-60 focus-within:shadow-lg',
      )}
    >
      <Link to="/" className={cn(row, 'mb-3')}>
        <span className="grid size-6 shrink-0 place-items-center rounded-md bg-foreground text-xs font-semibold text-background">
          S
        </span>
        <span className={cn(labelText, 'font-semibold')}>Social</span>
      </Link>

      <nav className="flex flex-1 flex-col gap-1">
        {items.map(({ label, icon: Icon, to }) =>
          to ? (
            <NavLink
              key={label}
              to={to}
              end
              className={({ isActive }) => cn(row, 'hover:bg-muted', isActive && 'bg-muted font-semibold')}
            >
              <Icon className="size-6 shrink-0" />
              <span className={labelText}>{label}</span>
            </NavLink>
          ) : (
            <button
              key={label}
              type="button"
              disabled
              title={`${label} — sắp có`}
              className={cn(row, 'cursor-not-allowed text-muted-foreground opacity-50')}
            >
              <Icon className="size-6 shrink-0" />
              <span className={labelText}>{label}</span>
            </button>
          ),
        )}

        {/* "Tạo" mở hộp thoại chứ không chuyển trang, nên nó không nằm trong
            mảng items vốn chỉ mô tả các đích điều hướng.

            "Tạo" opens a dialog instead of navigating, so it is not in the
            items array, which only describes navigation targets. */}
        <button type="button" onClick={openComposer} className={cn(row, 'hover:bg-muted')}>
          <SquarePlus className="size-6 shrink-0" />
          <span className={labelText}>Tạo bài viết</span>
        </button>
      </nav>

      <div className="flex flex-col gap-1">
        {me ? (
          <NavLink
            to={`/users/${me.id}`}
            className={({ isActive }) => cn(row, 'hover:bg-muted', isActive && 'bg-muted font-semibold')}
          >
            <UserAvatar user={me} className="size-6 shrink-0 text-[10px]" />
            <span className={labelText}>Trang cá nhân</span>
          </NavLink>
        ) : (
          <div className={row}>
            <Skeleton className="size-6 shrink-0 rounded-full" />
            <Skeleton className={cn(labelText, 'h-4 w-24')} />
          </div>
        )}

        <button type="button" onClick={() => openSettings()} className={cn(row, 'hover:bg-muted')}>
          <Settings className="size-6 shrink-0" />
          <span className={labelText}>Cài đặt</span>
        </button>

        <button
          type="button"
          onClick={onLogout}
          disabled={loggingOut}
          className={cn(row, 'hover:bg-muted disabled:opacity-40')}
        >
          <LogOut className="size-6 shrink-0" />
          <span className={labelText}>Đăng xuất</span>
        </button>
      </div>
    </aside>
  )
}
