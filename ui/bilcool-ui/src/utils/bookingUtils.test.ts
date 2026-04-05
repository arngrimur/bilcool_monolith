import { describe, it, expect } from 'vitest'
import { snapToQuarterHour, hasOverlap } from './bookingUtils'
import type { BookingResponse } from '../types/api'

describe('snapToQuarterHour', () => {
  it('snaps :00 unchanged', () => {
    const d = new Date('2026-01-01T10:00:00')
    expect(snapToQuarterHour(d).getMinutes()).toBe(0)
  })

  it('snaps :15 unchanged', () => {
    const d = new Date('2026-01-01T10:15:00')
    expect(snapToQuarterHour(d).getMinutes()).toBe(15)
  })

  it('snaps :30 unchanged', () => {
    const d = new Date('2026-01-01T10:30:00')
    expect(snapToQuarterHour(d).getMinutes()).toBe(30)
  })

  it('snaps :45 unchanged', () => {
    const d = new Date('2026-01-01T10:45:00')
    expect(snapToQuarterHour(d).getMinutes()).toBe(45)
  })

  it('snaps :07 down to :00', () => {
    const d = new Date('2026-01-01T10:07:00')
    expect(snapToQuarterHour(d).getMinutes()).toBe(0)
  })

  it('snaps :08 up to :15', () => {
    const d = new Date('2026-01-01T10:08:00')
    expect(snapToQuarterHour(d).getMinutes()).toBe(15)
  })

  it('snaps :22 down to :15', () => {
    const d = new Date('2026-01-01T10:22:00')
    expect(snapToQuarterHour(d).getMinutes()).toBe(15)
  })

  it('clears seconds and milliseconds', () => {
    const d = new Date('2026-01-01T10:00:30.500')
    const snapped = snapToQuarterHour(d)
    expect(snapped.getSeconds()).toBe(0)
    expect(snapped.getMilliseconds()).toBe(0)
  })
})

describe('hasOverlap', () => {
  const booking: BookingResponse = {
    user_ref: 'u1',
    booking_reference: 'ref-1',
    start_date: '2026-01-01T10:00:00Z',
    end_date: '2026-01-01T11:00:00Z',
  }

  it('returns true when proposed range overlaps at end', () => {
    expect(
      hasOverlap(
        new Date('2026-01-01T10:30:00Z'),
        new Date('2026-01-01T11:30:00Z'),
        [booking]
      )
    ).toBe(true)
  })

  it('returns false when proposed range starts at existing end (adjacent, no overlap)', () => {
    expect(
      hasOverlap(
        new Date('2026-01-01T11:00:00Z'),
        new Date('2026-01-01T12:00:00Z'),
        [booking]
      )
    ).toBe(false)
  })

  it('returns false when proposed range ends at existing start (adjacent, no overlap)', () => {
    expect(
      hasOverlap(
        new Date('2026-01-01T09:00:00Z'),
        new Date('2026-01-01T10:00:00Z'),
        [booking]
      )
    ).toBe(false)
  })

  it('returns true when proposed range is fully inside existing', () => {
    expect(
      hasOverlap(
        new Date('2026-01-01T10:15:00Z'),
        new Date('2026-01-01T10:45:00Z'),
        [booking]
      )
    ).toBe(true)
  })

  it('excludes the booking being edited by ref', () => {
    expect(
      hasOverlap(
        new Date('2026-01-01T10:30:00Z'),
        new Date('2026-01-01T11:30:00Z'),
        [booking],
        'ref-1'
      )
    ).toBe(false)
  })
})
