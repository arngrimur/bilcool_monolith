import { useState, useEffect, useRef, useCallback } from 'react'
import type { TrackPoint } from '../types/api'

const SAMPLE_INTERVAL_MS = 2 * 60 * 1000 // 2 minutes

interface TrackingState {
  isTracking: boolean
  isPaused: boolean
  currentPosition: { lat: number; lon: number } | null
  error: string | null
}

interface UseGpsTrackingOptions {
  onPoint?: (point: TrackPoint) => Promise<void>
}

export function useGpsTracking({ onPoint }: UseGpsTrackingOptions = {}) {
  const [state, setState] = useState<TrackingState>({
    isTracking: false,
    isPaused: false,
    currentPosition: null,
    error: null,
  })

  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null)
  const onPointRef = useRef(onPoint)
  onPointRef.current = onPoint

  const samplePosition = useCallback(() => {
    if (!navigator.geolocation) return
    navigator.geolocation.getCurrentPosition(
      (pos) => {
        const point: TrackPoint = {
          lat: pos.coords.latitude,
          lon: pos.coords.longitude,
          recorded_at: new Date(pos.timestamp).toISOString(),
        }
        setState((s) => ({ ...s, currentPosition: { lat: point.lat, lon: point.lon }, error: null }))
        onPointRef.current?.(point).catch(() => {
          // upload failure is non-fatal — point may be retried next interval
        })
      },
      (err) => setState((s) => ({ ...s, error: err.message })),
      { enableHighAccuracy: true, maximumAge: 30_000 },
    )
  }, [])

  const startInterval = useCallback(() => {
    samplePosition() // immediate first sample
    intervalRef.current = setInterval(samplePosition, SAMPLE_INTERVAL_MS)
  }, [samplePosition])

  const stopInterval = useCallback(() => {
    if (intervalRef.current !== null) {
      clearInterval(intervalRef.current)
      intervalRef.current = null
    }
  }, [])

  function startTracking() {
    if (!navigator.geolocation) {
      setState((s) => ({ ...s, error: 'GPS not supported on this device' }))
      return
    }
    setState((s) => ({ ...s, isTracking: true, isPaused: false, error: null }))
    startInterval()
  }

  function stopTracking() {
    stopInterval()
    setState((s) => ({ ...s, isTracking: false, isPaused: false }))
  }

  function pauseTracking(): { lat: number; lon: number } | null {
    stopInterval()
    setState((s) => ({ ...s, isPaused: true }))
    return state.currentPosition
  }

  function resumeTracking(_savedPosition?: { lat: number; lon: number }) {
    setState((s) => ({ ...s, isPaused: false }))
    startInterval()
  }

  useEffect(() => {
    return () => {
      stopInterval()
    }
  }, [stopInterval])

  return { ...state, startTracking, stopTracking, pauseTracking, resumeTracking }
}
