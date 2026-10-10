import { Link } from 'react-router-dom'
import { useInfiniteQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Loader2, StarOff } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { del, get } from '../../api/client'
import type { Pagination, User } from '../../api/types'
import { UserAvatar } from '../users/UserAvatar'

/**
 * Takes one person off the close friends list.
 *
 * It costs them access to every private post at once, not just future ones —
 * visibility is checked when a post is read, not when it is written. The
 * wording above the list says so, because "unfriend" sounds like it only
 * applies going forward.
 *
 * Bỏ một người khỏi danh sách bạn thân.
 *
 * Việc đó lấy đi quyền xem của họ với toàn bộ bài riêng tư cùng lúc, không
 * riêng bài đăng sau này — quyền hiển thị được kiểm lúc đọc bài chứ không
 * phải lúc viết. Dòng chữ phía trên danh sách nói rõ điều đó, vì "bỏ bạn
 * thân" nghe như chỉ có hiệu lực từ nay về sau.
 */
function RemoveButton({ userID }: { userID: number }) {
  const queryClient = useQueryClient()

  const mutation = useMutation({
    mutationFn: () => del<null>(`/v1/friend-ship/${userID}/close-friend`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['closeFriends'] })
      queryClient.invalidateQueries({ queryKey: ['friendship', userID] })
      // Their view of private posts changes, and so does what any list of
      // posts returns for them — but not for the person doing the removing.
      // Refreshing here is for the profile that shows the relationship.
      //
      // Cách họ nhìn các bài riêng tư thay đổi, và các danh sách bài trả về
      // cho họ cũng vậy — nhưng không đổi với người đang thực hiện việc bỏ.
      // Làm mới ở đây là cho trang cá nhân đang hiển thị mối quan hệ đó.
      queryClient.invalidateQueries({ queryKey: ['users'] })
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
      {mutation.isPending ? <Loader2 className="animate-spin" /> : <StarOff />}
      {mutation.isPending ? 'Đang bỏ' : 'Bỏ'}
    </Button>
  )
}

export function CloseFriendsSettings() {
  const query = useInfiniteQuery({
    queryKey: ['closeFriends'],
    initialPageParam: 1,
    queryFn: ({ pageParam, signal }) =>
      get<Pagination<User>>(`/v1/friend-ship/close-friends?page=${pageParam}&page_size=20`, signal),
    getNextPageParam: (last) => (last.page < last.total_pages ? last.page + 1 : undefined),
  })

  const people = query.data?.pages.flatMap((page) => page.items) ?? []

  return (
    <section className="space-y-4">
      <div>
        <h2 className="font-semibold">Bạn thân</h2>
        <p className="text-sm text-muted-foreground">
          Chỉ những người này xem được bài bạn đặt ở chế độ Bạn thân. Bỏ ai ra khỏi danh sách thì
          họ mất quyền xem cả những bài cũ, không riêng bài đăng sau này.
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
        <p className="text-sm text-muted-foreground">
          Bạn chưa thêm ai. Mở trang cá nhân của một người rồi bấm “Thêm bạn thân”.
        </p>
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

            <RemoveButton userID={person.id} />
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
