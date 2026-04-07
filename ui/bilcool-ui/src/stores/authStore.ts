import { create } from 'zustand'
import type { QueryClient } from '@tanstack/react-query'
import { getUser } from '../api/auth'

interface AuthState {
  token: string | null
  userRef: string | null
  username: string | null
  email: string | null
  role: 'admin' | 'user' | null
  isAuthenticated: boolean
  setAuth: (payload: {
    token: string
    userRef: string
    username: string
    email: string
    role: 'admin' | 'user'
  }) => void
  clearAuth: () => void
}

export const useAuthStore = create<AuthState>((set) => ({
  token: null,
  userRef: null,
  username: null,
  email: null,
  role: null,
  isAuthenticated: false,
  setAuth: (payload) =>
    set({
      token: payload.token,
      userRef: payload.userRef,
      username: payload.username,
      email: payload.email,
      role: payload.role,
      isAuthenticated: true,
    }),
  clearAuth: () =>
    set({
      token: null,
      userRef: null,
      username: null,
      email: null,
      role: null,
      isAuthenticated: false,
    }),
}))

function decodeJwtPayload(token: string): Record<string, unknown> {
  try {
    const segment = token.split('.')[1]
    const decoded = atob(segment.replace(/-/g, '+').replace(/_/g, '/'))
    return JSON.parse(decoded) as Record<string, unknown>
  } catch {
    return {}
  }
}

export function initAuth(queryClient: QueryClient) {
  const token = localStorage.getItem('bilcool_token')
  if (!token) return

  const payload = decodeJwtPayload(token)
  const exp = payload['exp']
  if (typeof exp === 'number' && exp * 1000 < Date.now()) {
    localStorage.removeItem('bilcool_token')
    return
  }

  const userRef = typeof payload['user_ref'] === 'string' ? payload['user_ref'] : null
  if (!userRef) return

  queryClient.fetchQuery({
    queryKey: ['user', userRef],
    queryFn: () => getUser(userRef),
    staleTime: 5 * 60_000,
  }).then((user) => {
    useAuthStore.getState().setAuth({
      token,
      userRef: user.user_ref,
      username: user.username,
      email: user.email,
      role: user.role,
    })
  }).catch(() => {
    localStorage.removeItem('bilcool_token')
  })
}

export function storeAuthToken(token: string) {
  localStorage.setItem('bilcool_token', token)
}

export function clearAuthToken() {
  localStorage.removeItem('bilcool_token')
}

export function decodeUserRefFromToken(token: string): string | null {
  const payload = decodeJwtPayload(token)
  return typeof payload['user_ref'] === 'string' ? payload['user_ref'] : null
}
