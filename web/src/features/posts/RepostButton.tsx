import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Repeat2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { cn } from 'cn'
import { HttpError, del, put } from '../../api/client'
import type { RepostState } from '../../api/types'
import { patchPostLists } from './patchPostLists'

/**
 * Shares someone's post under your name, or takes that back.
 *
 * Unlike saving, a repost is public and carries a count. The API refuses a
 * new repost while a block exists in either direction, but always allows
 * undoing one — otherwise a repost made before the block could never be
 * withdrawn.
 *
 * Chia sẻ bài của người khác dưới tên mình, hoặc rút lại việc đó.
 *
 * Khác với lưu bài, repost là công khai và có số đếm. API từ chối một lượt
 * repost mới khi có block ở bất kỳ chiều nào, nhưng luôn cho phép gỡ — nếu
 * không thì một lượt repost tạo ra trước khi bị chặn sẽ không bao giờ rút
 * lại được.
 */
export function RepostButton({
  postID,
  repostCount,
  isReposted,
}: {
  postID: number
  repostCount: number
  isReposted: boolean
}) {
  const queryClient = useQueryClient()

  const mutation = useMutation({
    mutationFn: () =>
      isReposted
        ? del<RepostState>(`/v1/posts/${postID}/repost`)
        : put<RepostState>(`/v1/posts/${postID}/repost`),
    onSuccess: (next) => {
      patchPostLists(queryClient, postID, next)

      // A repost list gains or loses a whole row, which patching a row in
      // place cannot express — so those lists are refetched.
      //
      // Danh sách repost thêm hoặc mất hẳn một dòng, mà việc vá một dòng tại
      // chỗ không diễn tả được — nên các danh sách đó phải tải lại.
      queryClient.invalidateQueries({ queryKey: ['userReposts'] })
    },
  })

  const blocked = mutation.error instanceof HttpError && mutation.error.status === 403

  return (
    <Button
      type="button"
      variant="ghost"
      size="sm"
      disabled={mutation.isPending}
      aria-pressed={isReposted}
      aria-label={isReposted ? 'Gỡ repost' : 'Repost'}
      title={blocked ? 'Không thể repost khi còn chặn giữa hai người' : undefined}
      onClick={() => mutation.mutate()}
      className={cn(isReposted && 'text-green-600 hover:text-green-600')}
    >
      <Repeat2 />
      {repostCount}
    </Button>
  )
}
