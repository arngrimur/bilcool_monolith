import { useState, useRef } from 'react'
import FullCalendar from '@fullcalendar/react'
import dayGridPlugin from '@fullcalendar/daygrid'
import timeGridPlugin from '@fullcalendar/timegrid'
import interactionPlugin from '@fullcalendar/interaction'
import type { DateSelectArg, EventClickArg, EventDropArg, EventInput } from '@fullcalendar/core'
import type { EventResizeDoneArg } from '@fullcalendar/interaction'
import enGbLocale from '@fullcalendar/core/locales/en-gb'
import svLocale from '@fullcalendar/core/locales/sv'
import { useSettingsStore } from '../../stores/settingsStore'
import { useAuthStore } from '../../stores/authStore'
import { useBookings, useUpsertBooking } from '../../hooks/useBookings'
import { useUsers } from '../../hooks/useUsers'
import BookingForm from '../bookings/BookingForm'
import BookingStartEndDialog from '../bookings/BookingStartEndDialog'
import { Button } from '../ui/button'
import type { BookingResponse } from '../../types/api'
import { snapToQuarterHour } from '../../utils/bookingUtils'
import { getColorForUserRef } from '../../utils/bookingColors'
import CalendarEvent from './CalendarEvent'

type ViewType = 'dayGridMonth' | 'timeGridWeek' | 'timeGridDay'

function useIsMobile() {
  return window.innerWidth <= 768
}

interface CalendarViewProps {
  completedBookingMap: Map<string, number>
}

export default function CalendarView({ completedBookingMap }: CalendarViewProps) {
  const language = useSettingsStore((s) => s.language)
  const bookingColor = useSettingsStore((s) => s.bookingColor)
  const userRef = useAuthStore((s) => s.userRef)
  const { data: bookings = [] } = useBookings()

  const uniqueUserRefs = [...new Set(bookings.map((b) => b.user_ref))]
  const userQueries = useUsers(uniqueUserRefs)
  const userMap = new Map(
    userQueries
      .map((q) => q.data)
      .filter(Boolean)
      .map((u) => [u!.user_ref, u!.username])
  )

  const upsert = useUpsertBooking()

  const isMobile = useIsMobile()
  const [view, setView] = useState<ViewType>(isMobile ? 'timeGridDay' : 'timeGridWeek')
  const calendarRef = useRef<FullCalendar>(null)

  const [formOpen, setFormOpen] = useState(false)
  const [formStart, setFormStart] = useState<Date>()
  const [formEnd, setFormEnd] = useState<Date>()
  const [editingBooking, setEditingBooking] = useState<BookingResponse>()

  const [detailOpen, setDetailOpen] = useState(false)
  const [selectedBooking, setSelectedBooking] = useState<BookingResponse>()

  const events: EventInput[] = bookings.map((b) => {
    const isOwn = b.user_ref === userRef
    const username = userMap.get(b.user_ref) ?? b.user_ref
    const isFuture = new Date(b.start_date) > new Date()
    return {
      id: b.booking_reference,
      title: username,
      start: b.start_date,
      end: b.end_date,
      color: isOwn ? bookingColor : getColorForUserRef(b.user_ref, bookingColor),
      editable: isOwn && isFuture,
      extendedProps: { booking: b, isOwn },
    }
  })

  function handleSelect(arg: DateSelectArg) {
    setFormStart(snapToQuarterHour(arg.start))
    setFormEnd(snapToQuarterHour(arg.end))
    setEditingBooking(undefined)
    setFormOpen(true)
  }

  function handleEventClick(arg: EventClickArg) {
    const booking = arg.event.extendedProps['booking'] as BookingResponse
    const isOwn = arg.event.extendedProps['isOwn'] as boolean
    if (!isOwn) return
    setSelectedBooking(booking)
    setDetailOpen(true)
  }

  async function handleEventDrop(arg: EventDropArg) {
    const booking = arg.event.extendedProps['booking'] as BookingResponse
    if (!arg.event.start || !arg.event.end) { arg.revert(); return }
    try {
      await upsert.mutateAsync({
        user_ref: booking.user_ref,
        booking_reference: booking.booking_reference,
        start_date: arg.event.start.toISOString(),
        end_date: arg.event.end.toISOString(),
      })
    } catch {
      arg.revert()
    }
  }

  async function handleEventResize(arg: EventResizeDoneArg) {
    const booking = arg.event.extendedProps['booking'] as BookingResponse
    if (!arg.event.start || !arg.event.end) { arg.revert(); return }
    try {
      await upsert.mutateAsync({
        user_ref: booking.user_ref,
        booking_reference: booking.booking_reference,
        start_date: arg.event.start.toISOString(),
        end_date: arg.event.end.toISOString(),
      })
    } catch {
      arg.revert()
    }
  }

  function handleEditBooking() {
    if (!selectedBooking) return
    setEditingBooking(selectedBooking)
    setFormStart(new Date(selectedBooking.start_date))
    setFormEnd(new Date(selectedBooking.end_date))
    setFormOpen(true)
  }

  function changeView(v: ViewType) {
    setView(v)
    calendarRef.current?.getApi().changeView(v)
  }

  const locale = language === 'sv' ? svLocale : enGbLocale

  return (
    <div className="space-y-4">
      <div className="flex gap-2 flex-wrap" role="group" aria-label="Calendar view switcher">
        <Button
          variant={view === 'dayGridMonth' ? 'default' : 'outline'}
          size="sm"
          onClick={() => changeView('dayGridMonth')}
          className="min-h-[44px]"
        >
          Month
        </Button>
        <Button
          variant={view === 'timeGridWeek' ? 'default' : 'outline'}
          size="sm"
          onClick={() => changeView('timeGridWeek')}
          className="min-h-[44px]"
        >
          Week
        </Button>
        <Button
          variant={view === 'timeGridDay' ? 'default' : 'outline'}
          size="sm"
          onClick={() => changeView('timeGridDay')}
          className="min-h-[44px]"
        >
          Day
        </Button>
      </div>

      <div className="rounded-lg border bg-card overflow-hidden">
        <FullCalendar
          ref={calendarRef}
          plugins={[dayGridPlugin, timeGridPlugin, interactionPlugin]}
          initialView={view}
          locale={locale}
          events={events}
          selectable
          selectMirror
          slotDuration="00:15:00"
          slotMinTime="06:00:00"
          slotMaxTime="22:00:00"
          allDaySlot={false}
          headerToolbar={{ left: 'prev,next today', center: 'title', right: '' }}
          select={handleSelect}
          eventClick={handleEventClick}
          eventDrop={handleEventDrop}
          eventResize={handleEventResize}
          editable
          eventContent={(arg) => <CalendarEvent {...arg} />}
          height="auto"
        />
      </div>

      <BookingForm
        open={formOpen}
        onOpenChange={setFormOpen}
        initialStart={formStart}
        initialEnd={formEnd}
        editingBooking={editingBooking}
        allBookings={bookings}
      />

      {selectedBooking && (
        <BookingStartEndDialog
          open={detailOpen}
          onOpenChange={setDetailOpen}
          booking={selectedBooking}
          hasDistance={completedBookingMap.has(selectedBooking.booking_reference)}
          distanceKm={completedBookingMap.get(selectedBooking.booking_reference)}
          onEdit={handleEditBooking}
        />
      )}
    </div>
  )
}
