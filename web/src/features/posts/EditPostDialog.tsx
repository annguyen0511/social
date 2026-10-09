import { useEffect, useState } from 'react'
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
import { HttpError, patch } from '../../api/client'
import type { Post, PostUpdate } from '../../api/types'
import { isValidPost, parseTags, PostForm, type PostFields } from './PostForm'

const toFields = (post: Post): PostFields => ({
  title: post.title,
  content: post.content,
  tags: (post.tags ?? []).join(', '),
})

export function EditPostDialog({
  post,
  open,
  onOpenChange,
}: {
  post: Post
  open: boolean
  onOpenChange: (open: boolean) => void
}) {
  const [fields, setFields] = useState<PostFields>(() => toFields(post))
  const queryClient = useQueryClient()

  useEffect(() => {
    if (open) setFields(toFields(post))
  }, [open, post])

  const mutation = useMutation({
    mutationFn: () => {
      // Only what changed is sent. The API treats an omitted field as "keep
      // the current value", so sending everything would overwrite with values
      // this dialog read earlier.
      //
      // Chỉ gửi phần đã đổi. API coi field không gửi là "giữ nguyên giá trị
      // hiện tại", nên gửi hết sẽ ghi đè bằng những giá trị dialog đọc từ
      // trước.
      const before = toFields(post)
      const body: PostUpdate = {}
      if (fields.title.trim() !== before.title) body.title = fields.title.trim()
      if (fields.content.trim() !== before.content) body.content = fields.content.trim()
      if (fields.tags !== before.tags) body.tags = parseTags(fields.tags)
      return patch<Post>(`/v1/posts/${post.id}`, body)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['post', post.id] })
      queryClient.invalidateQueries({ queryKey: ['feed'] })
      queryClient.invalidateQueries({ queryKey: ['userPosts'] })
      onOpenChange(false)
    },
  })

  // The posts table carries a version column and the update only matches a row
  // whose version is unchanged, so a 409 means someone else edited this post
  // since this dialog read it.
  //
  // Bảng posts có cột version và lệnh cập nhật chỉ khớp dòng nào còn nguyên
  // version, nên 409 nghĩa là có người khác đã sửa bài này kể từ lúc dialog
  // đọc nó.
  const errorText =
    mutation.error instanceof HttpError
      ? mutation.error.status === 409
        ? 'Bài này vừa được sửa ở nơi khác. Đóng rồi mở lại để lấy bản mới nhất.'
        : mutation.error.message
      : null

  const dirty = JSON.stringify(fields) !== JSON.stringify(toFields(post))

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Sửa bài viết</DialogTitle>
          <DialogDescription>Thay đổi sẽ hiện ngay với mọi người.</DialogDescription>
        </DialogHeader>

        <PostForm id="edit-post" fields={fields} onChange={setFields} onSubmit={() => mutation.mutate()} />

        {errorText && <p className="text-sm text-destructive">{errorText}</p>}

        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            Huỷ
          </Button>
          <Button
            type="submit"
            form="edit-post"
            disabled={!dirty || !isValidPost(fields) || mutation.isPending}
          >
            {mutation.isPending && <Loader2 className="animate-spin" />}
            {mutation.isPending ? 'Đang lưu' : 'Lưu'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
