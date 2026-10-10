import { useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { useMutation } from '@tanstack/react-query'
import { CheckCircle2, Loader2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { HttpError, post } from '../../api/client'

// Khớp với validate:"min=8" của resetPasswordRequest trong cmd/api/auth.go.
// Mirrors validate:"min=8" on resetPasswordRequest in cmd/api/auth.go.
const MIN_LENGTH = 8

/**
 * The page the reset email links to: choose a new password.
 *
 * Unlike the activation page this does not fire on mount. The token here does
 * not merely confirm something already decided — it is the one chance to set
 * a password, and spending it the instant the link is opened would leave
 * whoever clicked it with an account whose password they do not know. So the
 * token waits in the URL until there is something to spend it on.
 *
 * Success does not sign anyone in. A link sitting in a mailbox is weaker
 * proof of ownership than a password, so it buys the right to set one and
 * nothing more; the new password is then typed on the login page like any
 * other.
 *
 * Trang mà email đặt lại mật khẩu trỏ tới: chọn mật khẩu mới.
 *
 * Khác với trang kích hoạt, trang này không tự gọi khi vừa mở. Token ở đây
 * không chỉ xác nhận một việc đã quyết — nó là cơ hội duy nhất để đặt mật
 * khẩu, mà tiêu nó ngay lúc mở link sẽ để người vừa bấm vào với một tài khoản
 * có mật khẩu mà chính họ không biết. Nên token nằm chờ trên URL cho tới khi
 * có thứ để tiêu nó vào.
 *
 * Thành công không đăng nhập hộ ai cả. Một đường dẫn nằm trong hòm thư là
 * bằng chứng sở hữu yếu hơn mật khẩu, nên nó đổi được quyền đặt mật khẩu chứ
 * không hơn; mật khẩu mới sau đó được gõ ở trang đăng nhập như bình thường.
 */
export function ResetPasswordPage() {
  const { token } = useParams<{ token: string }>()
  const navigate = useNavigate()
  const [form, setForm] = useState({ next: '', confirm: '' })

  const mutation = useMutation({
    mutationFn: () =>
      post('/v1/authentication/reset-password', { token, password: form.next }),
  })

  const tooShort = form.next.length > 0 && form.next.length < MIN_LENGTH
  const mismatch = form.confirm.length > 0 && form.next !== form.confirm
  const ready = form.next.length >= MIN_LENGTH && form.next === form.confirm

  const errorText =
    mutation.error instanceof HttpError
      ? mutation.error.status === 404
        ? 'Đường dẫn không hợp lệ, đã dùng rồi hoặc đã hết hạn. Hãy xin một đường dẫn mới.'
        : mutation.error.message
      : null

  if (mutation.isSuccess) {
    return (
      <main className="grid min-h-dvh place-items-center p-6">
        <Card className="w-full max-w-sm">
          <CardContent className="space-y-4 text-center">
            <CheckCircle2 className="mx-auto size-10 text-muted-foreground" strokeWidth={1.5} />
            <div>
              <h1 className="mb-1 text-xl font-semibold">Đã đổi mật khẩu</h1>
              <p className="text-sm text-muted-foreground">
                Tài khoản của bạn đã được đăng xuất khỏi mọi thiết bị. Đăng nhập lại bằng
                mật khẩu mới.
              </p>
            </div>
            <Button size="lg" className="w-full" onClick={() => navigate('/login')}>
              Đăng nhập
            </Button>
          </CardContent>
        </Card>
      </main>
    )
  }

  const fields = [
    {
      key: 'next' as const,
      label: 'Mật khẩu mới',
      hint: tooShort ? `Mật khẩu phải có ít nhất ${MIN_LENGTH} ký tự.` : null,
    },
    {
      key: 'confirm' as const,
      label: 'Nhập lại mật khẩu mới',
      hint: mismatch ? 'Hai ô mật khẩu chưa khớp nhau.' : null,
    },
  ]

  return (
    <main className="grid min-h-dvh place-items-center p-6">
      <Card className="w-full max-w-sm">
        <CardContent>
          <form
            className="space-y-3"
            onSubmit={(event) => {
              event.preventDefault()
              mutation.mutate()
            }}
          >
            <div>
              <h1 className="text-xl font-semibold">Đặt mật khẩu mới</h1>
              <p className="mt-1 text-sm text-muted-foreground">
                Sau khi đổi, tài khoản sẽ bị đăng xuất khỏi mọi thiết bị.
              </p>
            </div>

            {fields.map(({ key, label, hint }) => (
              <div key={key} className="space-y-1.5">
                <Label htmlFor={`reset-${key}`}>{label}</Label>
                <Input
                  id={`reset-${key}`}
                  type="password"
                  autoComplete="new-password"
                  value={form[key]}
                  onChange={(event) => setForm({ ...form, [key]: event.target.value })}
                />
                {hint && <p className="text-xs text-destructive">{hint}</p>}
              </div>
            ))}

            {errorText && <p className="text-sm text-destructive">{errorText}</p>}

            <Button
              type="submit"
              size="lg"
              disabled={!ready || mutation.isPending}
              className="w-full"
            >
              {mutation.isPending && <Loader2 className="animate-spin" />}
              {mutation.isPending ? 'Đang đổi' : 'Đổi mật khẩu'}
            </Button>

            <p className="text-center text-sm text-muted-foreground">
              <Link to="/forgot-password" className="underline">
                Xin đường dẫn khác
              </Link>
            </p>
          </form>
        </CardContent>
      </Card>
    </main>
  )
}
