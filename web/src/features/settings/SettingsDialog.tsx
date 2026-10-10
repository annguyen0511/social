import type { ComponentType } from 'react'
import { Ban, SunMoon, UserRound } from 'lucide-react'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { cn } from 'cn'
import { BlockedSettings } from './BlockedSettings'
import { ProfileSettings } from './ProfileSettings'
import type { SettingsSection } from './SettingsProvider'
import { ThemeSettings } from './ThemeSettings'

const sections: {
  id: SettingsSection
  label: string
  icon: ComponentType<{ className?: string }>
  Panel: ComponentType
}[] = [
  { id: 'profile', label: 'Trang cá nhân', icon: UserRound, Panel: ProfileSettings },
  { id: 'theme', label: 'Giao diện', icon: SunMoon, Panel: ThemeSettings },
  { id: 'blocked', label: 'Đã chặn', icon: Ban, Panel: BlockedSettings },
]

/**
 * Settings, as one dialog with a list of sections down the side.
 *
 * Each section is its own component and the dialog only decides which one is
 * mounted, so a section that was not opened never runs its queries. The
 * blocked list in particular is a request nobody asked for until they click
 * it.
 *
 * Cài đặt, gộp trong một hộp thoại với danh sách mục dọc bên cạnh.
 *
 * Mỗi mục là một component riêng và hộp thoại chỉ quyết định mục nào được
 * mount, nên mục chưa mở thì không chạy truy vấn nào. Riêng danh sách người
 * bị chặn là một request không ai cần cho tới khi họ bấm vào.
 */
export function SettingsDialog({
  section,
  onSectionChange,
  onClose,
}: {
  section: SettingsSection
  onSectionChange: (next: SettingsSection) => void
  onClose: () => void
}) {
  const active = sections.find((entry) => entry.id === section) ?? sections[0]
  const Panel = active.Panel

  return (
    <Dialog open onOpenChange={(next) => !next && onClose()}>
      <DialogContent className="max-w-3xl overflow-hidden p-0 sm:max-w-3xl">
        <DialogHeader className="border-b border-border px-6 py-4">
          <DialogTitle>Cài đặt</DialogTitle>
          <DialogDescription className="sr-only">
            Chọn một mục ở bên trái để thay đổi cài đặt tương ứng.
          </DialogDescription>
        </DialogHeader>

        {/* Trên màn hình hẹp, danh sách mục nằm ngang phía trên thay vì thành
            một cột chiếm nửa bề ngang hộp thoại.

            On a narrow screen the section list sits across the top instead of
            becoming a column that eats half the dialog's width. */}
        <div className="flex max-h-[70dvh] flex-col sm:flex-row">
          <nav className="flex shrink-0 gap-1 overflow-x-auto border-b border-border p-2 sm:w-52 sm:flex-col sm:overflow-x-visible sm:border-b-0 sm:border-r">
            {sections.map(({ id, label, icon: Icon }) => (
              <button
                key={id}
                type="button"
                onClick={() => onSectionChange(id)}
                aria-current={id === section ? 'page' : undefined}
                className={cn(
                  'flex shrink-0 items-center gap-2.5 rounded-lg px-3 py-2 text-sm whitespace-nowrap transition-colors',
                  id === section ? 'bg-muted font-semibold' : 'hover:bg-muted',
                )}
              >
                <Icon className="size-4 shrink-0" />
                {label}
              </button>
            ))}
          </nav>

          <div className="min-w-0 flex-1 overflow-y-auto p-6">
            <Panel />
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}
