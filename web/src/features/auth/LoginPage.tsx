import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { useMutation } from '@tanstack/react-query'
import { Loader2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { HttpError, post } from '../../api/client'
import type { User } from '../../api/types'

export function LoginPage() {
  const [form, setForm] = useState({ email: '', password: '' })
  const navigate = useNavigate()

  // The response carries the user; the session itself arrives as an HttpOnly
  // cookie that JavaScript cannot read, which is the point.
  //
  // Response trả về thông tin user; còn phiên đăng nhập nằm trong cookie
  // HttpOnly mà JavaScript không đọc được — đó chính là chủ đích.
  const mutation = useMutation({
    mutationFn: () => post<User>('/v1/authentication/login', form),
    onSuccess: () => navigate('/'),
  })

  const errorText =
    mutation.error instanceof HttpError
      ? mutation.error.status === 403
        ? 'Tài khoản chưa kích hoạt. Hãy mở liên kết trong email.'
        : mutation.error.status === 401
          ? 'Email hoặc mật khẩu không đúng.'
          : mutation.error.message
      : null

  return (
    <main className="grid min-h-dvh place-items-center p-6">
      <Card className="w-full max-w-sm">
        <CardContent>
          <form
            className="space-y-3"
            onSubmit={(e) => {
              e.preventDefault()
              mutation.mutate()
            }}
          >
            <h1 className="mb-2 text-xl font-semibold">Đăng nhập</h1>

            <Input
              type="email"
              placeholder="email"
              autoComplete="email"
              value={form.email}
              onChange={(e) => setForm({ ...form, email: e.target.value })}
            />
            <Input
              type="password"
              placeholder="mật khẩu"
              autoComplete="current-password"
              value={form.password}
              onChange={(e) => setForm({ ...form, password: e.target.value })}
            />

            {/* Ngay dưới ô mật khẩu, không nhét xuống cuối thẻ: người cần
                nó là người vừa gõ sai mật khẩu, và họ đang nhìn đúng chỗ
                này.

                Right under the password box rather than tucked at the
                bottom: whoever needs it has just mistyped a password, and
                this is where they are already looking. */}
            <p className="text-right">
              <Link
                to="/forgot-password"
                className="text-sm text-muted-foreground underline underline-offset-2 hover:text-foreground"
              >
                Quên mật khẩu?
              </Link>
            </p>

            {errorText && <p className="text-sm text-destructive">{errorText}</p>}

            <Button type="submit" size="lg" disabled={mutation.isPending} className="w-full">
              {mutation.isPending && <Loader2 className="animate-spin" />}
              {mutation.isPending ? 'Đang vào' : 'Đăng nhập'}
            </Button>

            <p className="text-center text-sm text-muted-foreground">
              Chưa có tài khoản?{' '}
              <Link to="/register" className="underline">
                Đăng ký
              </Link>
            </p>
          </form>
        </CardContent>
      </Card>
    </main>
  )
}
