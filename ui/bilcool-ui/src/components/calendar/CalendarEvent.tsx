import type { EventContentArg } from '@fullcalendar/core'

export default function CalendarEvent({ event }: EventContentArg) {
  return (
    <div
      className="px-1 py-0.5 text-xs font-medium truncate w-full"
      aria-label={`Booking: ${event.title}`}
    >
      {event.title}
    </div>
  )
}
