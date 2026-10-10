import { useEffect, useState } from 'react'
import { Check, Monitor, Moon, Sun } from 'lucide-react'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { cn } from 'cn'
import { applyTheme, prefersDark, readTheme, storeTheme, type Theme } from '../../lib/theme'

const options: { value: Theme; label: string; icon: typeof Sun }[] = [
  { value: 'light', label: 'Sáng', icon: Sun },
  { value: 'dark', label: 'Tối', icon: Moon },
  { value: 'system', label: 'Theo hệ thống', icon: Monitor },
]

/**
 * Switches between the light and dark themes.
 *
 * "Theo hệ thống" is the default and a real third choice, not the absence of
 * one: picking it back after trying dark is otherwise impossible, and it is
 * what makes the app follow a phone that darkens itself in the evening.
 *
 * Chuyển giữa giao diện sáng và tối.
 *
 * "Theo hệ thống" là mặc định và là một lựa chọn thật sự, không phải là việc
 * không chọn gì: thiếu nó thì sau khi thử giao diện tối sẽ không có đường
 * quay lại, và chính nó khiến ứng dụng đi theo một chiếc điện thoại tự chuyển
 * tối vào buổi tối.
 */
export function ThemeToggle({
  className,
  labelClassName,
}: {
  className?: string
  labelClassName?: string
}) {
  const [theme, setTheme] = useState<Theme>(readTheme)

  useEffect(() => {
    applyTheme(theme)
  }, [theme])

  // Only while following the system does a change out there mean anything
  // here. Listening unconditionally would override an explicit choice the
  // moment the operating system switched.
  //
  // Chỉ khi đang đi theo hệ thống thì một thay đổi ngoài kia mới có ý nghĩa ở
  // đây. Lắng nghe vô điều kiện sẽ ghi đè lên lựa chọn người dùng đã nêu rõ,
  // ngay khi hệ điều hành đổi.
  useEffect(() => {
    if (theme !== 'system') return

    const media = window.matchMedia('(prefers-color-scheme: dark)')
    const onChange = () => applyTheme('system')
    media.addEventListener('change', onChange)
    return () => media.removeEventListener('change', onChange)
  }, [theme])

  const choose = (next: Theme) => {
    setTheme(next)
    storeTheme(next)
  }

  // The icon shows what is on screen, not what was chosen: under "system" the
  // word tells you nothing about whether it is currently dark.
  //
  // Icon thể hiện thứ đang hiển thị chứ không phải thứ đã chọn: ở chế độ
  // "theo hệ thống", cái tên không cho biết hiện tại đang sáng hay tối.
  const showingDark = theme === 'dark' || (theme === 'system' && prefersDark())
  const Icon = showingDark ? Moon : Sun

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          type="button"
          variant="ghost"
          aria-label="Đổi giao diện sáng tối"
          className={cn('justify-start', className)}
        >
          <Icon className="size-6 shrink-0" />
          <span className={labelClassName}>Giao diện</span>
        </Button>
      </DropdownMenuTrigger>

      <DropdownMenuContent align="start">
        {options.map(({ value, label: text, icon: OptionIcon }) => (
          <DropdownMenuItem key={value} onSelect={() => choose(value)}>
            <OptionIcon />
            {text}
            {theme === value && <Check className="ml-auto" />}
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
