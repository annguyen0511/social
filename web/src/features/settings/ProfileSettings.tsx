import { useEffect, useMemo, useRef, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Loader2, Trash, Upload } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Skeleton } from '@/components/ui/skeleton'
import { HttpError, del, get, patch, postForm } from '../../api/client'
import type { ProfileUpdate, User } from '../../api/types'
import { UserAvatar } from '../users/UserAvatar'

// Mirrors the server: upload.MaxBytes and the formats image.Decode is set up
// to read. Checking here only saves a doomed round trip; the server stays the
// authority, and it re-encodes whatever arrives.
//
// Khớp với server: upload.MaxBytes và các định dạng mà image.Decode được cài
// để đọc. Kiểm ở đây chỉ để đỡ một lượt gọi mạng chắc chắn thất bại; server
// vẫn là nơi quyết định, và nó mã hoá lại mọi thứ nhận được.
const MAX_BYTES = 5 * 1024 * 1024
const ACCEPT = 'image/jpeg,image/png,image/gif'

const textFields = [
  { name: 'first_name', label: 'Tên', autoComplete: 'given-name' },
  { name: 'last_name', label: 'Họ', autoComplete: 'family-name' },
  { name: 'username', label: 'Tên đăng nhập', autoComplete: 'username' },
] as const

type Form = Record<(typeof textFields)[number]['name'], string>

const toForm = (user: User): Form => ({
  first_name: user.first_name,
  last_name: user.last_name,
  username: user.username,
})

/**
 * Edits the signed-in user's own profile, including the picture.
 *
 * Nothing is sent until the reader confirms. A picked file is held in state
 * and shown as a preview, so choosing the wrong photo costs a second thought
 * rather than a round trip and an undo.
 *
 * Sửa profile của chính người đang đăng nhập, gồm cả ảnh đại diện.
 *
 * Không gửi gì cho tới khi người dùng xác nhận. File đã chọn được giữ trong
 * state và hiện ra để xem trước, nên chọn nhầm ảnh chỉ tốn một giây suy nghĩ
 * lại chứ không phải một lượt gọi mạng rồi phải hoàn tác.
 */
export function ProfileSettings() {
  // Read from the cache the layout already filled, so opening settings does
  // not wait on a request that has almost certainly already happened.
  //
  // Đọc từ cache mà layout đã nạp sẵn, để mở cài đặt không phải chờ một
  // request gần như chắc chắn đã chạy xong từ trước.
  const me = useQuery({
    queryKey: ['users', 'me'],
    queryFn: ({ signal }) => get<User>('/v1/users/me', signal),
  })

  if (!me.data) {
    return (
      <div className="space-y-4">
        <Skeleton className="h-16 w-full" />
        <Skeleton className="h-10 w-full" />
        <Skeleton className="h-10 w-full" />
      </div>
    )
  }

  // The form is a separate component so its fields can be initialised from
  // the account directly, instead of starting empty and being filled by an
  // effect. An effect does not run on the first render, which left the inputs
  // blank for a moment and — because blank differs from the saved values —
  // the save button wrongly enabled.
  //
  // key remounts it if a different account ever appears here.
  //
  // Form là một component riêng để các ô được khởi tạo thẳng từ dữ liệu tài
  // khoản, thay vì bắt đầu rỗng rồi được một effect điền vào. Effect không
  // chạy ở lần render đầu, nên các ô từng rỗng trong chốc lát và — vì rỗng
  // khác với giá trị đã lưu — nút Lưu lại đang mở.
  //
  // key làm nó mount lại nếu có một tài khoản khác xuất hiện ở đây.
  return <ProfileForm key={me.data.id} user={me.data} />
}

