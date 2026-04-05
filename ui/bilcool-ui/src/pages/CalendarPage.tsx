import { useQuery } from '@tanstack/react-query'
import { listEvents } from '../api/events'
import CalendarView from '../components/calendar/CalendarView'
import type { CompletedBookingPayload } from '../types/api'

export default function CalendarPage() {
  const { data: events = [] } = useQuery({
    queryKey: ['events', { producer: 'bookings', event_type: 'booking_ended' }],
    queryFn: () => listEvents({ producer: 'bookings', event_type: 'booking_ended' }),
  })

  const completedBookingMap = new Map<string, number>()
  for (const event of events) {
    const payload = event.payload as CompletedBookingPayload
    if (payload?.booking?.booking_reference && payload?.distance) {
      const km = (payload.distance.end_distance - payload.distance.start_distance) / 1000
      completedBookingMap.set(payload.booking.booking_reference, km)
    }
  }

  return <CalendarView completedBookingMap={completedBookingMap} />
}
