import { useEffect, useRef, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { Loader2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { HttpError, put } from '../../api/client'

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
    <main className="grid min-h-dvh place-items-center p-6">
      <Card className="w-full max-w-sm">
        <CardContent className="text-center">
          {state.kind === 'working' && (
            <p className="flex items-center justify-center gap-2 text-muted-foreground">
              <Loader2 className="size-4 animate-spin" />
              Đang kích hoạt tài khoản…
            </p>
          )}

          {state.kind === 'done' && (
            <>
              <h1 className="mb-2 text-xl font-semibold">Kích hoạt thành công</h1>
              <p className="mb-6 text-muted-foreground">Bạn có thể đăng nhập ngay bây giờ.</p>
              <Button asChild size="lg">
                <Link to="/login">Đăng nhập</Link>
              </Button>
            </>
          )}

          {state.kind === 'failed' && (
            <>
              <h1 className="mb-2 text-xl font-semibold">Không kích hoạt được</h1>
              <p className="mb-6 text-muted-foreground">{state.message}</p>
              <Button asChild size="lg" variant="outline">
                <Link to="/register">Đăng ký lại</Link>
              </Button>
            </>
          )}
        </CardContent>
      </Card>
    </main>
  )
}
