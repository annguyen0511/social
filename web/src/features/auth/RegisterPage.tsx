import { useState } from 'react'
import { Link } from 'react-router-dom'
import { useMutation } from '@tanstack/react-query'
import { HttpError, post } from '../../api/client'
import type { RegisteredUser } from '../../api/types'

const field = 'w-full rounded-lg border border-stone-300 px-3 py-2 outline-none focus:border-stone-900'

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
      <main className="min-h-dvh grid place-items-center bg-stone-100 p-6">
        <div className="w-full max-w-sm rounded-xl bg-white p-8 shadow-sm">
          <h1 className="mb-2 text-xl font-semibold">Kiểm tra email của bạn</h1>
          <p className="text-stone-600">Chúng tôi đã gửi liên kết kích hoạt tới {form.email}.</p>

          {/* Chỉ có khi API chạy ngoài production, dùng để thử luồng mà không cần hòm thư thật. */}
          {mutation.data.token && (
            <Link
              to={`/confirm/${mutation.data.token}`}
              className="mt-4 block break-all rounded-lg bg-stone-100 p-3 text-sm text-stone-700"
            >
              Chế độ dev: bấm vào đây để kích hoạt ngay
            </Link>
          )}
        </div>
      </main>
    )
  }

  return (
    <main className="min-h-dvh grid place-items-center bg-stone-100 p-6">
      <form
        className="w-full max-w-sm space-y-3 rounded-xl bg-white p-8 shadow-sm"
        onSubmit={(e) => {
          e.preventDefault()
          mutation.mutate()
        }}
      >
        <h1 className="mb-2 text-xl font-semibold">Tạo tài khoản</h1>

        {(['first_name', 'last_name', 'username', 'email', 'password'] as const).map((name) => (
          <input
            key={name}
            className={field}
            type={name === 'password' ? 'password' : name === 'email' ? 'email' : 'text'}
            placeholder={name}
            value={form[name]}
            onChange={(e) => setForm({ ...form, [name]: e.target.value })}
          />
        ))}

        {errorText && <p className="text-sm text-red-600">{errorText}</p>}

        <button
          type="submit"
          disabled={invalid || mutation.isPending}
          className="w-full rounded-lg bg-stone-900 py-2.5 font-medium text-white disabled:opacity-40"
        >
          {mutation.isPending ? 'Đang gửi…' : 'Đăng ký'}
        </button>

        <p className="text-center text-sm text-stone-500">
          Đã có tài khoản? <Link to="/login" className="underline">Đăng nhập</Link>
        </p>
      </form>
    </main>
  )
}
