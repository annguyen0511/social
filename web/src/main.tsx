import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { App } from './App'
import { HttpError } from './api/client'
import './index.css'

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      // Retrying a 401 or a 404 just delays the error the user needs to see.
      // Thử lại 401 hay 404 chỉ làm chậm thông báo lỗi mà người dùng cần thấy.
      retry: (failureCount, error) => {
        if (error instanceof HttpError && error.status >= 400 && error.status < 500) return false
        return failureCount < 2
      },
    },
  },
})

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <App />
    </QueryClientProvider>
  </StrictMode>,
)
