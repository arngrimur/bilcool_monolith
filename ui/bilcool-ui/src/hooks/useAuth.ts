import { useNavigate } from 'react-router-dom'
import { useAuthStore, clearAuthToken } from '../stores/authStore'

export function useAuth() {
  const navigate = useNavigate()
  const clearAuth = useAuthStore((s) => s.clearAuth)

  function logout() {
    clearAuthToken()
    clearAuth()
    navigate('/login')
  }

  return { logout }
}
