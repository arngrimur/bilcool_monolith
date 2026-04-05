import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, fireEvent, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import LoginPage from './LoginPage'

vi.mock('../../api/auth', () => ({
  beginLogin: vi.fn(),
  completeLogin: vi.fn(),
  getUser: vi.fn(),
}))

vi.mock('@simplewebauthn/browser', () => ({
  startAuthentication: vi.fn(),
}))

const mockNavigate = vi.fn()
vi.mock('react-router-dom', async (importOriginal) => {
  const mod = await importOriginal<typeof import('react-router-dom')>()
  return { ...mod, useNavigate: () => mockNavigate }
})

function renderLoginPage() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <LoginPage />
      </MemoryRouter>
    </QueryClientProvider>
  )
}

describe('LoginPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders email input', () => {
    renderLoginPage()
    const input = document.querySelector('input[type="email"]')
    expect(input).toBeInTheDocument()
  })

  it('submit triggers beginLogin', async () => {
    const { beginLogin } = await import('../../api/auth')
    vi.mocked(beginLogin).mockResolvedValue({ next_step: 'verify_token' })

    renderLoginPage()

    const input = document.querySelector('input[type="email"]') as HTMLInputElement
    fireEvent.change(input, { target: { value: 'test@example.com' } })
    fireEvent.submit(document.querySelector('form')!)

    await waitFor(() => {
      expect(beginLogin).toHaveBeenCalledWith({ email: 'test@example.com' })
    })
  })

  it('navigates to /login/otp when next_step is verify_token', async () => {
    const { beginLogin } = await import('../../api/auth')
    vi.mocked(beginLogin).mockResolvedValue({ next_step: 'verify_token' })

    renderLoginPage()

    const input = document.querySelector('input[type="email"]') as HTMLInputElement
    fireEvent.change(input, { target: { value: 'test@example.com' } })
    fireEvent.submit(document.querySelector('form')!)

    await waitFor(() => {
      expect(mockNavigate).toHaveBeenCalledWith('/login/otp', { state: { email: 'test@example.com' } })
    })
  })

  it('shows error for unknown email (404)', async () => {
    const { beginLogin } = await import('../../api/auth')
    vi.mocked(beginLogin).mockRejectedValue(Object.assign(new Error('Not found'), { status: 404 }))

    renderLoginPage()

    const input = document.querySelector('input[type="email"]') as HTMLInputElement
    fireEvent.change(input, { target: { value: 'unknown@example.com' } })
    fireEvent.submit(document.querySelector('form')!)

    await waitFor(() => {
      expect(document.querySelector('[role="alert"]')).toBeInTheDocument()
    })
  })

  it('calls startAuthentication when next_step is passkey_assertion', async () => {
    const { beginLogin, completeLogin, getUser } = await import('../../api/auth')
    const { startAuthentication } = await import('@simplewebauthn/browser')
    vi.mocked(beginLogin).mockResolvedValue({
      next_step: 'passkey_assertion',
      session_id: 'sess-1',
      options: {},
    })
    vi.mocked(startAuthentication).mockResolvedValue({} as Awaited<ReturnType<typeof startAuthentication>>)
    vi.mocked(completeLogin).mockResolvedValue({ token: 'eyJhbGciOiJIUzI1NiJ9.eyJ1c2VyX3JlZiI6InUxIn0.sig' })
    vi.mocked(getUser).mockResolvedValue({ user_ref: 'u1', username: 'alice', email: 'a@a.com', role: 'user' })

    renderLoginPage()

    const input = document.querySelector('input[type="email"]') as HTMLInputElement
    fireEvent.change(input, { target: { value: 'alice@example.com' } })
    fireEvent.submit(document.querySelector('form')!)

    await waitFor(() => {
      expect(startAuthentication).toHaveBeenCalled()
    })
  })
})
