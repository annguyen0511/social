import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import type { User } from '../../api/types'

/**
 * The user's picture, falling back to their initials.
 *
 * Radix decides when to show the fallback, which covers a case hand-written
 * markup usually misses: an avatar_url that is present but fails to load. An
 * <img> with a dead src leaves a broken image, while this swaps in initials.
 *
 * Ảnh của người dùng, không có thì lấy chữ cái đầu của tên.
 *
 * Radix quyết định khi nào hiện phần dự phòng, nhờ đó xử lý được một trường
 * hợp mà code viết tay thường bỏ sót: avatar_url có giá trị nhưng tải không
 * được. Một thẻ <img> với src chết sẽ để lại ảnh vỡ, còn ở đây nó tự đổi sang
 * chữ cái đầu của tên.
 */
export function UserAvatar({ user, className }: { user: User; className?: string }) {
  const initials = `${user.first_name.at(0) ?? ''}${user.last_name.at(0) ?? ''}`.toUpperCase()

  return (
    <Avatar className={className}>
      {/* src rỗng thì truyền undefined, để Radix coi là "không có ảnh" ngay
          thay vì thử tải một chuỗi rỗng rồi mới thất bại.

          An empty src becomes undefined so Radix treats it as "no image"
          straight away, instead of trying to load an empty string first. */}
      <AvatarImage src={user.avatar_url || undefined} alt="" />
      <AvatarFallback>{initials || '?'}</AvatarFallback>
    </Avatar>
  )
}
