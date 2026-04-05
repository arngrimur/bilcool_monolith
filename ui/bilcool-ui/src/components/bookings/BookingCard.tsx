import { useTranslation } from 'react-i18next'
import { Badge } from '../ui/badge'
import type { BookingResponse } from '../../types/api'
import { formatDate, formatTime } from '../../utils/dateUtils'
import { useSettingsStore } from '../../stores/settingsStore'

type BookingStatus = 'upcoming' | 'active' | 'completed' | 'overdue'

function getStatus(booking: BookingResponse, hasDistance: boolean): BookingStatus {
  const now = new Date()
  const start = new Date(booking.start_date)
  const end = new Date(booking.end_date)
  if (hasDistance) return 'completed'
  if (start <= now && now < end) return 'active'
  if (end < now) return 'overdue'
  return 'upcoming'
}

const statusVariant: Record<BookingStatus, 'default' | 'secondary' | 'destructive' | 'outline'> = {
  upcoming: 'secondary',
  active: 'default',
  completed: 'outline',
  overdue: 'destructive',
}

interface BookingCardProps {
  booking: BookingResponse
  username?: string
  hasDistance: boolean
  distanceKm?: number
  onClick?: () => void
}

export default function BookingCard({ booking, username, hasDistance, distanceKm, onClick }: BookingCardProps) {
  const { t } = useTranslation('bookings')
  const language = useSettingsStore((s) => s.language)
  const status = getStatus(booking, hasDistance)

  return (
    <div
      className="rounded-lg border bg-card p-4 space-y-2 cursor-pointer hover:bg-accent/50 transition-colors"
      onClick={onClick}
      role={onClick ? 'button' : undefined}
      tabIndex={onClick ? 0 : undefined}
      onKeyDown={onClick ? (e) => e.key === 'Enter' && onClick() : undefined}
      aria-label={`Booking by ${username ?? 'user'} on ${formatDate(booking.start_date, language)}`}
    >
      <div className="flex items-start justify-between gap-2">
        <div>
          {username && <p className="text-sm font-medium">{username}</p>}
          <p className="text-sm text-muted-foreground">
            {formatDate(booking.start_date, language)}
          </p>
        </div>
        <Badge variant={statusVariant[status]}>{t(`status_${status}`)}</Badge>
      </div>
      <div className="text-sm text-muted-foreground">
        {formatTime(booking.start_date, language)} – {formatTime(booking.end_date, language)}
      </div>
      {hasDistance && distanceKm !== undefined && (
        <p className="text-sm">{distanceKm.toFixed(1)} km</p>
      )}
    </div>
  )
}
