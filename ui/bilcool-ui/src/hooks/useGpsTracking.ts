import { useState, useEffect, useRef } from 'react'

interface TrackingState {
  isTracking: boolean
  distanceMeters: number
  currentPosition: { lat: number; lon: number } | null
  error: string | null
}

const LOW_SPEED_THRESHOLD_KMH = 7
const LOW_SPEED_PAUSE_MS = 5 * 60 * 1000 // 5 minutes

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
    distanceMeters: 0,
    currentPosition: null,
    error: null,
  })

  const watchIdRef = useRef<number | null>(null)
  // last accepted position (lat, lon, timestamp)
  const lastPosRef = useRef<{ lat: number; lon: number; ts: number } | null>(null)
  // when speed first dropped below threshold; null means speed is ok
  const lowSpeedSinceRef = useRef<number | null>(null)
  // whether accumulation is paused due to sustained low speed
  const isPausedRef = useRef(false)
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
    isPausedRef.current = false
    lowSpeedAccumRef.current = 0

    const watchId = navigator.geolocation.watchPosition(
      (pos) => {
        const { latitude: lat, longitude: lon, speed } = pos.coords
        const now = pos.timestamp

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

        if (speedKmh >= LOW_SPEED_THRESHOLD_KMH) {
          // Speed ok: reset low-speed timer and resume accumulation
          lowSpeedSinceRef.current = null
          lowSpeedAccumRef.current = 0
          isPausedRef.current = false

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
            if (!isPausedRef.current) {
              // Threshold just crossed: retroactively remove distance accumulated during low-speed window
              distanceRef.current = Math.max(0, distanceRef.current - lowSpeedAccumRef.current)
              lowSpeedAccumRef.current = 0
              isPausedRef.current = true
            }
          } else if (!isPausedRef.current && lastPosRef.current) {
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
          error: null,
        }))
      },
      (err) => setState((s) => ({ ...s, error: err.message })),
      { enableHighAccuracy: true, maximumAge: 0 },
    )

    watchIdRef.current = watchId
    setState((s) => ({ ...s, isTracking: true, distanceMeters: 0, error: null }))
  }

  function stopTracking() {
    if (watchIdRef.current !== null) {
      navigator.geolocation.clearWatch(watchIdRef.current)
      watchIdRef.current = null
    }
    setState((s) => ({ ...s, isTracking: false }))
  }

  useEffect(() => {
    return () => {
      if (watchIdRef.current !== null) {
        navigator.geolocation.clearWatch(watchIdRef.current)
      }
    }
  }, [])

  return { ...state, startTracking, stopTracking }
}
