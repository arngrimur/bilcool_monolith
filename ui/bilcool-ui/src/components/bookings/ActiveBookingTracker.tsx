import { useTranslation } from 'react-i18next'
import { useGpsTracking } from '../../hooks/useGpsTracking'
import { useEndBooking } from '../../hooks/useBookings'
import { Button } from '../ui/button'
import type { BookingResponse } from '../../types/api'
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
  const { isTracking, distanceMeters, currentPosition, error, startTracking, stopTracking } =
    useGpsTracking()

  const now = new Date()
  const hasOtherActive = allBookings.some(
    (b) =>
      b.booking_reference !== booking.booking_reference &&
      new Date(b.start_date) <= now &&
      !b.distance,
  )

  async function handleStop() {
    stopTracking()
    await endBooking.mutateAsync({
      id: booking.booking_reference,
      body: {
        start_distance: 0,
        end_distance: Math.round(distanceMeters),
        position: currentPosition ?? undefined,
      },
    })
  }

  return (
    <div className="rounded-lg border-2 border-primary bg-primary/5 p-4 space-y-3">
      <div className="flex items-center justify-between">
        <h2 className="font-semibold text-base">{t('active_booking_title')}</h2>
        <span className="text-xs font-medium text-primary bg-primary/10 px-2 py-0.5 rounded-full">
          {t('status_active')}
        </span>
      </div>

      <p className="text-sm text-muted-foreground">
        {formatTime(booking.start_date, language)} – {formatTime(booking.end_date, language)}
      </p>

      {error && (
        <p className="text-sm text-destructive">{error}</p>
      )}

      {isTracking ? (
        <div className="space-y-3">
          <div className="text-center py-2">
            <p className="text-3xl font-bold tabular-nums">
              {(distanceMeters / 1000).toFixed(2)} km
            </p>
            <p className="text-xs text-muted-foreground mt-1">{t('tracking_distance_label')}</p>
          </div>
          <Button
            className="w-full min-h-[52px] text-base"
            variant="destructive"
            onClick={handleStop}
            disabled={endBooking.isPending}
          >
            {endBooking.isPending ? '...' : t('stop_tracking')}
          </Button>
        </div>
      ) : (
        <div className="space-y-2">
          {hasOtherActive ? (
            <p className="text-sm text-muted-foreground">{t('other_booking_active')}</p>
          ) : (
            <Button
              className="w-full min-h-[52px] text-base"
              onClick={startTracking}
            >
              {t('start_tracking')}
            </Button>
          )}
        </div>
      )}
    </div>
  )
}
