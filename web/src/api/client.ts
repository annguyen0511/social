import type { ApiError, Envelope } from './types'

// Dev goes through the Vite proxy, so this stays a same-origin path. In
// production point VITE_API_URL at the API's real origin.
//
// Lúc dev đi qua proxy của Vite nên đây là đường dẫn cùng origin. Khi lên
// production thì trỏ VITE_API_URL tới origin thật của API.
const BASE = import.meta.env.VITE_API_URL ?? ''

export class HttpError extends Error {
  constructor(
    readonly status: number,
    message: string,
  ) {
    super(message)
  }
}

/**
 * Calls the API and returns the payload with the envelope removed.
 *
 * Every endpoint wraps success in {status, data, is_success, message} and
 * failure in {error, message}, so unwrapping belongs here and nowhere else.
 *
 * Gọi API và trả về phần dữ liệu đã bóc lớp vỏ envelope.
 *
 * Mọi endpoint đều bọc kết quả thành công trong {status, data, is_success,
 * message} và lỗi trong {error, message}, nên việc bóc vỏ chỉ nên nằm ở đây.
 */
export async function api<T>(path: string, init: RequestInit = {}): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    ...init,
    // Sends and stores the HttpOnly session cookie. Without this the browser
    // drops it on cross-origin calls and every protected route answers 401.
    //
    // Gửi và nhận cookie phiên HttpOnly. Thiếu nó thì trình duyệt bỏ cookie
    // khi gọi khác origin và mọi route cần đăng nhập đều trả 401.
    credentials: 'include',
    headers: {
      ...(init.body ? { 'Content-Type': 'application/json' } : {}),
      ...init.headers,
    },
  })

  const text = await res.text()
  const body = text ? JSON.parse(text) : null

  if (!res.ok) {
    const message = (body as ApiError | null)?.message ?? res.statusText
    throw new HttpError(res.status, message)
  }

  // Two endpoints answer with a bare object instead of the envelope:
  // POST /posts/{id}/comment and GET /health. Detecting the envelope by its
  // is_success field keeps those working without a special case per caller.
  //
  // Có hai endpoint trả object trần thay vì envelope:
  // POST /posts/{id}/comment và GET /health. Nhận diện envelope qua trường
  // is_success giúp hai chỗ đó vẫn chạy mà không phải xử lý riêng ở từng nơi.
  if (body && typeof body === 'object' && 'is_success' in body) {
    return (body as Envelope<T>).data
  }
  return body as T
}

/**
 * Reads. The signal comes from queryFn: passing it through lets Query abort a
 * request whose answer nobody wants any more — the component unmounted, or the
 * queryKey changed because the filter moved on. Without it a slow response can
 * still arrive late and overwrite a newer one.
 *
 * Đọc dữ liệu. signal đến từ queryFn: chuyển tiếp nó xuống giúp Query huỷ một
 * request mà không ai cần kết quả nữa — component đã rời màn hình, hoặc
 * queryKey đã đổi vì người dùng chuyển bộ lọc. Thiếu nó thì một response chậm
 * vẫn về muộn và ghi đè lên kết quả mới hơn.
 */
export const get = <T>(path: string, signal?: AbortSignal) => api<T>(path, { signal })
export const post = <T>(path: string, data?: unknown) =>
  api<T>(path, { method: 'POST', body: data === undefined ? undefined : JSON.stringify(data) })
export const put = <T>(path: string, data?: unknown) =>
  api<T>(path, { method: 'PUT', body: data === undefined ? undefined : JSON.stringify(data) })
