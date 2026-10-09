import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Heart } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { cn } from 'cn'
import { del, put } from '../../api/client'
import type { FeedPost, LikeState, Pagination, Post } from '../../api/types'

type InfinitePosts = { pages: Pagination<FeedPost>[]; pageParams: unknown[] }

/**
 * Rewrites one post wherever it is already cached.
 *
 * The same post appears in the feed, on its author's profile and on its own
 * page, under three different cache keys. Invalidating all of them would
 * refetch whole lists over one button press; patching them keeps every copy in
 * step without a single request.
 *
 * Ghi lại một bài viết ở mọi nơi nó đang nằm trong cache.
 *
 * Cùng một bài xuất hiện trong bảng tin, trên trang cá nhân của tác giả và ở
 * trang riêng của nó, dưới ba key cache khác nhau. Invalidate cả ba sẽ tải lại
 * nguyên các danh sách chỉ vì một cú bấm nút; vá trực tiếp thì mọi bản sao
 * khớp nhau mà không tốn request nào.
 */
const patchLists = (postID: number, next: LikeState) => (data: InfinitePosts | undefined) => {
  if (!data) return data
  return {
    ...data,
    pages: data.pages.map((page) => ({
      ...page,
      items: page.items.map((item) => (item.id === postID ? { ...item, ...next } : item)),
    })),
  }
}

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
    onSuccess: (next) => {
      queryClient.setQueryData<Post>(['post', postID], (old) =>
        old ? { ...old, ...next } : old,
      )
      queryClient.setQueriesData({ queryKey: ['feed'] }, patchLists(postID, next))
      queryClient.setQueriesData({ queryKey: ['userPosts'] }, patchLists(postID, next))
    },
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
