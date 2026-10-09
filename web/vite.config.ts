import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

// The API builds activation links from FRONTEND_URL, and Vite has to accept
// requests arriving under that same hostname. Deriving one from the other
// keeps a tunnel working after a single change in .envrc.
//
// API dựng link kích hoạt từ FRONTEND_URL, còn Vite thì phải chấp nhận request
// đi vào dưới đúng tên miền đó. Suy cái này ra từ cái kia để đổi tunnel chỉ
// cần sửa một chỗ duy nhất trong .envrc.
const frontendHost = (() => {
  if (!process.env.FRONTEND_URL) return []
  try {
    return [new URL(process.env.FRONTEND_URL).hostname]
  } catch {
    return []
  }
})()

export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: {
    port: 5173,
    // Vite 6 refuses requests whose Host header it does not recognise, which
    // blocks every tunnel. The leading dot covers quick tunnels, whose URL is
    // random each run; a named tunnel adds its own hostname from FRONTEND_URL.
    // Listing them beats allowedHosts: true, which would accept any Host and
    // open the dev server to DNS rebinding.
    //
    // Vite 6 từ chối request có Host lạ, nên mọi tunnel đều bị chặn. Dấu chấm
    // ở đầu bao quát tunnel tạm vốn có URL ngẫu nhiên mỗi lần; tunnel đặt tên
    // thì tự thêm tên miền của nó từ FRONTEND_URL. Liệt kê ra như vậy tốt hơn
    // allowedHosts: true — đặt true là nhận mọi Host và mở cửa cho tấn công
    // DNS rebinding.
    allowedHosts: ['.trycloudflare.com', ...frontendHost],
    // The Go API runs on another port. Proxying /v1 makes the browser see one
    // origin, which avoids CORS entirely in development and lets the session
    // cookie behave as a first-party cookie.
    //
    // API Go chạy ở cổng khác. Proxy /v1 khiến trình duyệt thấy mọi thứ cùng
    // một origin, nên lúc dev không vướng CORS và cookie phiên được coi là
    // cookie first-party.
    proxy: {
      '/v1': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
})
