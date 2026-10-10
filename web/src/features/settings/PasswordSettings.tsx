import { useState } from 'react'
import { useMutation } from '@tanstack/react-query'
import { Loader2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { HttpError, put } from '../../api/client'

// Khớp với validate:"min=8" của changePasswordRequest trong cmd/api/users.go.
// Mirrors validate:"min=8" on changePasswordRequest in cmd/api/users.go.
const MIN_LENGTH = 8

const empty = { current: '', next: '', confirm: '' }

/**
 * Changing your own password, from inside the app.
 *
 * The current password is asked for even though the session already proves
 * who this is: a session only proves the browser was left signed in, and an
 * unattended screen should not be enough to take an account away from its
 * owner.
 *
 * Confirming the new password is this form's own idea — the server does not
 * ask for it. A typo in a field nobody can read would otherwise lock the
 * person out of their own account, with every other session ended at the
 * same moment.
 *
 * Đổi mật khẩu của chính mình, từ bên trong ứng dụng.
 *
 * Vẫn phải hỏi mật khẩu hiện tại dù phiên đăng nhập đã chứng minh đây là ai:
 * một phiên chỉ chứng minh trình duyệt được để nguyên trạng thái đã đăng
 * nhập, mà một màn hình bỏ quên thì không đủ để lấy mất tài khoản khỏi tay
 * chủ nó.
 *
 * Việc nhập lại mật khẩu mới là ý của riêng biểu mẫu này — server không đòi.
 * Thiếu nó, một lỗi gõ nhầm trong ô không ai đọc được sẽ khoá người dùng ra
 * khỏi chính tài khoản của họ, đúng lúc mọi phiên khác cũng vừa bị kết thúc.
 */
export function PasswordSettings() {
  const [form, setForm] = useState(empty)

  const mutation = useMutation({
    mutationFn: () =>
      put('/v1/users/me/password', {
        current_password: form.current,
        new_password: form.next,
      }),
    // Xoá sạch các ô sau khi đổi: để mật khẩu nằm lại trong form đã đóng
    // chẳng được gì, mà lần mở sau sẽ thấy ô "mật khẩu hiện tại" chứa một giá
    // trị giờ đã sai.
    //
    // Cleared after a change: leaving passwords sitting in a closed form
    // gains nothing, and the next time it opens the "current password" box
    // would hold a value that is now wrong.
    onSuccess: () => setForm(empty),
  })

  const tooShort = form.next.length > 0 && form.next.length < MIN_LENGTH
  const mismatch = form.confirm.length > 0 && form.next !== form.confirm
  const ready =
    form.current.length > 0 && form.next.length >= MIN_LENGTH && form.next === form.confirm

  const errorText =
    mutation.error instanceof HttpError
      ? mutation.error.status === 401
        ? 'Mật khẩu hiện tại không đúng.'
        : mutation.error.message
      : null

  const fields = [
    {
      key: 'current' as const,
      label: 'Mật khẩu hiện tại',
      autoComplete: 'current-password',
      hint: null,
    },
    {
      key: 'next' as const,
      label: 'Mật khẩu mới',
      autoComplete: 'new-password',
      hint: tooShort ? `Mật khẩu phải có ít nhất ${MIN_LENGTH} ký tự.` : null,
    },
    {
      key: 'confirm' as const,
      label: 'Nhập lại mật khẩu mới',
      autoComplete: 'new-password',
      hint: mismatch ? 'Hai ô mật khẩu mới chưa khớp nhau.' : null,
    },
  ]

  return (
    <section className="space-y-4">
      <div>
        <h2 className="font-semibold">Mật khẩu</h2>
        <p className="text-sm text-muted-foreground">
          Đổi mật khẩu sẽ đăng xuất tài khoản này khỏi mọi thiết bị khác. Trình duyệt bạn
          đang dùng thì vẫn ở lại.
        </p>
      </div>

      <form
        className="space-y-4"
        onSubmit={(event) => {
          event.preventDefault()
          mutation.mutate()
        }}
      >
        {fields.map(({ key, label, autoComplete, hint }) => (
          <div key={key} className="space-y-1.5">
            <Label htmlFor={`password-${key}`}>{label}</Label>
            <Input
              id={`password-${key}`}
              type="password"
              autoComplete={autoComplete}
              value={form[key]}
              onChange={(event) => setForm({ ...form, [key]: event.target.value })}
            />
            {hint && <p className="text-xs text-destructive">{hint}</p>}
          </div>
        ))}

        {errorText && <p className="text-sm text-destructive">{errorText}</p>}
        {mutation.isSuccess && (
          <p className="text-sm text-muted-foreground">
            Đã đổi mật khẩu. Các thiết bị khác đã bị đăng xuất.
          </p>
        )}

        <Button type="submit" disabled={!ready || mutation.isPending}>
          {mutation.isPending && <Loader2 className="animate-spin" />}
          {mutation.isPending ? 'Đang đổi' : 'Đổi mật khẩu'}
        </Button>
      </form>
    </section>
  )
}
