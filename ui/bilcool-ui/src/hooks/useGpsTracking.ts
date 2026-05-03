import { useState, useEffect, useRef } from 'react'

interface TrackingState {
  isTracking: boolean
  isPaused: boolean
  distanceMeters: number
  currentPosition: { lat: number; lon: number } | null
  currentSpeedKmh: number
  error: string | null
}

const LOW_SPEED_THRESHOLD_KMH = 7
const LOW_SPEED_PAUSE_MS = 5 * 60 * 1000 // 5 minutes
// A gap this large between callbacks means the screen was locked (iOS suspends watchPosition)
const SCREEN_LOCK_GAP_MS = 6 * 60 * 1000

function haversineMeters(lat1: number, lon1: number, lat2: number, lon2: number): number {
  const R = 6371000
  const φ1 = (lat1 * Math.PI) / 180
  const φ2 = (lat2 * Math.PI) / 180
  const Δφ = ((lat2 - lat1) * Math.PI) / 180
  const Δλ = ((lon2 - lon1) * Math.PI) / 180
  const a = Math.sin(Δφ / 2) ** 2 + Math.cos(φ1) * Math.cos(φ2) * Math.sin(Δλ / 2) ** 2
  return R * 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1 - a))
}

export function useGpsTracking() {
  const [state, setState] = useState<TrackingState>({
    isTracking: false,
    isPaused: false,
    distanceMeters: 0,
    currentPosition: null,
    currentSpeedKmh: 0,
    error: null,
  })

  const watchIdRef = useRef<number | null>(null)
  // last accepted position (lat, lon, timestamp)
  const lastPosRef = useRef<{ lat: number; lon: number; ts: number } | null>(null)
  // when speed first dropped below threshold; null means speed is ok
  const lowSpeedSinceRef = useRef<number | null>(null)
  // whether accumulation is auto-paused due to sustained low speed
  const isAutoPausedRef = useRef(false)
  // whether the user has manually paused tracking
  const isManuallyPausedRef = useRef(false)
  const distanceRef = useRef(0)
  // distance accumulated since speed dropped below threshold (may be retroactively removed)
  const lowSpeedAccumRef = useRef(0)

  function startTracking() {
    if (!navigator.geolocation) {
      setState((s) => ({ ...s, error: 'GPS not supported on this device' }))
      return
    }
    distanceRef.current = 0
    lastPosRef.current = null
    lowSpeedSinceRef.current = null
    isAutoPausedRef.current = false
    isManuallyPausedRef.current = false
    lowSpeedAccumRef.current = 0

    const watchId = navigator.geolocation.watchPosition(
      (pos) => {
        const { latitude: lat, longitude: lon, speed } = pos.coords
        const now = pos.timestamp

        // When the screen is locked iOS stops firing callbacks. On unlock the gap between
        // this update and the last one will be large. Credit the straight-line displacement
        // unconditionally (the user definitely moved) and reset pause state. Return early
        // so the normal speed logic does not double-count this segment or mark it tentative.
        if (lastPosRef.current && now - lastPosRef.current.ts > SCREEN_LOCK_GAP_MS) {
          if (!isManuallyPausedRef.current) {
            const gapDist = haversineMeters(lastPosRef.current.lat, lastPosRef.current.lon, lat, lon)
            distanceRef.current += gapDist
          }
          isAutoPausedRef.current = false
          lowSpeedSinceRef.current = null
          lowSpeedAccumRef.current = 0
          lastPosRef.current = { lat, lon, ts: now }
          setState((s) => ({
            ...s,
            distanceMeters: distanceRef.current,
            currentPosition: { lat, lon },
            error: null,
          }))
          return
        }

        // Derive speed in km/h; fall back to calculating from consecutive points
        let speedKmh: number
        if (speed !== null) {
          speedKmh = speed * 3.6
        } else if (lastPosRef.current) {
          const dt = (now - lastPosRef.current.ts) / 1000 // seconds
          const dist = haversineMeters(lastPosRef.current.lat, lastPosRef.current.lon, lat, lon)
          speedKmh = dt > 0 ? (dist / dt) * 3.6 : 0
        } else {
          speedKmh = 0
        }

        // Manual pause: only track position, never accumulate distance
        if (isManuallyPausedRef.current) {
          lastPosRef.current = { lat, lon, ts: now }
          setState((s) => ({
            ...s,
            currentPosition: { lat, lon },
            currentSpeedKmh: speedKmh,
            error: null,
          }))
          return
        }

        if (speedKmh >= LOW_SPEED_THRESHOLD_KMH) {
          // Speed ok: reset low-speed timer and resume auto-pause state
          lowSpeedSinceRef.current = null
          lowSpeedAccumRef.current = 0
          isAutoPausedRef.current = false

          if (lastPosRef.current) {
            const dist = haversineMeters(lastPosRef.current.lat, lastPosRef.current.lon, lat, lon)
            distanceRef.current += dist
          }
        } else {
          // Speed below threshold
          if (lowSpeedSinceRef.current === null) {
            lowSpeedSinceRef.current = now
          }
          const lowSpeedDuration = now - lowSpeedSinceRef.current
          if (lowSpeedDuration >= LOW_SPEED_PAUSE_MS) {
            if (!isAutoPausedRef.current) {
              // Threshold just crossed: retroactively remove distance accumulated during low-speed window
              distanceRef.current = Math.max(0, distanceRef.current - lowSpeedAccumRef.current)
              lowSpeedAccumRef.current = 0
              isAutoPausedRef.current = true
            }
          } else if (!isAutoPausedRef.current && lastPosRef.current) {
            // Low speed but under the 5-minute threshold: accumulate tentatively
            const dist = haversineMeters(lastPosRef.current.lat, lastPosRef.current.lon, lat, lon)
            distanceRef.current += dist
            lowSpeedAccumRef.current += dist
          }
        }

        lastPosRef.current = { lat, lon, ts: now }

        setState((s) => ({
          ...s,
          distanceMeters: distanceRef.current,
          currentPosition: { lat, lon },
          currentSpeedKmh: speedKmh,
          error: null,
        }))
      },
      (err) => setState((s) => ({ ...s, error: err.message })),
      { enableHighAccuracy: true, maximumAge: 0 },
    )

    watchIdRef.current = watchId
    setState((s) => ({ ...s, isTracking: true, isPaused: false, distanceMeters: 0, currentSpeedKmh: 0, error: null }))
  }

  function stopTracking(): number {
    if (watchIdRef.current !== null) {
      navigator.geolocation.clearWatch(watchIdRef.current)
      watchIdRef.current = null
    }
    // Discard any tentative low-speed distance that never crossed the 5-minute threshold.
    // This prevents a short walk (< 5 min) from being counted when the user stops tracking.
    const finalDistance = Math.max(0, distanceRef.current - lowSpeedAccumRef.current)
    distanceRef.current = finalDistance
    lowSpeedAccumRef.current = 0
    isManuallyPausedRef.current = false
    setState((s) => ({ ...s, isTracking: false, isPaused: false, distanceMeters: finalDistance }))
    return finalDistance
  }

  // Manually pause tracking. Returns the current GPS position (to be saved to backend).
  function pauseTracking(): { lat: number; lon: number } | null {
    isManuallyPausedRef.current = true
    // Discard tentative low-speed distance
    distanceRef.current = Math.max(0, distanceRef.current - lowSpeedAccumRef.current)
    lowSpeedAccumRef.current = 0
    isAutoPausedRef.current = false
    lowSpeedSinceRef.current = null

    const pos = lastPosRef.current ? { lat: lastPosRef.current.lat, lon: lastPosRef.current.lon } : null
    setState((s) => ({ ...s, isPaused: true, distanceMeters: distanceRef.current }))
    return pos
  }

  // Resume tracking from a saved position. The next GPS reading will measure distance from savedPosition.
  function resumeTracking(savedPosition: { lat: number; lon: number }) {
    isManuallyPausedRef.current = false
    // Use the saved pause position as the reference point so distance is measured correctly
    lastPosRef.current = { lat: savedPosition.lat, lon: savedPosition.lon, ts: Date.now() }
    lowSpeedSinceRef.current = null
    isAutoPausedRef.current = false
    lowSpeedAccumRef.current = 0
    setState((s) => ({ ...s, isPaused: false }))
  }

  useEffect(() => {
    return () => {
      if (watchIdRef.current !== null) {
        navigator.geolocation.clearWatch(watchIdRef.current)
      }
    }
  }, [])

  return { ...state, startTracking, stopTracking, pauseTracking, resumeTracking }
}
