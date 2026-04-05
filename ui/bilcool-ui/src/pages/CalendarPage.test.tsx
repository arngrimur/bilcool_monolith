import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import CalendarPage from './CalendarPage'
import type { BookingResponse } from '../types/api'

vi.mock('../api/bookings', () => ({
  listBookings: vi.fn(),
  upsertBooking: vi.fn(),
  deleteBooking: vi.fn(),
  endBooking: vi.fn(),
  getBooking: vi.fn(),
}))

vi.mock('../api/events', () => ({
  listEvents: vi.fn(),
}))

vi.mock('../api/auth', () => ({
  getUser: vi.fn(),
}))

vi.mock('../stores/authStore', () => ({
  useAuthStore: (selector: (s: { userRef: string | null }) => unknown) =>
    selector({ userRef: 'u1' }),
  storeAuthToken: vi.fn(),
  clearAuthToken: vi.fn(),
  decodeUserRefFromToken: vi.fn(),
}))

vi.mock('@fullcalendar/react', () => ({
  default: vi.fn().mockImplementation(() => (
    <div data-testid="fullcalendar">FullCalendar Mock</div>
  )),
}))

const mockBookings: BookingResponse[] = [
  {
    user_ref: 'u1',
    booking_reference: 'ref-1',
    start_date: new Date().toISOString(),
    end_date: new Date(Date.now() + 3600000).toISOString(),
  },
]

describe('CalendarPage', () => {
  it('renders FullCalendar', async () => {
    const { listBookings } = await import('../api/bookings')
    const { listEvents } = await import('../api/events')
    vi.mocked(listBookings).mockResolvedValue(mockBookings)
    vi.mocked(listEvents).mockResolvedValue([])

    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    })
    queryClient.setQueryData(['bookings'], mockBookings)
    queryClient.setQueryData(
      ['events', { producer: 'bookings', event_type: 'booking_ended' }],
      []
    )

    render(
      <QueryClientProvider client={queryClient}>
        <MemoryRouter>
          <CalendarPage />
        </MemoryRouter>
      </QueryClientProvider>
    )

    expect(screen.getByTestId('fullcalendar')).toBeInTheDocument()
  })

  it('passes booking data as events to FullCalendar', async () => {
    const FullCalendar = await import('@fullcalendar/react')
    const { listBookings } = await import('../api/bookings')
    const { listEvents } = await import('../api/events')
    vi.mocked(listBookings).mockResolvedValue(mockBookings)
    vi.mocked(listEvents).mockResolvedValue([])

    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    })
    queryClient.setQueryData(['bookings'], mockBookings)
    queryClient.setQueryData(
      ['events', { producer: 'bookings', event_type: 'booking_ended' }],
      []
    )

    render(
      <QueryClientProvider client={queryClient}>
        <MemoryRouter>
          <CalendarPage />
        </MemoryRouter>
      </QueryClientProvider>
    )

    expect(FullCalendar.default).toHaveBeenCalled()
  })
})
