import { describe, it, expect, beforeEach } from 'vitest'
import { useAuthStore } from './authStore'

describe('authStore', () => {
  beforeEach(() => {
    useAuthStore.getState().clearAuth()
  })

  it('setAuth stores all fields and sets isAuthenticated true', () => {
    useAuthStore.getState().setAuth({
      token: 'tok',
      userRef: 'ref-1',
      username: 'alice',
      email: 'alice@example.com',
      role: 'admin',
    })
    const state = useAuthStore.getState()
    expect(state.token).toBe('tok')
    expect(state.userRef).toBe('ref-1')
    expect(state.username).toBe('alice')
    expect(state.email).toBe('alice@example.com')
    expect(state.role).toBe('admin')
    expect(state.isAuthenticated).toBe(true)
  })

  it('clearAuth resets all fields to null and isAuthenticated to false', () => {
    useAuthStore.getState().setAuth({
      token: 'tok',
      userRef: 'ref-1',
      username: 'alice',
      email: 'alice@example.com',
      role: 'user',
    })
    useAuthStore.getState().clearAuth()
    const state = useAuthStore.getState()
    expect(state.token).toBeNull()
    expect(state.userRef).toBeNull()
    expect(state.username).toBeNull()
    expect(state.email).toBeNull()
    expect(state.role).toBeNull()
    expect(state.isAuthenticated).toBe(false)
  })
})
