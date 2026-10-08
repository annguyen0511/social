import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { useMutation } from '@tanstack/react-query'
import { HttpError, post } from '../../api/client'
import type { User } from '../../api/types'

const field = 'w-full rounded-lg border border-stone-300 px-3 py-2 outline-none focus:border-stone-900'

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
    <main className="min-h-dvh grid place-items-center bg-stone-100 p-6">
      <form
        className="w-full max-w-sm space-y-3 rounded-xl bg-white p-8 shadow-sm"
        onSubmit={(e) => {
          e.preventDefault()
          mutation.mutate()
        }}
      >
        <h1 className="mb-2 text-xl font-semibold">Đăng nhập</h1>

        <input
          className={field}
          type="email"
          placeholder="email"
          value={form.email}
          onChange={(e) => setForm({ ...form, email: e.target.value })}
        />
        <input
          className={field}
          type="password"
          placeholder="mật khẩu"
          value={form.password}
          onChange={(e) => setForm({ ...form, password: e.target.value })}
        />

        {errorText && <p className="text-sm text-red-600">{errorText}</p>}

        <button
          type="submit"
          disabled={mutation.isPending}
          className="w-full rounded-lg bg-stone-900 py-2.5 font-medium text-white disabled:opacity-40"
        >
          {mutation.isPending ? 'Đang vào…' : 'Đăng nhập'}
        </button>

        <p className="text-center text-sm text-stone-500">
          Chưa có tài khoản? <Link to="/register" className="underline">Đăng ký</Link>
        </p>
      </form>
    </main>
  )
}
