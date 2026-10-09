import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom'
import { ConfirmPage } from './features/auth/ConfirmPage'
import { LoginPage } from './features/auth/LoginPage'
import { RegisterPage } from './features/auth/RegisterPage'
import { FeedPage } from './features/feed/FeedPage'
import { AppLayout } from './features/layout/AppLayout'
import { ProfilePage } from './features/users/ProfilePage'

export function App() {
  return (
    <BrowserRouter>
      <Routes>
        {/* Các trang cần đăng nhập dùng chung một khung: thanh header mount
            một lần và sống qua các lần chuyển trang.

            The signed-in pages share one frame: the header mounts once and
            survives navigation between them. */}
        <Route element={<AppLayout />}>
          <Route path="/" element={<FeedPage />} />
          <Route path="/users/:userID" element={<ProfilePage />} />
        </Route>
        <Route path="/login" element={<LoginPage />} />
        <Route path="/register" element={<RegisterPage />} />
        {/* Phải khớp với FRONTEND_URL của API: {FRONTEND_URL}/confirm/{token} */}
        <Route path="/confirm/:token" element={<ConfirmPage />} />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </BrowserRouter>
  )
}
