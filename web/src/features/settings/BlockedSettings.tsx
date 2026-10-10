import { Link } from 'react-router-dom'
import { useInfiniteQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Loader2, ShieldOff } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { del, get } from '../../api/client'
import type { Pagination, User } from '../../api/types'
import { UserAvatar } from '../users/UserAvatar'

/**
 * Lifts a block from one person.
 *
 * Unblocking does not put back the follows the block removed, in either
 * direction — the API severed them in one transaction and keeps no record of
 * what was there. The wording below says so, because the opposite is what
 * most people assume.
 *
 * Gỡ chặn một người.
 *
 * Bỏ chặn không khôi phục lại các lượt theo dõi mà lệnh chặn đã xoá, ở cả hai
 * chiều — API đã cắt chúng trong một transaction và không lưu lại thứ gì từng
 * có. Dòng chữ bên dưới nói rõ điều đó, vì hầu hết mọi người mặc định điều
 * ngược lại.
 */
function UnblockButton({ userID }: { userID: number }) {
  const queryClient = useQueryClient()

  const mutation = useMutation({
    mutationFn: () => del<null>(`/v1/friend-ship/${userID}/block`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['blocked'] })
      // Someone unblocked reappears in search and can be followed again.
      // Người được bỏ chặn sẽ hiện lại trong tìm kiếm và theo dõi lại được.
      queryClient.invalidateQueries({ queryKey: ['users'] })
      queryClient.invalidateQueries({ queryKey: ['friendship', userID] })
    },
  })

  return (
    <Button
      type="button"
      variant="outline"
      size="sm"
      disabled={mutation.isPending}
      onClick={() => mutation.mutate()}
    >
      {mutation.isPending ? <Loader2 className="animate-spin" /> : <ShieldOff />}
      {mutation.isPending ? 'Đang gỡ' : 'Bỏ chặn'}
    </Button>
  )
}

export function BlockedSettings() {
  const query = useInfiniteQuery({
    queryKey: ['blocked'],
    initialPageParam: 1,
    queryFn: ({ pageParam, signal }) =>
      get<Pagination<User>>(`/v1/friend-ship/blocking?page=${pageParam}&page_size=20`, signal),
    getNextPageParam: (last) => (last.page < last.total_pages ? last.page + 1 : undefined),
  })

  const people = query.data?.pages.flatMap((page) => page.items) ?? []

  return (
    <section className="space-y-4">
      <div>
        <h2 className="font-semibold">Người đã chặn</h2>
        <p className="text-sm text-muted-foreground">
          Họ không thấy bài của bạn và không theo dõi bạn được. Bỏ chặn không khôi phục lại việc
          theo dõi trước đó.
        </p>
      </div>

      {query.isPending && (
        <div className="space-y-3">
          {[0, 1].map((i) => (
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
        <p className="text-sm text-muted-foreground">Bạn chưa chặn ai.</p>
      )}

      <ul className="divide-y divide-border">
        {people.map((person) => (
          <li key={person.id} className="flex items-center gap-3 py-3">
            <Link to={`/users/${person.id}`} className="flex min-w-0 flex-1 items-center gap-3">
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

            <UnblockButton userID={person.id} />
          </li>
        ))}
      </ul>

      {query.hasNextPage && (
        <Button
          variant="outline"
          onClick={() => query.fetchNextPage()}
          disabled={query.isFetchingNextPage}
          className="w-full"
        >
          {query.isFetchingNextPage && <Loader2 className="animate-spin" />}
          {query.isFetchingNextPage ? 'Đang tải' : 'Tải thêm'}
        </Button>
      )}
    </section>
  )
}
