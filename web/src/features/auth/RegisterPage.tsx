import { useState } from 'react'
import { Link } from 'react-router-dom'
import { useMutation } from '@tanstack/react-query'
import { Loader2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { HttpError, post } from '../../api/client'
import type { RegisteredUser } from '../../api/types'

const fields = [
  { name: 'first_name', label: 'Tên', type: 'text', autoComplete: 'given-name' },
  { name: 'last_name', label: 'Họ', type: 'text', autoComplete: 'family-name' },
  { name: 'username', label: 'Tên đăng nhập', type: 'text', autoComplete: 'username' },
  { name: 'email', label: 'Email', type: 'email', autoComplete: 'email' },
  { name: 'password', label: 'Mật khẩu', type: 'password', autoComplete: 'new-password' },
] as const

export function RegisterPage() {
  const [form, setForm] = useState({
    first_name: '',
    last_name: '',
    username: '',
    email: '',
    password: '',
  })

  const mutation = useMutation({
    mutationFn: () => post<RegisteredUser>('/v1/authentication/register', form),
  })

  // Mirrors the rules in createUserRequest on the Go side. Keeping them here
  // too only saves a round trip; the server stays the authority.
  //
  // Phản chiếu luật trong createUserRequest phía Go. Để ở đây chỉ để đỡ một
  // lượt gọi mạng; server vẫn là nơi quyết định.
  const invalid =
    form.first_name.length < 2 ||
    form.last_name.length < 2 ||
    form.username.length < 2 ||
    !form.email.includes('@') ||
    form.password.length < 8

  const errorText =
    mutation.error instanceof HttpError
      ? mutation.error.status === 409
        ? 'Email hoặc tên đăng nhập đã có người dùng.'
        : mutation.error.message
      : null

  if (mutation.isSuccess) {
    return (
      <main className="grid min-h-dvh place-items-center p-6">
        <Card className="w-full max-w-sm">
          <CardContent>
            <h1 className="mb-2 text-xl font-semibold">Kiểm tra email của bạn</h1>
            <p className="text-muted-foreground">
              Chúng tôi đã gửi liên kết kích hoạt tới {form.email}.
            </p>

            {/* Chỉ có khi API chạy ngoài production, dùng để thử luồng mà không
                cần hòm thư thật.

                Only present when the API runs outside production, so the flow
                can be exercised without a real mailbox. */}
            {mutation.data.token && (
              <Button asChild variant="secondary" className="mt-4 w-full">
                <Link to={`/confirm/${mutation.data.token}`}>
                  Chế độ dev: kích hoạt ngay
                </Link>
              </Button>
            )}
          </CardContent>
        </Card>
      </main>
    )
  }

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
            <h1 className="mb-2 text-xl font-semibold">Tạo tài khoản</h1>

            {fields.map((field) => (
              <Input
                key={field.name}
                type={field.type}
                placeholder={field.label}
                autoComplete={field.autoComplete}
                value={form[field.name]}
                onChange={(e) => setForm({ ...form, [field.name]: e.target.value })}
              />
            ))}

            {errorText && <p className="text-sm text-destructive">{errorText}</p>}

            <Button
              type="submit"
              size="lg"
              disabled={invalid || mutation.isPending}
              className="w-full"
            >
              {mutation.isPending && <Loader2 className="animate-spin" />}
              {mutation.isPending ? 'Đang gửi' : 'Đăng ký'}
            </Button>

            <p className="text-center text-sm text-muted-foreground">
              Đã có tài khoản?{' '}
              <Link to="/login" className="underline">
                Đăng nhập
              </Link>
            </p>
          </form>
        </CardContent>
      </Card>
    </main>
  )
}
