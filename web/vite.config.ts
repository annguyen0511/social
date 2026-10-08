import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: {
    port: 5173,
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
