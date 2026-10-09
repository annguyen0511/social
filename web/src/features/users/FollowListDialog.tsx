import { Link } from 'react-router-dom'
import { useInfiniteQuery, useQuery } from '@tanstack/react-query'
import { Loader2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Skeleton } from '@/components/ui/skeleton'
import { get } from '../../api/client'
import type { Pagination, User, UserSummary } from '../../api/types'
import { FollowButton } from './FollowButton'
import { UserAvatar } from './UserAvatar'

export type FollowListKind = 'followers' | 'following'

const titles: Record<FollowListKind, { title: string; empty: (own: boolean) => string }> = {
  followers: {
    title: 'Người theo dõi',
    empty: (own) => (own ? 'Chưa có ai theo dõi bạn.' : 'Chưa có ai theo dõi người này.'),
  },
  following: {
    title: 'Đang theo dõi',
    empty: (own) => (own ? 'Bạn chưa theo dõi ai.' : 'Người này chưa theo dõi ai.'),
  },
}

/**
 * The list behind the two numbers on a profile.
 *
 * The query is disabled until the dialog opens, so a profile does not fetch
 * two lists nobody asked to see. It keys on the kind as well as the user, so
 * switching from followers to following is a different cache entry rather
 * than the first list showing under the second one's heading.
 *
 * Danh sách nằm sau hai con số trên trang cá nhân.
 *
 * Truy vấn bị tắt cho tới khi hộp thoại mở, để một trang cá nhân không đi tải
 * hai danh sách mà chưa ai muốn xem. Key gồm cả loại danh sách lẫn user, nên
 * chuyển từ "người theo dõi" sang "đang theo dõi" là một mục cache khác, chứ
 * không phải danh sách thứ nhất hiện dưới tiêu đề của cái thứ hai.
 */
export function FollowListDialog({
  userID,
  kind,
  open,
  onOpenChange,
}: {
  userID: string
  kind: FollowListKind
  open: boolean
  onOpenChange: (open: boolean) => void
}) {
  const me = useQuery({
    queryKey: ['users', 'me'],
    queryFn: ({ signal }) => get<User>('/v1/users/me', signal),
  })

  const query = useInfiniteQuery({
    queryKey: ['people', kind, userID],
    initialPageParam: 1,
    queryFn: ({ pageParam, signal }) =>
      get<Pagination<UserSummary>>(
        `/v1/users/${userID}/${kind}?page=${pageParam}&page_size=20`,
        signal,
      ),
    getNextPageParam: (last) => (last.page < last.total_pages ? last.page + 1 : undefined),
    enabled: open,
  })

  const people = query.data?.pages.flatMap((page) => page.items) ?? []
  const own = me.data?.id === Number(userID)
  const labels = titles[kind]

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[80dvh] overflow-hidden">
        <DialogHeader>
          <DialogTitle>{labels.title}</DialogTitle>
          <DialogDescription className="sr-only">
            Danh sách người, bấm vào một dòng để mở trang cá nhân.
          </DialogDescription>
        </DialogHeader>

        <div className="-mx-2 max-h-[60dvh] overflow-y-auto px-2">
          {query.isPending && (
            <div className="space-y-3">
              {[0, 1, 2].map((i) => (
                <div key={i} className="flex items-center gap-3">
                  <Skeleton className="size-10 shrink-0 rounded-full" />
                  <div className="flex-1 space-y-1.5">
                    <Skeleton className="h-4 w-32" />
                    <Skeleton className="h-3 w-20" />
                  </div>
                </div>
              ))}
            </div>
          )}

          {query.isError && <p className="text-sm text-destructive">Không tải được danh sách.</p>}

          {query.isSuccess && people.length === 0 && (
            <p className="text-sm text-muted-foreground">{labels.empty(own)}</p>
          )}

          <ul className="divide-y divide-border">
            {people.map((person) => (
              <li key={person.id} className="flex items-center gap-3 py-2">
                <Link
                  to={`/users/${person.id}`}
                  onClick={() => onOpenChange(false)}
                  className="flex min-w-0 flex-1 items-center gap-3"
                >
                  <UserAvatar user={person} className="size-10" />
                  <span className="min-w-0">
                    <span className="block truncate text-sm font-medium">
                      {person.first_name} {person.last_name}
                    </span>
                    <span className="block truncate text-sm text-muted-foreground">
                      @{person.username}
                    </span>
                  </span>
                </Link>

                {/* Không vẽ nút theo dõi trên chính mình: API trả 400 cho hành
                    động nhắm vào bản thân, nên đó sẽ là một cái nút chắc chắn
                    hỏng.

                    No follow button on your own row: the API answers 400 for
                    an action aimed at yourself, so it would be a button that
                    cannot work. */}
                {me.data && me.data.id !== person.id && (
                  <FollowButton userID={person.id} isFollowing={person.is_following} />
                )}
              </li>
            ))}
          </ul>

          {query.hasNextPage && (
            <Button
              variant="outline"
              onClick={() => query.fetchNextPage()}
              disabled={query.isFetchingNextPage}
              className="mt-4 w-full"
            >
              {query.isFetchingNextPage && <Loader2 className="animate-spin" />}
              {query.isFetchingNextPage ? 'Đang tải' : 'Tải thêm'}
            </Button>
          )}
        </div>
      </DialogContent>
    </Dialog>
  )
}
