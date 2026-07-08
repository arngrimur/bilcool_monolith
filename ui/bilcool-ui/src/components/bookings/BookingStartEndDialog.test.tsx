import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, fireEvent, waitFor, cleanup } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import BookingStartEndDialog from './BookingStartEndDialog'
import type { BookingResponse } from '../../types/api'

vi.mock('../../api/bookings', () => ({
  endBooking: vi.fn(),
  deleteBooking: vi.fn(),
  pauseBooking: vi.fn(),
  resumeBooking: vi.fn(),
  addTrackPoints: vi.fn(),
}))

const startedBooking: BookingResponse = {
  user_ref: 'user-1',
  booking_reference: 'booking-ref-1',
  start_date: '2020-01-01T10:00:00Z',
  end_date: '2020-01-01T18:00:00Z',
}

function renderDialog() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={queryClient}>
      <BookingStartEndDialog
        open={true}
        onOpenChange={vi.fn()}
        booking={startedBooking}
        hasDistance={false}
      />
    </QueryClientProvider>,
  )
}

describe('BookingStartEndDialog', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    sessionStorage.clear()
  })

  afterEach(() => {
    cleanup()
  })

  it('submits the correct distance after the dialog is remounted following a logout', async () => {
    const { endBooking } = await import('../../api/bookings')
    vi.mocked(endBooking).mockResolvedValue(undefined)

    // 1. user opens "end booking" and enters odometer readings
    const first = renderDialog()
    fireEvent.change(document.getElementById('start-odo') as HTMLInputElement, { target: { value: '100' } })
    fireEvent.change(document.getElementById('end-odo') as HTMLInputElement, { target: { value: '150' } })

    // 2. user is logged out mid-entry: RequireAuth redirects to /login,
    // unmounting the dialog before they hit submit
    first.unmount()

    // 3. user logs back in and reopens the same booking
    const second = renderDialog()
    const form = document.getElementById('end-booking-form')!
    fireEvent.submit(form)

    // 4. the booking should end with the distance the user had already
    // entered before being logged out - not lost, not re-typed
    await waitFor(() => {
      expect(endBooking).toHaveBeenCalledWith('booking-ref-1', {
        odometer_start: 100000,
        odometer_end: 150000,
      })
    })

    second.unmount()
  })
})
