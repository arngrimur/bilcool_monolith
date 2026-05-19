import { act, renderHook } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { useGpsTracking } from './useGpsTracking'

function mockGeolocation(lat: number, lon: number) {
  Object.defineProperty(navigator, 'geolocation', {
    value: {
      getCurrentPosition: vi.fn((success: PositionCallback) => {
        success({
          coords: { latitude: lat, longitude: lon, speed: null, accuracy: 10, altitude: null, altitudeAccuracy: null, heading: null },
          timestamp: Date.now(),
        } as GeolocationPosition)
      }),
    },
    writable: true,
    configurable: true,
  })
}

describe('useGpsTracking', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    mockGeolocation(59.0, 18.0)
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('isTracking is false before startTracking', () => {
    const { result } = renderHook(() => useGpsTracking())
    expect(result.current.isTracking).toBe(false)
  })

  it('isTracking becomes true after startTracking', () => {
    const { result } = renderHook(() => useGpsTracking())
    act(() => result.current.startTracking())
    expect(result.current.isTracking).toBe(true)
  })

  it('calls onPoint immediately on startTracking', async () => {
    const onPoint = vi.fn().mockResolvedValue(undefined)
    const { result } = renderHook(() => useGpsTracking({ onPoint }))
    await act(async () => { result.current.startTracking() })
    expect(onPoint).toHaveBeenCalledTimes(1)
    expect(onPoint).toHaveBeenCalledWith(expect.objectContaining({ lat: 59.0, lon: 18.0 }))
  })

  it('calls onPoint again after 2 minutes', async () => {
    const onPoint = vi.fn().mockResolvedValue(undefined)
    const { result } = renderHook(() => useGpsTracking({ onPoint }))
    await act(async () => { result.current.startTracking() })
    await act(async () => { vi.advanceTimersByTime(2 * 60 * 1000) })
    expect(onPoint).toHaveBeenCalledTimes(2)
  })

  it('stopTracking sets isTracking to false', async () => {
    const { result } = renderHook(() => useGpsTracking())
    await act(async () => { result.current.startTracking() })
    act(() => result.current.stopTracking())
    expect(result.current.isTracking).toBe(false)
  })

  it('no more onPoint calls after stopTracking', async () => {
    const onPoint = vi.fn().mockResolvedValue(undefined)
    const { result } = renderHook(() => useGpsTracking({ onPoint }))
    await act(async () => { result.current.startTracking() })
    act(() => result.current.stopTracking())
    await act(async () => { vi.advanceTimersByTime(4 * 60 * 1000) })
    expect(onPoint).toHaveBeenCalledTimes(1) // only the initial sample
  })

  it('isPaused becomes true after pauseTracking', async () => {
    const { result } = renderHook(() => useGpsTracking())
    await act(async () => { result.current.startTracking() })
    act(() => { result.current.pauseTracking() })
    expect(result.current.isPaused).toBe(true)
  })

  it('no more onPoint calls while paused', async () => {
    const onPoint = vi.fn().mockResolvedValue(undefined)
    const { result } = renderHook(() => useGpsTracking({ onPoint }))
    await act(async () => { result.current.startTracking() })
    act(() => { result.current.pauseTracking() })
    await act(async () => { vi.advanceTimersByTime(4 * 60 * 1000) })
    expect(onPoint).toHaveBeenCalledTimes(1) // only initial
  })

  it('isPaused becomes false and interval resumes after resumeTracking', async () => {
    const onPoint = vi.fn().mockResolvedValue(undefined)
    const { result } = renderHook(() => useGpsTracking({ onPoint }))
    await act(async () => { result.current.startTracking() })
    act(() => { result.current.pauseTracking() })
    await act(async () => { result.current.resumeTracking() })
    expect(result.current.isPaused).toBe(false)
    await act(async () => { vi.advanceTimersByTime(2 * 60 * 1000) })
    expect(onPoint).toHaveBeenCalledTimes(3) // initial + resume immediate + 2min tick
  })

  it('currentPosition is updated on each sample', async () => {
    const { result } = renderHook(() => useGpsTracking())
    await act(async () => { result.current.startTracking() })
    expect(result.current.currentPosition).toEqual({ lat: 59.0, lon: 18.0 })
  })

  it('pauseTracking returns the current position', async () => {
    const { result } = renderHook(() => useGpsTracking())
    await act(async () => { result.current.startTracking() })
    let pos: ReturnType<typeof result.current.pauseTracking> = null
    act(() => { pos = result.current.pauseTracking() })
    expect(pos).toEqual({ lat: 59.0, lon: 18.0 })
  })
})
