import { useBookings } from '../hooks/useBookings'
import CalendarView from '../components/calendar/CalendarView'

export default function CalendarPage() {
  const { data: bookings = [] } = useBookings()

  const completedBookingMap = new Map<string, number>()
  for (const b of bookings) {
    if (b.distance) {
      const km = (b.distance.end_distance - b.distance.start_distance) / 1000
      completedBookingMap.set(b.booking_reference, km)
    }
  }

  return <CalendarView completedBookingMap={completedBookingMap} />
}
