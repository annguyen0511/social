import { useEffect, useState, useSyncExternalStore } from 'react'
import { Check, Monitor, Moon, Sun } from 'lucide-react'
import { cn } from 'cn'
import {
  applyTheme,
  prefersDark,
  readTheme,
  storeTheme,
  subscribeToSystemTheme,
  type Theme,
} from '../../lib/theme'

const options: { value: Theme; label: string; hint: string; icon: typeof Sun }[] = [
  { value: 'light', label: 'Sáng', hint: 'Luôn dùng giao diện sáng', icon: Sun },
  { value: 'dark', label: 'Tối', hint: 'Luôn dùng giao diện tối', icon: Moon },
  {
    value: 'system',
    label: 'Theo hệ thống',
    hint: 'Đổi theo cài đặt của máy bạn',
    icon: Monitor,
  },
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
export function ThemeSettings() {
  const [theme, setTheme] = useState<Theme>(readTheme)

  // The third argument is the value when there is no window at all. It keeps
  // this component renderable outside a browser, which is what a render test
  // does.
  //
  // Tham số thứ ba là giá trị khi hoàn toàn không có window. Nó giữ cho
  // component này render được bên ngoài trình duyệt, đúng kiểu một bài kiểm
  // tra render làm.
  const systemDark = useSyncExternalStore(subscribeToSystemTheme, prefersDark, () => false)

  // systemDark is a dependency so that following the system re-applies the
  // class when the system changes, with no separate listener to keep in sync.
  //
  // systemDark là một phụ thuộc, nhờ vậy chế độ theo hệ thống tự áp lại class
  // khi hệ thống đổi, mà không cần một listener riêng phải giữ cho khớp.
  useEffect(() => {
    applyTheme(theme)
  }, [theme, systemDark])

  return (
    <section className="space-y-4">
      <div>
        <h2 className="font-semibold">Giao diện</h2>
        <p className="text-sm text-muted-foreground">
          Lựa chọn được nhớ trên trình duyệt này.
        </p>
      </div>

      <ul className="space-y-1">
        {options.map(({ value, label, hint, icon: Icon }) => (
          <li key={value}>
            <button
              type="button"
              onClick={() => {
                setTheme(value)
                storeTheme(value)
              }}
              aria-pressed={theme === value}
              className={cn(
                'flex w-full items-center gap-3 rounded-lg px-3 py-2.5 text-left transition-colors',
                theme === value ? 'bg-muted' : 'hover:bg-muted',
              )}
            >
              <Icon className="size-5 shrink-0" />
              <span className="min-w-0 flex-1">
                <span className="block text-sm font-medium">{label}</span>
                <span className="block text-sm text-muted-foreground">{hint}</span>
              </span>
              {theme === value && <Check className="size-4 shrink-0" />}
            </button>
          </li>
        ))}
      </ul>
    </section>
  )
}
