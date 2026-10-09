import { Link } from 'react-router-dom'
import { MessageCircle } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import type { FeedPost } from '../../api/types'
import { formatDateTime } from '../../lib/format'
import { UserAvatar } from '../users/UserAvatar'
import { LikeButton } from './LikeButton'

/**
 * One post as it appears in a list.
 *
 * The author's name and the body are two separate links rather than one card
 * wrapped in a link: an anchor inside an anchor is invalid HTML, and browsers
 * disagree about which one a click belongs to.
 *
 * Một bài viết khi nằm trong danh sách.
 *
 * Tên tác giả và phần thân là hai link tách rời chứ không phải cả thẻ bọc
 * trong một link: một thẻ a lồng trong thẻ a là HTML không hợp lệ, và các
 * trình duyệt không thống nhất cú bấm thuộc về cái nào.
 */
export function PostCard({ post }: { post: FeedPost }) {
  return (
    <Card>
      <CardContent>
        <div className="mb-3 flex items-center gap-2">
          <Link to={`/users/${post.user.id}`} className="flex min-w-0 items-center gap-2">
            <UserAvatar user={post.user} className="size-8 text-xs" />
            <span className="truncate text-sm font-medium hover:underline">
              @{post.user.username}
            </span>
          </Link>
          <span className="shrink-0 text-sm text-muted-foreground">
            · {formatDateTime(post.created_at)}
          </span>
        </div>

        <Link to={`/posts/${post.id}`} className="block">
          <h2 className="font-semibold hover:underline">{post.title}</h2>
          <p className="mt-1 line-clamp-3 text-foreground/80">{post.content}</p>
        </Link>

        {post.tags && post.tags.length > 0 && (
          <ul className="mt-3 flex flex-wrap gap-1.5">
            {post.tags.map((tag) => (
              <li key={tag} className="rounded-md bg-muted px-2 py-0.5 text-xs text-muted-foreground">
                #{tag}
              </li>
            ))}
          </ul>
        )}

        <div className="mt-3 -ml-2 flex items-center gap-1">
          <LikeButton postID={post.id} likeCount={post.like_count} isLiked={post.is_liked} />

          {/* Số bình luận dẫn tới trang chi tiết, vì đó là nơi duy nhất đọc và
              viết được bình luận.

              The comment count leads to the detail page, which is the only
              place comments can be read and written. */}
          <Button asChild variant="ghost" size="sm">
            <Link to={`/posts/${post.id}`}>
              <MessageCircle />
              {post.comment_count}
            </Link>
          </Button>
        </div>
      </CardContent>
    </Card>
  )
}
