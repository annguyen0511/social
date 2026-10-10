import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Bookmark } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { cn } from 'cn'
import { del, put } from '../../api/client'
import type { SaveState } from '../../api/types'
import { patchPostLists } from './patchPostLists'

/**
 * Saves a post for later, or takes it back off the list.
 *
 * There is no number beside it, unlike the like button: saving is private, so
 * there is nothing to count in public.
 *
 * Lưu một bài để đọc lại, hoặc bỏ nó khỏi danh sách.
 *
 * Không có con số nào bên cạnh, khác với nút thích: việc lưu là riêng tư nên
 * không có gì để đếm công khai.
 */
export function SaveButton({ postID, isSaved }: { postID: number; isSaved: boolean }) {
  const queryClient = useQueryClient()

  const mutation = useMutation({
    mutationFn: () =>
      isSaved
        ? del<SaveState>(`/v1/posts/${postID}/save`)
        : put<SaveState>(`/v1/posts/${postID}/save`),
    onSuccess: (next) => {
      patchPostLists(queryClient, postID, next)

      // The saved list itself gains or loses a row, which patching a row in
      // place cannot express — so that one list is refetched.
      //
      // Riêng danh sách đã lưu thì thêm hoặc mất hẳn một dòng, mà việc vá một
      // dòng tại chỗ không diễn tả được — nên chỉ danh sách đó phải tải lại.
      queryClient.invalidateQueries({ queryKey: ['saved'] })
    },
  })

  return (
    <Button
      type="button"
      variant="ghost"
      size="sm"
      disabled={mutation.isPending}
      aria-pressed={isSaved}
      aria-label={isSaved ? 'Bỏ lưu' : 'Lưu bài'}
      title={isSaved ? 'Bỏ lưu' : 'Lưu bài'}
      onClick={() => mutation.mutate()}
      className={cn(isSaved && 'text-foreground')}
    >
      <Bookmark className={cn(isSaved && 'fill-current')} />
    </Button>
  )
}
