import { Routes, Route, Navigate, Outlet } from 'react-router-dom'
import { useEffect } from 'react'
import { useAuthStore } from './store/auth'
import MainLayout from './layouts/MainLayout'
import LoginPage from './pages/LoginPage'
import RegisterPage from './pages/RegisterPage'
import KBListPage from './pages/KBListPage'
import DocumentsPage from './pages/DocumentsPage'
import QAPage from './pages/QAPage'
import DashboardPage from './pages/DashboardPage'

// App owns the route table. A tiny RequireAuth wrapper redirects unauthenticated
// users to /login; the layout is applied to all authenticated routes so the
// nav shell stays consistent across pages.
export default function App() {
  const restore = useAuthStore((s) => s.restore)
  // On first mount, validate the stored token (if any) so a stale session is
  // cleared and the user is asked to log in again.
  useEffect(() => {
    void restore()
  }, [restore])

  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route path="/register" element={<RegisterPage />} />
      <Route element={<RequireAuth />}>
        <Route element={<MainLayout />}>
          <Route path="/" element={<Navigate to="/kb" replace />} />
          <Route path="/kb" element={<KBListPage />} />
          <Route path="/kb/:kbId/documents" element={<DocumentsPage />} />
          <Route path="/kb/:kbId/qa" element={<QAPage />} />
          <Route path="/dashboard" element={<DashboardPage />} />
        </Route>
      </Route>
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  )
}

// RequireAuth gates the layout subtree. Reads the user from the store; if there
// is none after restore, send the visitor to login. Otherwise render the nested
// routes via <Outlet/>.
function RequireAuth() {
  const user = useAuthStore((s) => s.user)
  if (!user) {
    return <Navigate to="/login" replace />
  }
  return <Outlet />
}
