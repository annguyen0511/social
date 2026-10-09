import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Loader2, SendHorizontal } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Textarea } from '@/components/ui/textarea'
import { post } from '../../api/client'
import type { Comment } from '../../api/types'

const MAX_LENGTH = 1000

/**
 * Adds a comment to a post.
 *
 * The response carries the new comment but not its author, so the list is
 * refetched rather than appended to. Appending would show a comment with a
 * blank name and a blank avatar until something else happened to reload it.
 *
 * Thêm một bình luận vào bài viết.
 *
 * Response trả về bình luận mới nhưng không kèm tác giả, nên danh sách được
 * tải lại chứ không nối thêm tại chỗ. Nối thêm sẽ hiện một bình luận trống
 * tên và trống ảnh, cho tới khi có việc gì khác tình cờ nạp lại nó.
 */
export function CommentForm({ postID }: { postID: number }) {
  const [content, setContent] = useState('')
  const queryClient = useQueryClient()

  const mutation = useMutation({
    mutationFn: () => post<Comment>(`/v1/posts/${postID}/comment`, { content: content.trim() }),
    onSuccess: () => {
      setContent('')
      queryClient.invalidateQueries({ queryKey: ['post', postID] })
      // The card in a list shows a comment count, which just moved.
      // Thẻ bài trong danh sách có hiện số bình luận, mà con số đó vừa đổi.
      queryClient.invalidateQueries({ queryKey: ['feed'] })
      queryClient.invalidateQueries({ queryKey: ['userPosts'] })
    },
  })

  const ready = content.trim().length > 0 && content.trim().length <= MAX_LENGTH

  return (
    <form
      className="mb-4 space-y-2"
      onSubmit={(event) => {
        event.preventDefault()
        if (ready) mutation.mutate()
      }}
    >
      <Textarea
        value={content}
        rows={3}
        maxLength={MAX_LENGTH}
        placeholder="Viết bình luận…"
        aria-label="Viết bình luận"
        onChange={(event) => setContent(event.target.value)}
      />

      {mutation.isError && <p className="text-sm text-destructive">{mutation.error.message}</p>}

      <div className="flex items-center justify-between gap-4">
        <span className="text-xs text-muted-foreground">
          {content.length}/{MAX_LENGTH}
        </span>
        <Button type="submit" size="sm" disabled={!ready || mutation.isPending}>
          {mutation.isPending ? <Loader2 className="animate-spin" /> : <SendHorizontal />}
          {mutation.isPending ? 'Đang gửi' : 'Gửi'}
        </Button>
      </div>
    </form>
  )
}
