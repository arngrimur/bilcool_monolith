import { useTranslation } from 'react-i18next'
import { useGpsTracking } from '../../hooks/useGpsTracking'
import { useEndBooking, usePauseBooking, useResumeBooking, useAddTrackPoints } from '../../hooks/useBookings'
import { Button } from '../ui/button'
import type { BookingResponse, TrackPoint } from '../../types/api'
import { formatTime } from '../../utils/dateUtils'
import { useSettingsStore } from '../../stores/settingsStore'

interface Props {
  booking: BookingResponse
  allBookings: BookingResponse[]
}

export default function ActiveBookingTracker({ booking, allBookings }: Props) {
  const { t } = useTranslation('bookings')
  const language = useSettingsStore((s) => s.language)
  const endBooking = useEndBooking()
  const pauseBooking = usePauseBooking()
  const resumeBooking = useResumeBooking()
  const addTrackPoints = useAddTrackPoints()

  const onPoint = async (point: TrackPoint) => {
    await addTrackPoints.mutateAsync({ id: booking.booking_reference, points: [point] })
  }

  const { isTracking, isPaused, currentPosition, error, startTracking, stopTracking, pauseTracking, resumeTracking } =
    useGpsTracking({ onPoint })

  const now = new Date()
  const hasOtherActive = allBookings.some(
    (b) =>
      b.booking_reference !== booking.booking_reference &&
      new Date(b.start_date) <= now &&
      new Date(b.end_date) > now &&
      !b.distance,
  )

  async function handleStop() {
    stopTracking()
    await endBooking.mutateAsync({
      id: booking.booking_reference,
      body: {
        position: currentPosition ?? undefined,
      },
    })
  }

  async function handlePause() {
    const pos = pauseTracking()
    if (pos) {
      await pauseBooking.mutateAsync({
        id: booking.booking_reference,
        body: { lat: pos.lat, lon: pos.lon },
      })
    }
  }

  async function handleResume() {
    await resumeBooking.mutateAsync(booking.booking_reference)
    resumeTracking()
  }

  return (
    <div className="rounded-lg border-2 border-primary bg-primary/5 p-4 space-y-3">
      <div className="flex items-center justify-between">
        <h2 className="font-semibold text-base">{t('active_booking_title')}</h2>
        <span className="text-xs font-medium text-primary bg-primary/10 px-2 py-0.5 rounded-full">
          {isPaused ? t('status_paused') : t('status_active')}
        </span>
      </div>

      <p className="text-sm text-muted-foreground">
        {formatTime(booking.start_date, language)} – {formatTime(booking.end_date, language)}
      </p>

      {error && <p className="text-sm text-destructive">{error}</p>}

      {isTracking ? (
        <div className="space-y-3">
          {!isPaused && (
            <p className="text-sm text-center text-muted-foreground py-2">{t('tracking_active_label')}</p>
          )}

          {isPaused ? (
            <Button
              className="w-full min-h-[52px] text-base"
              onClick={handleResume}
              disabled={resumeBooking.isPending}
            >
              {resumeBooking.isPending ? '...' : t('resume_tracking')}
            </Button>
          ) : (
            <div className="flex gap-2">
              <Button
                className="flex-1 min-h-[52px] text-base"
                variant="outline"
                onClick={handlePause}
                disabled={pauseBooking.isPending}
              >
                {pauseBooking.isPending ? '...' : t('pause_tracking')}
              </Button>
              <Button
                className="flex-1 min-h-[52px] text-base"
                variant="destructive"
                onClick={handleStop}
                disabled={endBooking.isPending}
              >
                {endBooking.isPending ? '...' : t('stop_tracking')}
              </Button>
            </div>
          )}
        </div>
      ) : (
        <div className="space-y-2">
          {hasOtherActive ? (
            <p className="text-sm text-muted-foreground">{t('other_booking_active')}</p>
          ) : (
            <Button className="w-full min-h-[52px] text-base" onClick={startTracking}>
              {t('start_tracking')}
            </Button>
          )}
        </div>
      )}
    </div>
  )
}
