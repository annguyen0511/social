import { SquarePlus } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { useComposer } from './ComposerProvider'

/**
 * Opens the composer. The dialog itself lives in ComposerProvider, so every
 * one of these buttons shares a single draft.
 *
 * Mở trình soạn bài. Bản thân hộp thoại nằm trong ComposerProvider, nên mọi
 * nút kiểu này đều dùng chung một bản nháp duy nhất.
 */
export function NewPostButton({ size = 'sm' }: { size?: 'sm' | 'default' }) {
  const openComposer = useComposer()

  return (
    <Button type="button" size={size} onClick={openComposer}>
      <SquarePlus />
      Viết bài
    </Button>
  )
}
