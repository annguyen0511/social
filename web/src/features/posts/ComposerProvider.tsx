import { createContext, useCallback, useContext, useState, type ReactNode } from 'react'
import { CreatePostDialog } from './CreatePostDialog'

const ComposerContext = createContext<(() => void) | null>(null)

/**
 * Holds the one "new post" dialog for the whole signed-in app.
 *
 * The button that opens it appears in several places — the navigation rail,
 * the feed, your own profile — and each rendering its own dialog would mean
 * several copies of the same draft, with whichever one happens to be mounted
 * deciding what gets posted. One dialog above all of them has a single draft.
 *
 * Giữ duy nhất một hộp thoại "bài viết mới" cho toàn bộ phần đã đăng nhập.
 *
 * Nút mở nó xuất hiện ở nhiều chỗ — thanh điều hướng, bảng tin, trang cá nhân
 * của bạn — mà mỗi chỗ tự dựng một hộp thoại riêng thì sẽ có nhiều bản nháp
 * của cùng một bài, và bản nào đang được mount sẽ quyết định cái gì được
 * đăng. Một hộp thoại đặt trên tất cả thì chỉ có một bản nháp.
 */
export function ComposerProvider({ children }: { children: ReactNode }) {
  const [open, setOpen] = useState(false)
  const openComposer = useCallback(() => setOpen(true), [])

  return (
    <ComposerContext.Provider value={openComposer}>
      {children}
      <CreatePostDialog open={open} onOpenChange={setOpen} />
    </ComposerContext.Provider>
  )
}

/**
 * Returns the function that opens the composer.
 *
 * It throws rather than returning a no-op when the provider is missing: a
 * button that silently does nothing is a bug someone finds by clicking it,
 * while this one shows up the first time the page renders.
 *
 * Trả về hàm mở trình soạn bài.
 *
 * Nó ném lỗi thay vì trả về một hàm rỗng khi thiếu provider: một cái nút bấm
 * vào không làm gì là lỗi chỉ lộ ra khi có người bấm thử, còn thế này thì lộ
 * ngay lần đầu trang được render.
 */
export function useComposer(): () => void {
  const open = useContext(ComposerContext)
  if (!open) {
    throw new Error('useComposer must be used inside a ComposerProvider')
  }
  return open
}
