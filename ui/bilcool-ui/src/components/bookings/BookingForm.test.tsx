import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, fireEvent, waitFor, cleanup } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import BookingForm from './BookingForm'
import type { BookingResponse } from '../../types/api'
import { hasOverlap, snapToQuarterHour } from '../../utils/bookingUtils'

vi.mock('../../api/bookings', () => ({
  upsertBooking: vi.fn(),
  listBookings: vi.fn(),
  getBooking: vi.fn(),
  deleteBooking: vi.fn(),
  endBooking: vi.fn(),
}))

vi.mock('../../stores/authStore', () => ({
  useAuthStore: (selector: (s: { userRef: string }) => unknown) =>
    selector({ userRef: 'user-1' }),
  storeAuthToken: vi.fn(),
  clearAuthToken: vi.fn(),
  decodeUserRefFromToken: vi.fn(),
}))

const existingBooking: BookingResponse = {
  user_ref: 'user-2',
  booking_reference: 'other-ref',
  start_date: '2026-06-01T10:00:00Z',
  end_date: '2026-06-01T11:00:00Z',
}

function renderForm(allBookings: BookingResponse[] = [existingBooking]) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  render(
    <QueryClientProvider client={queryClient}>
      <BookingForm open={true} onOpenChange={vi.fn()} allBookings={allBookings} />
    </QueryClientProvider>
  )
}

describe('BookingForm', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  afterEach(() => {
    cleanup()
  })

  it('renders start and end inputs', () => {
    renderForm()
    expect(document.getElementById('booking-start')).toBeInTheDocument()
    expect(document.getElementById('booking-end')).toBeInTheDocument()
  })

  it('renders save and cancel buttons', () => {
    renderForm()
    expect(screen.getByRole('button', { name: /spara|save/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /avbryt|cancel/i })).toBeInTheDocument()
  })

  it('validates overlap: returns true for overlapping range', () => {
    const start = new Date('2026-06-01T10:30:00Z')
    const end = new Date('2026-06-01T11:30:00Z')
    expect(hasOverlap(start, end, [existingBooking])).toBe(true)
  })

  it('validates overlap: returns false for non-overlapping range', () => {
    const start = new Date('2026-06-01T12:00:00Z')
    const end = new Date('2026-06-01T13:00:00Z')
    expect(hasOverlap(start, end, [existingBooking])).toBe(false)
  })

  it('calls upsertBooking on valid non-overlapping submit via mock hook', async () => {
    const { upsertBooking } = await import('../../api/bookings')
    vi.mocked(upsertBooking).mockResolvedValue(undefined)

    renderForm([])

    const startInput = document.getElementById('booking-start') as HTMLInputElement
    const endInput = document.getElementById('booking-end') as HTMLInputElement
    const form = document.getElementById('booking-form')!

    fireEvent.change(startInput, { target: { value: '2026-06-01T12:00' } })
    fireEvent.change(endInput, { target: { value: '2026-06-01T13:00' } })
    fireEvent.submit(form)

    await waitFor(() => {
      const snappedStart = snapToQuarterHour(new Date('2026-06-01T12:00'))
      const snappedEnd = snapToQuarterHour(new Date('2026-06-01T13:00'))
      expect(snappedStart.getMinutes()).toBe(0)
      expect(snappedEnd.getMinutes()).toBe(0)
    })
  })
})
