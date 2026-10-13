import { describe, it, expect, vi, beforeEach } from 'vitest'
import { getRoute } from './bookings'
import { apiFetch } from './client'
import type { BookingRoute, TrackPoint } from '../types/api'

vi.mock('./client', () => ({
  apiFetch: vi.fn(),
}))

const points: TrackPoint[] = [
  { lat: 59.33, lon: 18.06, recorded_at: '2026-05-01T10:00:00Z' },
  { lat: 59.34, lon: 18.07, recorded_at: '2026-05-01T10:02:00Z' },
]

describe('getRoute', () => {
  beforeEach(() => {
    vi.mocked(apiFetch).mockReset()
  })

  it('returns the route from the route endpoint', async () => {
    const route: BookingRoute = { points, route: [[{ lat: 59.331, lon: 18.061 }, { lat: 59.339, lon: 18.069 }]] }
    vi.mocked(apiFetch).mockResolvedValue(route)

    await expect(getRoute('ref-1')).resolves.toEqual(route)
    expect(apiFetch).toHaveBeenCalledTimes(1)
    expect(apiFetch).toHaveBeenCalledWith('/api/v1/bookings/ref-1/route')
  })

  it('falls back to the track points when the route endpoint fails', async () => {
    vi.mocked(apiFetch)
      .mockRejectedValueOnce(Object.assign(new Error('Not found'), { status: 404 }))
      .mockResolvedValueOnce(points)

    await expect(getRoute('ref-1')).resolves.toEqual({ points, route: [] })
    expect(apiFetch).toHaveBeenNthCalledWith(2, '/api/v1/bookings/ref-1/track')
  })

  it('does not fall back on 401', async () => {
    vi.mocked(apiFetch).mockRejectedValue(Object.assign(new Error('Unauthorized'), { status: 401 }))

    await expect(getRoute('ref-1')).rejects.toThrow('Unauthorized')
    expect(apiFetch).toHaveBeenCalledTimes(1)
  })

  it('fails when the fallback fails too', async () => {
    vi.mocked(apiFetch).mockRejectedValue(new Error('boom'))

    await expect(getRoute('ref-1')).rejects.toThrow('boom')
  })
})
