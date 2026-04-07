import type { BookingResponse } from '../types/api'

export function snapToQuarterHour(date: Date): Date {
  const snapped = new Date(date)
  const minutes = snapped.getMinutes()
  const remainder = minutes % 15
  if (remainder === 0) {
    snapped.setSeconds(0, 0)
    return snapped
  }
  const snap = remainder < 8 ? minutes - remainder : minutes + (15 - remainder)
  snapped.setMinutes(snap, 0, 0)
  return snapped
}

export function hasOverlap(
  proposedStart: Date,
  proposedEnd: Date,
  bookings: BookingResponse[],
  excludeRef?: string
): boolean {
  return bookings.some((booking) => {
    if (excludeRef && booking.booking_reference === excludeRef) return false
    const existingStart = new Date(booking.start_date)
    const existingEnd = new Date(booking.end_date)
    return proposedStart < existingEnd && proposedEnd > existingStart
  })
}
