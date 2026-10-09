import { useEffect, useRef, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { Command, CommandInput, CommandItem, CommandList } from '@/components/ui/command'
import { get } from '../../api/client'
import type { Pagination, SearchedUser } from '../../api/types'
import { useDebounced } from '../../hooks/useDebounced'
import { FollowButton } from './FollowButton'
import { UserAvatar } from './UserAvatar'

// Mirrors minSearchQuery in cmd/api/users.go. Asking with one character would
// only earn a 400, so the request is not worth sending.
//
// Khớp với minSearchQuery trong cmd/api/users.go. Gọi với một ký tự thì chỉ
// nhận về 400, nên không đáng gửi request.
const MIN_QUERY = 2

const RESULT_LIMIT = 8

/**
 * Search box with a dropdown of matching people.
 *
 * The debounced term is the queryKey, which buys three things at once: typing
 * does not flood the API, Query aborts the request for a term the reader has
 * already moved past, and a term typed twice comes back from cache.
 *
 * Built on cmdk rather than a plain input and list, because that is what
 * supplies arrow-key navigation and the listbox/option roles a screen reader
 * needs. Everything lives inside one Command: splitting the input and the list
 * across two components would put them in separate cmdk contexts, and the
 * arrow keys would stop reaching the results.
 *
 * Ô tìm kiếm kèm danh sách người khớp hiện xuống dưới.
 *
 * Từ khoá đã debounce chính là queryKey, và điều đó đem lại ba thứ cùng lúc:
 * việc gõ không làm ngập API, Query tự huỷ request của từ khoá mà người dùng
 * đã bỏ qua, và một từ khoá gõ lại lần hai thì lấy luôn từ cache.
 *
 * Dựng trên cmdk thay vì một input với một danh sách tự viết, vì đó là thứ
 * đem lại điều hướng bằng mũi lên/xuống và các role listbox/option mà trình
 * đọc màn hình cần. Tất cả nằm trong cùng một Command: tách input và danh sách
 * ra hai chỗ sẽ đặt chúng vào hai context cmdk khác nhau, và các phím mũi tên
 * không còn tới được danh sách kết quả.
 */
export function UserSearch() {
  const [term, setTerm] = useState('')
  const [open, setOpen] = useState(false)
  const boxRef = useRef<HTMLDivElement>(null)
  const navigate = useNavigate()

  const debounced = useDebounced(term.trim(), 300)
  const enabled = debounced.length >= MIN_QUERY

  const query = useQuery({
    queryKey: ['users', 'search', debounced],
    queryFn: ({ signal }) =>
      get<Pagination<SearchedUser>>(
        // encodeURIComponent, because a name can hold a space or an ampersand
        // and either one would otherwise cut the query string short.
        //
        // encodeURIComponent, vì một cái tên có thể chứa dấu cách hay dấu &,
        // mà cả hai đều sẽ cắt ngắn query string nếu không mã hoá.
        `/v1/users/search?q=${encodeURIComponent(debounced)}&page_size=${RESULT_LIMIT}`,
        signal,
      ),
    enabled,
    // Keeps the previous matches on screen while the next ones load, so the
    // dropdown does not blink empty between keystrokes.
    //
    // Giữ lại kết quả trước trên màn hình trong lúc kết quả mới đang tải, để
    // danh sách không nháy trắng giữa hai lần gõ.
    placeholderData: keepPreviousData,
  })

  // A click anywhere else closes the dropdown. pointerdown rather than click,
  // so the list is gone before a click on the page behind it resolves.
  //
  // This stays hand-written on purpose. Popover would supply it, but Popover
  // also moves focus into its content when it opens, which is wrong for a
  // search box: the reader has to keep typing in the input.
  //
  // Bấm ra ngoài thì đóng danh sách. Dùng pointerdown thay vì click, để danh
  // sách biến mất trước khi cú bấm vào phần trang phía sau được xử lý.
  //
  // Phần này cố ý viết tay. Popover có sẵn tính năng đó, nhưng Popover còn
  // chuyển focus vào trong nội dung khi mở, và điều đó sai với một ô tìm kiếm:
  // người dùng phải gõ tiếp được trong input.
  useEffect(() => {
    if (!open) return

    const onPointerDown = (event: PointerEvent) => {
      if (!boxRef.current?.contains(event.target as Node)) setOpen(false)
    }

    document.addEventListener('pointerdown', onPointerDown)
    return () => document.removeEventListener('pointerdown', onPointerDown)
  }, [open])

  const results = query.data?.items ?? []
  const showList = open && enabled

  return (
    <div ref={boxRef} className="relative">
      <Command
        // shouldFilter tắt vì Postgres đã lọc rồi. Để mặc định thì cmdk lọc
        // lần hai trên text hiển thị, và một kết quả khớp nhờ họ tên sẽ bị ẩn
        // đi chỉ vì username của người đó không chứa từ khoá.
        //
        // shouldFilter is off because Postgres already filtered. Left on, cmdk
        // would filter a second time over the visible text, and a row matched
        // by full name would vanish just because that person's username does
        // not contain the term.
        shouldFilter={false}
        loop
        className="overflow-visible bg-transparent p-0"
        onKeyDown={(event) => event.key === 'Escape' && setOpen(false)}
      >
        <CommandInput
          value={term}
          onValueChange={(value) => {
            setTerm(value)
            setOpen(true)
          }}
          onFocus={() => setOpen(true)}
          placeholder="Tìm người dùng…"
          aria-label="Tìm người dùng"
          className="h-9"
        />

        {showList && (
          <CommandList className="absolute top-full left-0 z-50 mt-1 w-full rounded-xl border border-border bg-popover p-1 text-popover-foreground shadow-lg">
            {query.isPending && (
              <p className="px-3 py-2 text-sm text-muted-foreground">Đang tìm…</p>
            )}

            {query.isError && (
              <p className="px-3 py-2 text-sm text-destructive">Không tìm được. Thử lại sau.</p>
            )}

            {query.isSuccess && results.length === 0 && (
              <p className="px-3 py-2 text-sm text-muted-foreground">
                Không tìm thấy ai khớp “{debounced}”.
              </p>
            )}

            {results.map((user) => (
              <CommandItem
                key={user.id}
                // cmdk cần value để biết item nào đang được chọn bằng bàn phím.
                // cmdk needs a value to track which item the keyboard is on.
                value={String(user.id)}
                onSelect={() => {
                  setOpen(false)
                  navigate(`/users/${user.id}`)
                }}
                // CommandItem tự chèn một CheckIcon ở cuối để đánh dấu lựa
                // chọn. Ở đây không có gì để tick, mà nó lại chiếm chỗ cạnh nút
                // theo dõi, nên ẩn đi. Selector chỉ nhắm svg là con trực tiếp,
                // nên icon trong nút không bị ảnh hưởng.
                //
                // CommandItem appends a CheckIcon to mark a selection. There is
                // nothing to tick here and it takes space next to the follow
                // button, so it is hidden. The selector only matches a direct
                // svg child, so icons inside the button are untouched.
                className="cursor-pointer gap-3 px-2 py-2 [&>svg:last-of-type]:hidden"
              >
                <UserAvatar user={user} className="size-9" />

                <span className="min-w-0 flex-1">
                  <span className="block truncate text-sm font-medium">
                    {user.first_name} {user.last_name}
                  </span>
                  <span className="block truncate text-sm text-muted-foreground">
                    @{user.username}
                  </span>
                </span>

                <FollowButton userID={user.id} isFollowing={user.is_following} />
              </CommandItem>
            ))}
          </CommandList>
        )}
      </Command>
    </div>
  )
}
