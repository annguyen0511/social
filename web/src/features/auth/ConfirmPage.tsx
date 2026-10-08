import { useEffect, useRef, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { put } from '../../api/client'
import { HttpError } from '../../api/client'

type State = { kind: 'working' } | { kind: 'done' } | { kind: 'failed'; message: string }

/**
 * The page the activation email links to. The API endpoint is a PUT, which a
 * mail client cannot issue by following a link, so this page makes the call
 * on the user's behalf.
 *
 * Trang mà email kích hoạt trỏ tới. Endpoint của API là PUT, trình đọc mail
 * bấm link chỉ gửi được GET, nên trang này gọi thay cho người dùng.
 */
export function ConfirmPage() {
  const { token } = useParams<{ token: string }>()
  const [state, setState] = useState<State>({ kind: 'working' })

  // React 19 in development mounts effects twice. The token is consumed on
  // first use, so the second call would always report "not found" and
  // overwrite a successful result. The ref makes the call happen once.
  //
  // React 19 lúc dev chạy effect hai lần. Token bị tiêu huỷ ngay lần dùng đầu,
  // nên lần gọi thứ hai luôn báo "không tìm thấy" và ghi đè lên kết quả thành
  // công. Dùng ref để chỉ gọi đúng một lần.
  const called = useRef(false)

  useEffect(() => {
    if (!token || called.current) return
    called.current = true

    put(`/v1/authentication/activate/${token}`)
      .then(() => setState({ kind: 'done' }))
      .catch((err: unknown) => {
        const message =
          err instanceof HttpError && err.status === 404
            ? 'Liên kết không hợp lệ, đã dùng rồi hoặc đã hết hạn.'
            : 'Không kích hoạt được. Thử lại sau ít phút.'
        setState({ kind: 'failed', message })
      })
  }, [token])

  return (
    <main className="min-h-dvh grid place-items-center bg-stone-100 p-6">
      <div className="w-full max-w-sm rounded-xl bg-white p-8 text-center shadow-sm">
        {state.kind === 'working' && <p className="text-stone-600">Đang kích hoạt tài khoản…</p>}

        {state.kind === 'done' && (
          <>
            <h1 className="mb-2 text-xl font-semibold">Kích hoạt thành công</h1>
            <p className="mb-6 text-stone-600">Bạn có thể đăng nhập ngay bây giờ.</p>
            <Link to="/login" className="inline-block rounded-lg bg-stone-900 px-5 py-2.5 font-medium text-white">
              Đăng nhập
            </Link>
          </>
        )}

        {state.kind === 'failed' && (
          <>
            <h1 className="mb-2 text-xl font-semibold">Không kích hoạt được</h1>
            <p className="mb-6 text-stone-600">{state.message}</p>
            <Link to="/register" className="inline-block rounded-lg bg-stone-900 px-5 py-2.5 font-medium text-white">
              Đăng ký lại
            </Link>
          </>
        )}
      </div>
    </main>
  )
}
