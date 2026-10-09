import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Ellipsis, Loader2, Pencil, Trash } from 'lucide-react'
import {
  AlertDialog,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { del } from '../../api/client'
import type { Post } from '../../api/types'
import { EditPostDialog } from './EditPostDialog'

/**
 * The "..." menu on a post you wrote: edit, or delete.
 *
 * Deleting asks first, in a dialog that cannot be dismissed by clicking the
 * backdrop. The post and its comments go in one irreversible step, and the
 * button sits right next to the one that merely opens an editor.
 *
 * Menu "..." trên bài do chính bạn viết: sửa, hoặc xoá.
 *
 * Xoá thì hỏi lại trước, trong một hộp thoại không đóng được bằng cách bấm ra
 * nền. Bài viết cùng toàn bộ bình luận của nó mất đi trong một bước không thể
 * hoàn tác, mà nút đó lại nằm ngay cạnh nút chỉ mở trình soạn thảo.
 */
export function PostActions({ post }: { post: Post }) {
  const [editing, setEditing] = useState(false)
  const [confirming, setConfirming] = useState(false)
  const queryClient = useQueryClient()
  const navigate = useNavigate()

  const remove = useMutation({
    mutationFn: () => del<null>(`/v1/posts/${post.id}`),
    onSuccess: () => {
      // The page being viewed is about to stop existing, so leave before the
      // refetch turns it into a 404.
      //
      // Trang đang xem sắp không còn tồn tại, nên rời đi trước khi lượt tải
      // lại biến nó thành 404.
      queryClient.removeQueries({ queryKey: ['post', post.id] })
      queryClient.invalidateQueries({ queryKey: ['feed'] })
      queryClient.invalidateQueries({ queryKey: ['userPosts'] })
      navigate(`/users/${post.user_id}`, { replace: true })
    },
  })

  return (
    <>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button type="button" variant="ghost" size="icon-sm" aria-label="Tuỳ chọn bài viết">
            <Ellipsis />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          <DropdownMenuItem onSelect={() => setEditing(true)}>
            <Pencil />
            Sửa bài viết
          </DropdownMenuItem>
          <DropdownMenuItem variant="destructive" onSelect={() => setConfirming(true)}>
            <Trash />
            Xoá bài viết
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>

      <EditPostDialog post={post} open={editing} onOpenChange={setEditing} />

      <AlertDialog open={confirming} onOpenChange={setConfirming}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Xoá bài viết này?</AlertDialogTitle>
            <AlertDialogDescription>
              Bài viết và toàn bộ bình luận của nó sẽ bị xoá vĩnh viễn. Không hoàn tác được.
            </AlertDialogDescription>
          </AlertDialogHeader>

          {remove.isError && <p className="text-sm text-destructive">{remove.error.message}</p>}

          <AlertDialogFooter>
            <AlertDialogCancel disabled={remove.isPending}>Huỷ</AlertDialogCancel>
            <Button
              type="button"
              variant="destructive"
              disabled={remove.isPending}
              onClick={() => remove.mutate()}
            >
              {remove.isPending && <Loader2 className="animate-spin" />}
              {remove.isPending ? 'Đang xoá' : 'Xoá'}
            </Button>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}
