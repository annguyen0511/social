import { Lock } from 'lucide-react'
import type { Visibility } from '../../api/types'

/**
 * Marks a post as private.
 *
 * Nothing is drawn for a public post: a badge on every single card would be
 * noise, and public is what people already assume. The badge is there to
 * catch the eye on the exception.
 *
 * Đánh dấu một bài là riêng tư.
 *
 * Bài công khai thì không vẽ gì: gắn nhãn lên từng thẻ một sẽ thành nhiễu, mà
 * công khai vốn đã là thứ mọi người mặc định. Nhãn tồn tại để đập vào mắt ở
 * đúng trường hợp ngoại lệ.
 */
export function VisibilityBadge({ visibility }: { visibility: Visibility }) {
  if (visibility === 'public') return null

  return (
    <span
      className="inline-flex shrink-0 items-center gap-1 rounded-md bg-muted px-1.5 py-0.5 text-xs text-muted-foreground"
      title="Chỉ bạn thân của tác giả xem được bài này"
    >
      <Lock className="size-3" />
      Bạn thân
    </span>
  )
}
