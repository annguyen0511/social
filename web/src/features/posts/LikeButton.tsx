import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Heart } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { cn } from 'cn'
import { del, put } from '../../api/client'
import type { LikeState } from '../../api/types'
import { patchPostLists } from './patchPostLists'

export function LikeButton({
  postID,
  likeCount,
  isLiked,
}: {
  postID: number
  likeCount: number
  isLiked: boolean
}) {
  const queryClient = useQueryClient()

  const mutation = useMutation({
    mutationFn: () =>
      isLiked
        ? del<LikeState>(`/v1/posts/${postID}/like`)
        : put<LikeState>(`/v1/posts/${postID}/like`),
    onSuccess: (next) => patchPostLists(queryClient, postID, next),
  })

  return (
    <Button
      type="button"
      variant="ghost"
      size="sm"
      disabled={mutation.isPending}
      aria-pressed={isLiked}
      aria-label={isLiked ? 'Bỏ thích' : 'Thích'}
      onClick={() => mutation.mutate()}
      className={cn(isLiked && 'text-red-600 hover:text-red-600')}
    >
      <Heart className={cn(isLiked && 'fill-current')} />
      {likeCount}
    </Button>
  )
}
