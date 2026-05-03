import { createBrowserRouter, Navigate, Outlet } from 'react-router-dom'
import { useAuthStore } from './stores/authStore'
import AppShell from './components/layout/AppShell'
import LoginPage from './pages/auth/LoginPage'
import OtpPage from './pages/auth/OtpPage'
import CalendarPage from './pages/CalendarPage'
import BookingsPage from './pages/BookingsPage'
import ProfilePage from './pages/ProfilePage'
import AdminUsersPage from './pages/AdminUsersPage'
import WhereIsPage from './pages/WhereIsPage'
import NotFoundPage from './pages/NotFoundPage'

function RequireAuth() {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated)
  if (!isAuthenticated) return <Navigate to="/login" replace />
  return <Outlet />
}

function RequireAdmin() {
  const role = useAuthStore((s) => s.role)
  if (role !== 'admin') return <Navigate to="/" replace />
  return <Outlet />
}

export const router = createBrowserRouter([
  { path: '/login', element: <LoginPage /> },
  { path: '/login/otp', element: <OtpPage /> },
  {
    element: <RequireAuth />,
    children: [
      {
        element: <AppShell />,
        children: [
          { path: '/', element: <CalendarPage /> },
          { path: '/bookings', element: <BookingsPage /> },
          { path: '/profile', element: <ProfilePage /> },
          { path: '/where-is', element: <WhereIsPage /> },
          {
            element: <RequireAdmin />,
            children: [{ path: '/admin/users', element: <AdminUsersPage /> }],
          },
        ],
      },
    ],
  },
  { path: '*', element: <NotFoundPage /> },
])
