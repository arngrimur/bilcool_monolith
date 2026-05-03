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

  it('records first and second drive segments but not a 15-minute walk between them', () => {
    const { result } = renderHook(() => useGpsTracking())
    act(() => result.current.startTracking())

    let t = 0

    // First drive
    sendPosition(makePosition(0, HIGH_SPEED_KMH, t))
    t += 1_000
    sendPosition(makePosition(1, HIGH_SPEED_KMH, t))
    t += 1_000
    sendPosition(makePosition(2, HIGH_SPEED_KMH, t))
    const distanceAfterFirstDrive = result.current.distanceMeters
    expect(distanceAfterFirstDrive).toBeGreaterThan(0)

    // 15-minute walk — three ~5-minute steps, each gap < 6 min to avoid screen-lock detection
    t += 1_000
    sendPosition(makePosition(3, LOW_SPEED_KMH, t))       // start walking
    t += FIVE_MIN_MS + 1_000                               // 5 min → retroactive removal fires
    sendPosition(makePosition(4, LOW_SPEED_KMH, t))
    t += FIVE_MIN_MS - 1_000                               // another 5 min
    sendPosition(makePosition(5, LOW_SPEED_KMH, t))
    t += FIVE_MIN_MS - 1_000                               // another 5 min (15 min total)
    sendPosition(makePosition(6, LOW_SPEED_KMH, t))

    // Walk distance must have been stripped
    expect(result.current.distanceMeters).toBeCloseTo(distanceAfterFirstDrive, 2)

    // Second drive
    t += 1_000
    sendPosition(makePosition(7, HIGH_SPEED_KMH, t))
    t += 1_000
    sendPosition(makePosition(8, HIGH_SPEED_KMH, t))

    expect(result.current.distanceMeters).toBeGreaterThan(distanceAfterFirstDrive)
  })

  it('discards trailing low-speed distance when stopping before the 5-minute threshold', () => {
    const { result } = renderHook(() => useGpsTracking())
    act(() => result.current.startTracking())

    sendPosition(makePosition(0, HIGH_SPEED_KMH, 0))
    sendPosition(makePosition(1, HIGH_SPEED_KMH, 1_000))
    const distanceAfterDriving = result.current.distanceMeters

    // Walk for 2 minutes — tentative accumulation, threshold not yet crossed
    sendPosition(makePosition(2, LOW_SPEED_KMH, 2_000))
    sendPosition(makePosition(3, LOW_SPEED_KMH, 2_000 + 2 * 60_000))

    // Stop while still in the tentative low-speed window
    act(() => result.current.stopTracking())

    expect(result.current.distanceMeters).toBeCloseTo(distanceAfterDriving, 2)
  })

  it('records zero when the entire session is a walk (< 5 min)', () => {
    const { result } = renderHook(() => useGpsTracking())
    act(() => result.current.startTracking())

    sendPosition(makePosition(0, LOW_SPEED_KMH, 0))
    sendPosition(makePosition(1, LOW_SPEED_KMH, 1_000))
    sendPosition(makePosition(2, LOW_SPEED_KMH, 2_000))

    act(() => result.current.stopTracking())

    expect(result.current.distanceMeters).toBe(0)
  })

  // Manual pause / resume

  it('isPaused is false before any manual pause', () => {
    const { result } = renderHook(() => useGpsTracking())
    act(() => result.current.startTracking())
    expect(result.current.isPaused).toBe(false)
  })

  it('isPaused becomes true after calling pauseTracking', () => {
    const { result } = renderHook(() => useGpsTracking())
    act(() => result.current.startTracking())
    sendPosition(makePosition(0, HIGH_SPEED_KMH, 0))
    act(() => { result.current.pauseTracking() })
    expect(result.current.isPaused).toBe(true)
  })

  it('pauseTracking returns the last known GPS position', () => {
    const { result } = renderHook(() => useGpsTracking())
    act(() => result.current.startTracking())
    sendPosition(makePosition(2, HIGH_SPEED_KMH, 0))

    let savedPos: ReturnType<typeof result.current.pauseTracking> = null
    act(() => { savedPos = result.current.pauseTracking() })

    expect(savedPos).not.toBeNull()
    expect(savedPos!.lat).toBeCloseTo(59.002, 5)
    expect(savedPos!.lon).toBeCloseTo(18.0, 5)
  })

  it('distance does not change while manually paused', () => {
    const { result } = renderHook(() => useGpsTracking())
    act(() => result.current.startTracking())

    sendPosition(makePosition(0, HIGH_SPEED_KMH, 0))
    sendPosition(makePosition(1, HIGH_SPEED_KMH, 1_000))
    const distBeforePause = result.current.distanceMeters

    act(() => { result.current.pauseTracking() })

    sendPosition(makePosition(2, HIGH_SPEED_KMH, 2_000))
    sendPosition(makePosition(3, HIGH_SPEED_KMH, 3_000))

    expect(result.current.distanceMeters).toBeCloseTo(distBeforePause, 2)
  })

  it('isPaused becomes false after resumeTracking', () => {
    const { result } = renderHook(() => useGpsTracking())
    act(() => result.current.startTracking())
    sendPosition(makePosition(0, HIGH_SPEED_KMH, 0))
    act(() => { result.current.pauseTracking() })
    act(() => { result.current.resumeTracking({ lat: 59.0, lon: 18.0 }) })
    expect(result.current.isPaused).toBe(false)
  })

  it('distance accumulates again after resumeTracking', () => {
    const { result } = renderHook(() => useGpsTracking())
    act(() => result.current.startTracking())

    sendPosition(makePosition(0, HIGH_SPEED_KMH, 0))
    sendPosition(makePosition(1, HIGH_SPEED_KMH, 1_000))
    const distBeforePause = result.current.distanceMeters

    act(() => { result.current.pauseTracking() })
    act(() => { result.current.resumeTracking({ lat: 59.001, lon: 18.0 }) })

    sendPosition(makePosition(2, HIGH_SPEED_KMH, 2_000))

    expect(result.current.distanceMeters).toBeGreaterThan(distBeforePause)
  })

  it('resumeTracking uses the saved position as the reference, not the latest GPS position', () => {
    const { result } = renderHook(() => useGpsTracking())
    act(() => result.current.startTracking())

    sendPosition(makePosition(0, HIGH_SPEED_KMH, 0))
    sendPosition(makePosition(1, HIGH_SPEED_KMH, 1_000))
    const distAfterDriving = result.current.distanceMeters

    // Pause — savedPos is at step 1 (lat 59.001)
    let savedPos: ReturnType<typeof result.current.pauseTracking> = null
    act(() => { savedPos = result.current.pauseTracking() })

    // GPS fires at step 2 while paused — must not accumulate distance
    sendPosition(makePosition(2, HIGH_SPEED_KMH, 2_000))
    expect(result.current.distanceMeters).toBeCloseTo(distAfterDriving, 2)

    // Resume from the saved position (step 1, lat 59.001)
    act(() => { result.current.resumeTracking(savedPos!) })

    // Step 3 (lat 59.003): distance from saved pos (59.001→59.003) ≈ 222 m
    // If lastPos was step 2 (59.002) instead, it would only be ≈ 111 m
    sendPosition(makePosition(3, HIGH_SPEED_KMH, 3_000))
    const addedAfterResume = result.current.distanceMeters - distAfterDriving
    expect(addedAfterResume).toBeGreaterThan(150) // must be ~222 m, not ~111 m
  })

  it('stopTracking resets isPaused to false', () => {
    const { result } = renderHook(() => useGpsTracking())
    act(() => result.current.startTracking())
    sendPosition(makePosition(0, HIGH_SPEED_KMH, 0))
    act(() => { result.current.pauseTracking() })
    act(() => { result.current.stopTracking() })
    expect(result.current.isPaused).toBe(false)
    expect(result.current.isTracking).toBe(false)
  })

  it('currentSpeedKmh is updated from GPS speed', () => {
    const { result } = renderHook(() => useGpsTracking())
    act(() => result.current.startTracking())
    sendPosition(makePosition(0, HIGH_SPEED_KMH, 0))
    expect(result.current.currentSpeedKmh).toBeCloseTo(HIGH_SPEED_KMH, 0)
  })

  it('currentSpeedKmh reflects low speed while paused', () => {
    const { result } = renderHook(() => useGpsTracking())
    act(() => result.current.startTracking())
    sendPosition(makePosition(0, HIGH_SPEED_KMH, 0))
    act(() => { result.current.pauseTracking() })
    sendPosition(makePosition(1, LOW_SPEED_KMH, 1_000))
    expect(result.current.currentSpeedKmh).toBeCloseTo(LOW_SPEED_KMH, 0)
  })

  it('does not reset pause state for a short gap (< 6 min)', () => {
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

    // Short gap (20 s) — well under the 6-min threshold, must not reset paused state
    t += 20_000
    sendPosition(makePosition(1, LOW_SPEED_KMH, t))

    expect(result.current.distanceMeters).toBeCloseTo(distWhenPaused, 0)
  })
})
