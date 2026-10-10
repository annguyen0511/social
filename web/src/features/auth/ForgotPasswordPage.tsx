import { useState } from 'react'
import { Link } from 'react-router-dom'
import { useMutation } from '@tanstack/react-query'
import { Loader2, MailCheck } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { post } from '../../api/client'

type Sent = { token?: string }

/**
 * Asking for a password reset link.
 *
 * The confirmation never says whether that address has an account, because
 * the server never says either: a form that answered "no such user" would be
 * a way for anyone to check who is registered here, one address at a time.
 * So the page is written to be true in both cases — "if that address has an
 * account" is not a hedge, it is the whole guarantee.
 *
 * Xin một đường dẫn đặt lại mật khẩu.
 *
 * Màn hình xác nhận không bao giờ nói địa chỉ đó có tài khoản hay không, vì
 * server cũng không nói: một biểu mẫu trả lời "không có người dùng này" sẽ
 * thành công cụ để bất kỳ ai dò xem những ai đã đăng ký ở đây, mỗi lần một
 * địa chỉ. Nên trang này được viết sao cho đúng trong cả hai trường hợp —
 * "nếu địa chỉ đó có tài khoản" không phải là nói tránh, nó chính là điều
 * được bảo đảm.
 */
export function ForgotPasswordPage() {
  const [email, setEmail] = useState('')

  const mutation = useMutation({
    mutationFn: () => post<Sent>('/v1/authentication/forgot-password', { email }),
  })

  return (
    <main className="grid min-h-dvh place-items-center p-6">
      <Card className="w-full max-w-sm">
        <CardContent>
          {mutation.isSuccess ? (
            <div className="space-y-4 text-center">
              <MailCheck className="mx-auto size-10 text-muted-foreground" strokeWidth={1.5} />
              <div>
                <h1 className="mb-1 text-xl font-semibold">Kiểm tra hòm thư</h1>
                <p className="text-sm text-muted-foreground">
                  Nếu <span className="text-foreground">{email}</span> có tài khoản, chúng tôi
                  vừa gửi tới đó một đường dẫn để đặt mật khẩu mới. Đường dẫn chỉ dùng được
                  một lần và sẽ hết hạn.
                </p>
              </div>

              {/* Ở môi trường dev, API trả thẳng token về vì có thể chưa có
                  mail nào thật sự được gửi. Hiện nó ra ở đây để thử luồng
                  không cần hòm thư; production không bao giờ có trường này.

                  In development the API hands the token straight back,
                  because no mail may actually have gone anywhere. Showing it
                  here makes the flow testable without a mailbox; production
                  never sends this field. */}
              {mutation.data?.token && (
                <div className="rounded-lg border border-dashed border-border p-3 text-left">
                  <p className="mb-1 text-xs text-muted-foreground">
                    Chỉ hiện khi chạy dev — mở thẳng đường dẫn:
                  </p>
                  <Link
                    to={`/reset/${mutation.data.token}`}
                    className="text-xs break-all underline"
                  >
                    /reset/{mutation.data.token}
                  </Link>
                </div>
              )}

              <Button asChild variant="outline" className="w-full">
                <Link to="/login">Về trang đăng nhập</Link>
              </Button>
            </div>
          ) : (
            <form
              className="space-y-3"
              onSubmit={(event) => {
                event.preventDefault()
                mutation.mutate()
              }}
            >
              <div>
                <h1 className="text-xl font-semibold">Quên mật khẩu</h1>
                <p className="mt-1 text-sm text-muted-foreground">
                  Nhập email bạn dùng để đăng nhập. Chúng tôi sẽ gửi một đường dẫn đặt lại
                  mật khẩu.
                </p>
              </div>

              <Input
                type="email"
                placeholder="email"
                autoComplete="email"
                required
                value={email}
                onChange={(event) => setEmail(event.target.value)}
              />

              {mutation.isError && (
                <p className="text-sm text-destructive">{mutation.error.message}</p>
              )}

              <Button
                type="submit"
                size="lg"
                disabled={mutation.isPending || email.trim().length === 0}
                className="w-full"
              >
                {mutation.isPending && <Loader2 className="animate-spin" />}
                {mutation.isPending ? 'Đang gửi' : 'Gửi đường dẫn'}
              </Button>

              <p className="text-center text-sm text-muted-foreground">
                Nhớ ra rồi?{' '}
                <Link to="/login" className="underline">
                  Đăng nhập
                </Link>
              </p>
            </form>
          )}
        </CardContent>
      </Card>
    </main>
  )
}
