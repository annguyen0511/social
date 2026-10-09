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
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { HttpError, patch } from '../../api/client'
import type { ProfileUpdate, User } from '../../api/types'
import { UserAvatar } from './UserAvatar'

type Props = {
  user: User
  open: boolean
  onOpenChange: (open: boolean) => void
}

const fields = [
  { name: 'first_name', label: 'Tên', autoComplete: 'given-name' },
  { name: 'last_name', label: 'Họ', autoComplete: 'family-name' },
  { name: 'username', label: 'Tên đăng nhập', autoComplete: 'username' },
  { name: 'avatar_url', label: 'Đường dẫn ảnh đại diện', autoComplete: 'off' },
] as const

type Form = Record<(typeof fields)[number]['name'], string>

const toForm = (user: User): Form => ({
  first_name: user.first_name,
  last_name: user.last_name,
  username: user.username,
  avatar_url: user.avatar_url,
})

/**
 * Edits the signed-in user's own profile.
 *
 * Only the fields that actually changed are sent. The API treats an omitted
 * field as "keep the current value", so sending the whole form would overwrite
 * with values this dialog read earlier — and lose anything changed meanwhile
 * in another tab.
 *
 * Sửa profile của chính người đang đăng nhập.
 *
 * Chỉ gửi những field thật sự đổi. API coi field không gửi là "giữ nguyên giá
 * trị hiện tại", nên gửi cả form sẽ ghi đè bằng những giá trị mà dialog này
 * đọc từ trước — và làm mất thứ vừa được đổi ở tab khác.
 */
export function EditProfileDialog({ user, open, onOpenChange }: Props) {
  const [form, setForm] = useState<Form>(() => toForm(user))
  const queryClient = useQueryClient()

  // Reopening after a cancel must not show the abandoned edits, and the form
  // has to pick up a profile changed elsewhere.
  //
  // Mở lại sau khi huỷ thì không được hiện những thay đổi đã bỏ dở, và form
  // phải lấy theo profile nếu nó đã đổi ở nơi khác.
  useEffect(() => {
    if (open) setForm(toForm(user))
  }, [open, user])

  const mutation = useMutation({
    mutationFn: () => {
      const changed: ProfileUpdate = {}
      for (const { name } of fields) {
        if (form[name] !== toForm(user)[name]) changed[name] = form[name]
      }
      return patch<User>('/v1/users/me', changed)
    },
    onSuccess: (updated) => {
      // The username shows up on the profile, on every feed post and in search
      // results, so all three are stale once it changes.
      //
      // Username xuất hiện trên profile, trên mọi bài trong feed và trong kết
      // quả tìm kiếm, nên cả ba đều cũ ngay khi nó đổi.
      queryClient.setQueryData(['users', 'me'], updated)
      queryClient.invalidateQueries({ queryKey: ['users'] })
      queryClient.invalidateQueries({ queryKey: ['feed'] })
      onOpenChange(false)
    },
  })

  const errorText =
    mutation.error instanceof HttpError
      ? mutation.error.status === 409
        ? 'Tên đăng nhập này đã có người dùng.'
        : mutation.error.message
      : null

  const dirty = fields.some(({ name }) => form[name] !== toForm(user)[name])

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Chỉnh sửa trang cá nhân</DialogTitle>
          <DialogDescription>
            Email không đổi được ở đây vì nó là địa chỉ dùng để đăng nhập.
          </DialogDescription>
        </DialogHeader>

        <form
          id="edit-profile"
          className="space-y-4"
          onSubmit={(event) => {
            event.preventDefault()
            mutation.mutate()
          }}
        >
          <div className="flex items-center gap-3">
            {/* Xem trước theo đúng những gì đang gõ, nên dán sai đường dẫn là
                thấy ngay thay vì phải lưu rồi mới biết.

                Previews what is being typed, so a wrong URL shows up here
                instead of only after saving. */}
            <UserAvatar user={{ ...user, avatar_url: form.avatar_url }} className="size-12" />
            <p className="text-sm text-muted-foreground">
              Để trống đường dẫn ảnh để quay về chữ cái đầu của tên.
            </p>
          </div>

          {fields.map(({ name, label, autoComplete }) => (
            <div key={name} className="space-y-1.5">
              <Label htmlFor={name}>{label}</Label>
              <Input
                id={name}
                value={form[name]}
                autoComplete={autoComplete}
                placeholder={name === 'avatar_url' ? 'https://…' : undefined}
                onChange={(event) => setForm({ ...form, [name]: event.target.value })}
              />
            </div>
          ))}

          {errorText && <p className="text-sm text-destructive">{errorText}</p>}
        </form>

        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            Huỷ
          </Button>
          <Button type="submit" form="edit-profile" disabled={!dirty || mutation.isPending}>
            {mutation.isPending && <Loader2 className="animate-spin" />}
            {mutation.isPending ? 'Đang lưu' : 'Lưu'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