function ProfileForm({ user }: { user: User }) {
  const queryClient = useQueryClient()

  const [form, setForm] = useState<Form>(() => toForm(user))
  const [file, setFile] = useState<File | null>(null)
  const [clearAvatar, setClearAvatar] = useState(false)
  const [fileError, setFileError] = useState<string | null>(null)
  const fileInput = useRef<HTMLInputElement>(null)

  // A blob: URL is a reference the browser holds onto until it is revoked, so
  // the old one has to go whenever the file changes or this unmounts. Leaving
  // them behind keeps every picture the reader tried in memory.
  //
  // blob: URL là một tham chiếu mà trình duyệt giữ lại cho tới khi bị thu
  // hồi, nên phải bỏ cái cũ mỗi lần đổi file hoặc khi component rời màn hình.
  // Để lại thì mọi tấm ảnh người dùng đã thử đều nằm mãi trong bộ nhớ.
  const preview = useMemo(() => (file ? URL.createObjectURL(file) : null), [file])
  useEffect(() => {
    if (!preview) return
    return () => URL.revokeObjectURL(preview)
  }, [preview])

  const pick = (chosen: File | undefined) => {
    if (!chosen) return
    if (!chosen.type.startsWith('image/')) {
      setFileError('Hãy chọn một file ảnh.')
      return
    }
    if (chosen.size > MAX_BYTES) {
      setFileError('Ảnh phải nhỏ hơn 5 MB.')
      return
    }
    setFileError(null)
    setClearAvatar(false)
    setFile(chosen)
  }

  const mutation = useMutation({
    mutationFn: async () => {
      // The picture and the text fields are separate endpoints, because one
      // is a file and the other is JSON. The picture goes first: if the name
      // then fails validation, the reader keeps the avatar they just chose
      // and only has the name left to fix.
      //
      // Ảnh và các field chữ là hai endpoint khác nhau, vì một bên là file và
      // một bên là JSON. Ảnh đi trước: nếu sau đó tên không hợp lệ thì người
      // dùng vẫn giữ được avatar vừa chọn và chỉ còn phải sửa cái tên.
      if (file) {
        const body = new FormData()
        body.append('avatar', file)
        await postForm<User>('/v1/users/me/avatar', body)
      } else if (clearAvatar && user.avatar_url) {
        await del<User>('/v1/users/me/avatar')
      }

      const changed: ProfileUpdate = {}
      for (const { name } of textFields) {
        if (form[name] !== toForm(user)[name]) changed[name] = form[name]
      }
      if (Object.keys(changed).length > 0) {
        await patch<User>('/v1/users/me', changed)
      }
    },
    onSuccess: () => {
      // The username and the avatar show up on the profile, on every feed post
      // and in search results, so all of them are stale now.
      //
      // Username và avatar xuất hiện trên profile, trên mọi bài trong feed và
      // trong kết quả tìm kiếm, nên tất cả đều đã cũ.
      queryClient.invalidateQueries({ queryKey: ['users'] })
      queryClient.invalidateQueries({ queryKey: ['feed'] })

      // The text fields already hold what was just saved, so only the picture
      // choice needs clearing — otherwise the form stays "dirty" forever.
      //
      // Các ô chữ vốn đã mang đúng thứ vừa lưu, nên chỉ phần chọn ảnh cần dọn
      // — nếu không thì form mãi mãi ở trạng thái "có thay đổi chưa lưu".
      setFile(null)
      setClearAvatar(false)
    },
  })

  const shownAvatar = preview ?? (clearAvatar ? '' : user.avatar_url)
  const dirty =
    file !== null ||
    (clearAvatar && user.avatar_url !== '') ||
    textFields.some(({ name }) => form[name] !== toForm(user)[name])

  const errorText =
    mutation.error instanceof HttpError
      ? mutation.error.status === 409
        ? 'Tên đăng nhập này đã có người dùng.'
        : mutation.error.message
      : null

  return (
    <section className="space-y-4">
      <div>
        <h2 className="font-semibold">Trang cá nhân</h2>
        <p className="text-sm text-muted-foreground">
          Email không đổi được ở đây vì nó là địa chỉ dùng để đăng nhập.
        </p>
      </div>

      <form
        className="space-y-4"
        onSubmit={(event) => {
          event.preventDefault()
          mutation.mutate()
        }}
      >
        <div className="flex items-center gap-4 rounded-xl bg-muted/50 p-4">
          {/* Xem trước theo đúng những gì đang gõ, nên dán sai đường dẫn là
              thấy ngay thay vì phải lưu rồi mới biết.

              Previews what is being chosen, so a wrong picture shows up here
              instead of only after saving. */}
          <UserAvatar user={{ ...user, avatar_url: shownAvatar }} className="size-16 text-lg" />

          <div className="min-w-0 flex-1 space-y-2">
            <p className="truncate text-sm font-medium">@{user.username}</p>

            <div className="flex flex-wrap gap-2">
              {/* Input file thật bị ẩn và được kích hoạt từ nút, vì nút của
                  thư viện giao diện không thể là input, mà giao diện mặc định
                  của input file thì không theo được phần còn lại.

                  The real file input is hidden and triggered from the button,
                  because a design-system button cannot be an input, and the
                  browser's own file input cannot be made to match the rest of
                  the interface. */}
              <input
                ref={fileInput}
                type="file"
                accept={ACCEPT}
                className="sr-only"
                onChange={(event) => pick(event.target.files?.[0])}
              />
              <Button type="button" size="sm" onClick={() => fileInput.current?.click()}>
                <Upload />
                Đổi ảnh
              </Button>

              {shownAvatar && (
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  onClick={() => {
                    setFile(null)
                    setClearAvatar(true)
                    setFileError(null)
                    // Chọn lại đúng file vừa bỏ sẽ không bắn onChange nếu giá
                    // trị của input còn nguyên.
                    //
                    // Picking the very same file again fires no onChange
                    // unless the input's value is cleared.
                    if (fileInput.current) fileInput.current.value = ''
                  }}
                >
                  <Trash />
                  Xoá ảnh
                </Button>
              )}
            </div>

            <p className="text-xs text-muted-foreground">
              JPEG, PNG hoặc GIF, tối đa 5 MB. Ảnh sẽ được cắt vuông 256×256.
            </p>
            {fileError && <p className="text-sm text-destructive">{fileError}</p>}
          </div>
        </div>

        {textFields.map(({ name, label, autoComplete }) => (
          <div key={name} className="space-y-1.5">
            <Label htmlFor={`settings-${name}`}>{label}</Label>
            <Input
              id={`settings-${name}`}
              value={form[name]}
              autoComplete={autoComplete}
              onChange={(event) => setForm({ ...form, [name]: event.target.value })}
            />
          </div>
        ))}

        {errorText && <p className="text-sm text-destructive">{errorText}</p>}
        {mutation.isSuccess && !dirty && (
          <p className="text-sm text-muted-foreground">Đã lưu.</p>
        )}

        <Button type="submit" disabled={!dirty || mutation.isPending}>
          {mutation.isPending && <Loader2 className="animate-spin" />}
          {mutation.isPending ? 'Đang lưu' : 'Lưu thay đổi'}
        </Button>
      </form>
    </section>
  )
}
