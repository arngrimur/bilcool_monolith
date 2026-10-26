import { describe, it, expect, vi, afterEach } from 'vitest'
import { render, cleanup } from '@testing-library/react'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import { useEffect } from 'react'
import { RequireAuth } from './router'
import { useAuthStore } from './stores/authStore'

function TrackedChild({ onUnmount }: { onUnmount: () => void }) {
  useEffect(() => onUnmount, [onUnmount])
  return <div>booking-in-progress</div>
}

function renderGuarded() {
  return render(
    <MemoryRouter initialEntries={['/bookings']}>
      <Routes>
        <Route path="/login" element={<div>login</div>} />
        <Route element={<RequireAuth />}>
          <Route
            path="/bookings"
            element={<TrackedChild onUnmount={onUnmountSpy} />}
          />
        </Route>
      </Routes>
    </MemoryRouter>,
  )
}

let onUnmountSpy: () => void

describe('RequireAuth', () => {
  afterEach(() => {
    cleanup()
    useAuthStore.getState().clearAuth()
  })

  it('unmounts the in-progress booking flow when the user is logged out', () => {
    useAuthStore.setState({ isAuthenticated: true })
    onUnmountSpy = vi.fn()

    const { rerender } = renderGuarded()
    expect(onUnmountSpy).not.toHaveBeenCalled()

    // simulate a logout occurring mid-booking (explicit logout, or an
    // expired-token check clearing auth state)
    useAuthStore.getState().clearAuth()
    rerender(
      <MemoryRouter initialEntries={['/bookings']}>
        <Routes>
          <Route path="/login" element={<div>login</div>} />
          <Route element={<RequireAuth />}>
            <Route
              path="/bookings"
              element={<TrackedChild onUnmount={onUnmountSpy} />}
            />
          </Route>
        </Routes>
      </MemoryRouter>,
    )

    expect(onUnmountSpy).toHaveBeenCalledTimes(1)
  })
})
