import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Loader2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { post } from '../../api/client'
import type { Post, PostCreate } from '../../api/types'
import { emptyPost, isValidPost, parseTags, PostForm, type PostFields } from './PostForm'

export function CreatePostDialog({
  open,
  onOpenChange,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
}) {
  const [fields, setFields] = useState<PostFields>(emptyPost)
  const queryClient = useQueryClient()
  const navigate = useNavigate()

  useEffect(() => {
    if (open) setFields(emptyPost)
  }, [open])

  const mutation = useMutation({
    mutationFn: () => {
      const body: PostCreate = {
        title: fields.title.trim(),
        content: fields.content.trim(),
        tags: parseTags(fields.tags),
        visibility: fields.visibility,
      }
      return post<Post>('/v1/posts', body)
    },
    onSuccess: (created) => {
      // A new post belongs in the author's own feed and on their profile, and
      // both are now one row out of date.
      //
      // Bài mới thuộc về feed của chính tác giả và trang cá nhân của họ, mà cả
      // hai đều vừa thiếu mất một dòng.
      queryClient.invalidateQueries({ queryKey: ['feed'] })
      queryClient.invalidateQueries({ queryKey: ['userPosts'] })
      onOpenChange(false)
      navigate(`/posts/${created.id}`)
    },
  })

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Bài viết mới</DialogTitle>
          <DialogDescription>Bài sẽ hiện trên bảng tin của bạn và những người theo dõi bạn.</DialogDescription>
        </DialogHeader>

        <PostForm id="create-post" fields={fields} onChange={setFields} onSubmit={() => mutation.mutate()} />

        {mutation.isError && (
          <p className="text-sm text-destructive">{mutation.error.message}</p>
        )}

        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            Huỷ
          </Button>
          <Button
            type="submit"
            form="create-post"
            disabled={!isValidPost(fields) || mutation.isPending}
          >
            {mutation.isPending && <Loader2 className="animate-spin" />}
            {mutation.isPending ? 'Đang đăng' : 'Đăng'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
