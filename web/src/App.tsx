import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom'
import { ConfirmPage } from './features/auth/ConfirmPage'
import { LoginPage } from './features/auth/LoginPage'
import { RegisterPage } from './features/auth/RegisterPage'
import { FeedPage } from './features/feed/FeedPage'
import { ProfilePage } from './features/users/ProfilePage'

export function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<FeedPage />} />
        <Route path="/users/:userID" element={<ProfilePage />} />
        <Route path="/login" element={<LoginPage />} />
        <Route path="/register" element={<RegisterPage />} />
        {/* Phải khớp với FRONTEND_URL của API: {FRONTEND_URL}/confirm/{token} */}
        <Route path="/confirm/:token" element={<ConfirmPage />} />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </BrowserRouter>
  )
}
