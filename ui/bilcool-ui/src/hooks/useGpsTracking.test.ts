import { act, renderHook } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { useGpsTracking } from './useGpsTracking'

const HIGH_SPEED_KMH = 15
const LOW_SPEED_KMH = 3
const FIVE_MIN_MS = 5 * 60 * 1000

// Each step moves ~111 metres north, giving a predictable haversine distance
function makePosition(step: number, speedKmh: number, timestamp: number): GeolocationPosition {
  return {
    coords: {
      latitude: 59.0 + step * 0.001,
      longitude: 18.0,
      speed: speedKmh / 3.6, // m/s, as the Geolocation API provides
      accuracy: 10,
      altitude: null,
      altitudeAccuracy: null,
      heading: null,
    },
    timestamp,
  } as GeolocationPosition
}

describe('useGpsTracking', () => {
  let sendPosition: (pos: GeolocationPosition) => void

  beforeEach(() => {
    Object.defineProperty(navigator, 'geolocation', {
      value: {
        watchPosition: vi.fn((successCb: (position: GeolocationPosition) => void) => {
          sendPosition = (pos) => act(() => successCb(pos))
          return 1
        }),
        clearWatch: vi.fn(),
      },
      writable: true,
      configurable: true,
    })
  })

  it('accumulates distance while speed stays above 7 km/h', () => {
    const { result } = renderHook(() => useGpsTracking())
    act(() => result.current.startTracking())

    sendPosition(makePosition(0, HIGH_SPEED_KMH, 0))
    sendPosition(makePosition(1, HIGH_SPEED_KMH, 1_000))
    sendPosition(makePosition(2, HIGH_SPEED_KMH, 2_000))

    expect(result.current.distanceMeters).toBeGreaterThan(0)
  })

  it('counts distance accumulated during a brief low-speed window shorter than 5 minutes', () => {
    const { result } = renderHook(() => useGpsTracking())
    act(() => result.current.startTracking())

    sendPosition(makePosition(0, HIGH_SPEED_KMH, 0))
    sendPosition(makePosition(1, HIGH_SPEED_KMH, 1_000))
    const distanceBeforeSlow = result.current.distanceMeters

    // 2-minute slow window — below the 5-minute threshold
    sendPosition(makePosition(2, LOW_SPEED_KMH, 2_000))
    sendPosition(makePosition(3, LOW_SPEED_KMH, 2_000 + 2 * 60_000))

    // Recover to high speed
    sendPosition(makePosition(4, HIGH_SPEED_KMH, 2_000 + 2 * 60_000 + 1_000))

    // The distance covered during the brief slow period must still be included
    expect(result.current.distanceMeters).toBeGreaterThan(distanceBeforeSlow)
  })

  it('retroactively removes distance once low speed is sustained for more than 5 minutes', () => {
    const { result } = renderHook(() => useGpsTracking())
    act(() => result.current.startTracking())

    sendPosition(makePosition(0, HIGH_SPEED_KMH, 0))
    sendPosition(makePosition(1, HIGH_SPEED_KMH, 1_000))
    sendPosition(makePosition(2, HIGH_SPEED_KMH, 2_000))
    const distanceBeforeSlow = result.current.distanceMeters

    // Low-speed positions — distance is tentatively added
    sendPosition(makePosition(3, LOW_SPEED_KMH, 3_000))
    sendPosition(makePosition(4, LOW_SPEED_KMH, 3_000 + 2 * 60_000))
    const distanceDuringSlow = result.current.distanceMeters
    expect(distanceDuringSlow).toBeGreaterThan(distanceBeforeSlow) // tentatively added

    // Cross the 5-minute mark — retroactive removal must trigger
    sendPosition(makePosition(5, LOW_SPEED_KMH, 3_000 + FIVE_MIN_MS + 1_000))

    expect(result.current.distanceMeters).toBeCloseTo(distanceBeforeSlow, 2)
    expect(result.current.distanceMeters).toBeLessThan(distanceDuringSlow)
  })

  it('does not accumulate further distance while paused after the 5-minute threshold', () => {
    const { result } = renderHook(() => useGpsTracking())
    act(() => result.current.startTracking())

    sendPosition(makePosition(0, HIGH_SPEED_KMH, 0))
    sendPosition(makePosition(1, HIGH_SPEED_KMH, 1_000))

    // Trigger the 5-minute pause
    sendPosition(makePosition(2, LOW_SPEED_KMH, 2_000))
    sendPosition(makePosition(3, LOW_SPEED_KMH, 2_000 + FIVE_MIN_MS + 1_000))
    const distanceAtPause = result.current.distanceMeters

    // Further low-speed positions must not add to the total
    sendPosition(makePosition(4, LOW_SPEED_KMH, 2_000 + FIVE_MIN_MS + 2_000))
    sendPosition(makePosition(5, LOW_SPEED_KMH, 2_000 + FIVE_MIN_MS + 3_000))

    expect(result.current.distanceMeters).toBe(distanceAtPause)
  })

  it('resumes accumulating distance once speed recovers after a sustained low-speed pause', () => {
    const { result } = renderHook(() => useGpsTracking())
    act(() => result.current.startTracking())

    sendPosition(makePosition(0, HIGH_SPEED_KMH, 0))
    sendPosition(makePosition(1, HIGH_SPEED_KMH, 1_000))
    const distanceBeforeSlow = result.current.distanceMeters

    // Sustained low speed — triggers retroactive removal and pause
    sendPosition(makePosition(2, LOW_SPEED_KMH, 2_000))
    sendPosition(makePosition(3, LOW_SPEED_KMH, 2_000 + FIVE_MIN_MS + 1_000))
    const distanceAfterPause = result.current.distanceMeters
    expect(distanceAfterPause).toBeCloseTo(distanceBeforeSlow, 2)

    // Recover to high speed — new movement must count again
    sendPosition(makePosition(4, HIGH_SPEED_KMH, 2_000 + FIVE_MIN_MS + 2_000))
    sendPosition(makePosition(5, HIGH_SPEED_KMH, 2_000 + FIVE_MIN_MS + 3_000))

    expect(result.current.distanceMeters).toBeGreaterThan(distanceAfterPause)
  })

  it('resumes accumulation after a screen-lock gap at low speed even when paused', () => {
    const { result } = renderHook(() => useGpsTracking())
    act(() => result.current.startTracking())

    let t = 0

    sendPosition(makePosition(0, HIGH_SPEED_KMH, t))
    t += 1_000
    sendPosition(makePosition(1, HIGH_SPEED_KMH, t))
    t += 1_000

    // Sit still for >5 min → retroactive removal + isPaused = true
    sendPosition(makePosition(1, LOW_SPEED_KMH, t))
    t += FIVE_MIN_MS + 1_000
    sendPosition(makePosition(1, LOW_SPEED_KMH, t))
    const distWhenPaused = result.current.distanceMeters

    // Screen locks for 10 minutes while driving ~1.5 km north (15 steps)
    t += 10 * 60_000
    sendPosition(makePosition(16, LOW_SPEED_KMH, t))

    expect(result.current.distanceMeters).toBeGreaterThan(distWhenPaused + 1_000)
  })

  it('resumes accumulation after a screen-lock gap at high speed', () => {
    const { result } = renderHook(() => useGpsTracking())
    act(() => result.current.startTracking())

    let t = 0

    sendPosition(makePosition(0, HIGH_SPEED_KMH, t))
    t += 1_000

    // Trigger pause
    sendPosition(makePosition(0, LOW_SPEED_KMH, t))
    t += FIVE_MIN_MS + 1_000
    sendPosition(makePosition(0, LOW_SPEED_KMH, t))
    const distWhenPaused = result.current.distanceMeters

    // Screen locked for 8 min, unlocked while still driving at speed
    t += 8 * 60_000
    sendPosition(makePosition(12, HIGH_SPEED_KMH, t))

    expect(result.current.distanceMeters).toBeGreaterThan(distWhenPaused + 1_000)
  })

  it('does not reset pause state for a short gap (< 30 s)', () => {
    const { result } = renderHook(() => useGpsTracking())
    act(() => result.current.startTracking())

    let t = 0

    sendPosition(makePosition(0, HIGH_SPEED_KMH, t))
    t += 1_000

    // Trigger pause
    sendPosition(makePosition(0, LOW_SPEED_KMH, t))
    t += FIVE_MIN_MS + 1_000
    sendPosition(makePosition(0, LOW_SPEED_KMH, t))
    const distWhenPaused = result.current.distanceMeters

    // Short gap (20 s) — must not reset paused state
    t += 20_000
    sendPosition(makePosition(1, LOW_SPEED_KMH, t))

    expect(result.current.distanceMeters).toBeCloseTo(distWhenPaused, 0)
  })
})
