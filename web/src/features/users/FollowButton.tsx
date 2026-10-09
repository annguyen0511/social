import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Ban, Loader2, UserCheck, UserPlus } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { HttpError, del, put } from '../../api/client'

type Props = {
  userID: number
  isFollowing: boolean
  className?: string
}

/**
 * Follows or unfollows, picking the verb from the state it is handed.
 *
 * Both API calls are idempotent, so a double click cannot land the server in a
 * state the button did not mean. What it does invalidate is the feed: who you
 * follow decides what the feed contains, so leaving the old pages cached would
 * show a feed that contradicts the button the reader just pressed.
 *
 * Theo dõi hoặc bỏ theo dõi, chọn hành động dựa trên trạng thái được truyền vào.
 *
 * Cả hai lệnh gọi API đều idempotent, nên bấm hai lần không thể đưa server tới
 * một trạng thái mà nút không chủ ý. Thứ cần làm mới là feed: việc theo dõi ai
 * quyết định feed có gì, nên giữ lại cache cũ sẽ hiện một feed trái ngược với
 * cái nút mà người dùng vừa bấm.
 */
export function FollowButton({ userID, isFollowing, className }: Props) {
  const queryClient = useQueryClient()

  const mutation = useMutation({
    mutationFn: () =>
      isFollowing
        ? del<null>(`/v1/friend-ship/${userID}/follow`)
        : put<null>(`/v1/friend-ship/${userID}/follow`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['feed'] })
      // The whole 'users' prefix, not just search: a search row carries
      // is_following, and a profile carries the follower count that this press
      // just changed. Invalidating only one of them leaves a number on screen
      // that contradicts the button beside it.
      //
      // Cả tiền tố 'users' chứ không riêng search: một dòng kết quả tìm kiếm
      // mang theo is_following, còn trang cá nhân mang con số người theo dõi
      // mà cú bấm này vừa làm đổi. Chỉ làm mới một trong hai sẽ để lại trên
      // màn hình một con số mâu thuẫn với chính cái nút bên cạnh nó.
      queryClient.invalidateQueries({ queryKey: ['users'] })
      queryClient.invalidateQueries({ queryKey: ['friendship', userID] })
      // Each row of a follower or following list carries is_following too, and
      // this button is often pressed from inside one of those lists.
      //
      // Mỗi dòng trong danh sách người theo dõi hay đang theo dõi cũng mang
      // theo is_following, mà nút này thường được bấm từ ngay trong một trong
      // hai danh sách đó.
      queryClient.invalidateQueries({ queryKey: ['people'] })
    },
  })

  // 403 is the API refusing because a block exists in one direction or the
  // other. It is the one failure worth explaining in place.
  //
  // 403 là khi API từ chối vì có block ở một trong hai chiều. Đây là lỗi duy
  // nhất đáng giải thích ngay tại chỗ.
  const blocked = mutation.error instanceof HttpError && mutation.error.status === 403

  const { icon, label } = mutation.isPending
    ? { icon: <Loader2 className="animate-spin" />, label: 'Đang lưu' }
    : blocked
      ? { icon: <Ban />, label: 'Bị chặn' }
      : isFollowing
        ? { icon: <UserCheck />, label: 'Đang theo dõi' }
        : { icon: <UserPlus />, label: 'Theo dõi' }

  return (
    <Button
      type="button"
      size="sm"
      variant={isFollowing || blocked ? 'outline' : 'default'}
      disabled={mutation.isPending}
      title={blocked ? 'Không thể theo dõi khi còn chặn giữa hai người' : undefined}
      className={className}
      // Nút này nằm trong một CommandItem ở ô tìm kiếm, mà cmdk coi mọi cú bấm
      // trong item là "chọn item" và sẽ điều hướng sang trang profile. Chặn
      // lan truyền để bấm nút chỉ là bấm nút.
      //
      // This button sits inside a CommandItem in the search box, and cmdk
      // treats any click within an item as selecting it, which would navigate
      // to the profile. Stopping propagation keeps a click on the button a
      // click on the button.
      onPointerDown={(event) => event.stopPropagation()}
      onClick={(event) => {
        event.stopPropagation()
        mutation.mutate()
      }}
    >
      {icon}
      {label}
    </Button>
  )
}
