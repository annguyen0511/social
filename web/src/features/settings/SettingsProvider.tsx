import { createContext, useCallback, useContext, useState, type ReactNode } from 'react'
import { SettingsDialog } from './SettingsDialog'

export type SettingsSection = 'profile' | 'theme' | 'blocked'

type OpenSettings = (section?: SettingsSection) => void

const SettingsContext = createContext<OpenSettings | null>(null)

/**
 * Holds the one settings dialog for the whole signed-in app.
 *
 * Several places open it — the navigation rail, the edit button on your own
 * profile — and each at a different section. One dialog above all of them
 * means one place that knows how to close it, and no chance of two copies of
 * the same half-filled form.
 *
 * Giữ duy nhất một hộp thoại cài đặt cho toàn bộ phần đã đăng nhập.
 *
 * Nhiều chỗ mở nó ra — thanh điều hướng, nút chỉnh sửa trên trang cá nhân của
 * bạn — và mỗi chỗ mở ở một mục khác nhau. Một hộp thoại đặt trên tất cả
 * nghĩa là chỉ một nơi biết cách đóng nó, và không có chuyện hai bản sao của
 * cùng một biểu mẫu đang điền dở.
 */
export function SettingsProvider({ children }: { children: ReactNode }) {
  // null nghĩa là hộp thoại đang đóng; một giá trị vừa mở nó vừa chọn mục.
  // null means closed; a value both opens it and picks the section.
  const [section, setSection] = useState<SettingsSection | null>(null)

  const openSettings = useCallback<OpenSettings>((next = 'profile') => setSection(next), [])

  return (
    <SettingsContext.Provider value={openSettings}>
      {children}
      {section && (
        <SettingsDialog
          section={section}
          onSectionChange={setSection}
          onClose={() => setSection(null)}
        />
      )}
    </SettingsContext.Provider>
  )
}

/**
 * Returns the function that opens settings, optionally at a section.
 *
 * It throws rather than returning a no-op when the provider is missing: a
 * button that silently does nothing is a bug someone finds by clicking it,
 * while this one shows up the first time the page renders.
 *
 * Trả về hàm mở cài đặt, có thể kèm mục cụ thể.
 *
 * Nó ném lỗi thay vì trả về hàm rỗng khi thiếu provider: một cái nút bấm vào
 * không làm gì là lỗi chỉ lộ ra khi có người bấm thử, còn thế này thì lộ ngay
 * lần đầu trang được render.
 */
export function useSettings(): OpenSettings {
  const open = useContext(SettingsContext)
  if (!open) {
    throw new Error('useSettings must be used inside a SettingsProvider')
  }
  return open
}
