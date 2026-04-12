import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useBookings } from '../hooks/useBookings'
import { useUsers } from '../hooks/useUsers'
import { useAuthStore } from '../stores/authStore'
import { useSettingsStore } from '../stores/settingsStore'
import type { BookingResponse } from '../types/api'
import { formatDate, formatTime, formatMonthYear, formatMonthKey } from '../utils/dateUtils'

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

export default function BookingsPage() {
  const { t } = useTranslation('bookings')
  const language = useSettingsStore((s) => s.language)
  const userRef = useAuthStore((s) => s.userRef)
  const role = useAuthStore((s) => s.role)

  const { data: bookings = [] } = useBookings()

  const completedBookingMap = new Map<string, number>()
  for (const b of bookings) {
    if (b.distance) {
      const km = (b.distance.end_distance - b.distance.start_distance) / 1000
      completedBookingMap.set(b.booking_reference, km)
    }
  }

  const uniqueUserRefs = [...new Set(bookings.map((b) => b.user_ref))]
  const userQueries = useUsers(uniqueUserRefs)
  const userMap = new Map(
    userQueries
      .map((q) => q.data)
      .filter(Boolean)
      .map((u) => [u!.user_ref, u!.username])
  )

  const allMonthKeys = [...new Set(bookings.map((b) => formatMonthKey(b.start_date)))].sort()

  const [selectedMonth, setSelectedMonth] = useState<string>(allMonthKeys[allMonthKeys.length - 1] ?? '')
  const [selectedUser, setSelectedUser] = useState<string>('all')

  const monthlySummary = allMonthKeys.map((monthKey) => {
    const inMonth = bookings.filter((b) => formatMonthKey(b.start_date) === monthKey)
    const myInMonth = inMonth.filter((b) => b.user_ref === userRef)
    const myKm = myInMonth.reduce((sum, b) => {
      const km = completedBookingMap.get(b.booking_reference)
      return sum + (km ?? 0)
    }, 0)
    return {
      monthKey,
      label: formatMonthYear(new Date(monthKey + '-01'), language),
      myCount: myInMonth.length,
      totalCount: inMonth.length,
      myKm,
    }
  })

  const filteredBookings = bookings
    .filter((b) => {
      if (selectedMonth && formatMonthKey(b.start_date) !== selectedMonth) return false
      if (role !== 'admin' && b.user_ref !== userRef) return false
      if (role === 'admin' && selectedUser !== 'all' && b.user_ref !== selectedUser) return false
      return true
    })
    .sort((a, b) => new Date(b.start_date).getTime() - new Date(a.start_date).getTime())

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">{t('title')}</h1>

      <section aria-labelledby="summary-heading">
        <h2 id="summary-heading" className="text-lg font-semibold mb-3">{t('summary_title')}</h2>
        <div className="rounded-lg border overflow-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b bg-muted/50">
                <th className="px-4 py-3 text-left font-medium">{t('summary_col_month')}</th>
                <th className="px-4 py-3 text-left font-medium">{t('summary_col_my_bookings')}</th>
                <th className="px-4 py-3 text-left font-medium">{t('summary_col_total')}</th>
                <th className="px-4 py-3 text-left font-medium">{t('summary_col_my_km')}</th>
              </tr>
            </thead>
            <tbody>
              {monthlySummary.map((row) => (
                <tr key={row.monthKey} className="border-b last:border-0">
                  <td className="px-4 py-3">{row.label}</td>
                  <td className="px-4 py-3">{row.myCount}</td>
                  <td className="px-4 py-3">{row.totalCount}</td>
                  <td className="px-4 py-3">{row.myKm.toFixed(1)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>

      <section aria-labelledby="list-heading">
        <h2 id="list-heading" className="text-lg font-semibold mb-3">{t('title')}</h2>

        <div className="flex flex-wrap gap-3 mb-4">
          <div>
            <label htmlFor="month-filter" className="text-sm font-medium mr-2">
              {t('filter_month')}
            </label>
            <select
              id="month-filter"
              value={selectedMonth}
              onChange={(e) => setSelectedMonth(e.target.value)}
              className="rounded-md border border-input bg-background px-3 py-1.5 text-sm"
            >
              <option value="">{t('filter_all_users')}</option>
              {allMonthKeys.map((mk) => (
                <option key={mk} value={mk}>
                  {formatMonthYear(new Date(mk + '-01'), language)}
                </option>
              ))}
            </select>
          </div>

          {role === 'admin' && (
            <div>
              <label htmlFor="user-filter" className="text-sm font-medium mr-2">
                {t('filter_user')}
              </label>
              <select
                id="user-filter"
                value={selectedUser}
                onChange={(e) => setSelectedUser(e.target.value)}
                className="rounded-md border border-input bg-background px-3 py-1.5 text-sm"
              >
                <option value="all">{t('filter_all_users')}</option>
                {uniqueUserRefs.map((ref) => (
                  <option key={ref} value={ref}>
                    {userMap.get(ref) ?? ref}
                  </option>
                ))}
              </select>
            </div>
          )}
        </div>

        {filteredBookings.length === 0 ? (
          <p className="text-muted-foreground text-sm">{t('no_bookings')}</p>
        ) : (
          <>
            <div className="hidden md:block rounded-lg border overflow-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b bg-muted/50">
                    <th className="px-4 py-3 text-left font-medium">{t('col_user')}</th>
                    <th className="px-4 py-3 text-left font-medium">{t('col_date')}</th>
                    <th className="px-4 py-3 text-left font-medium">{t('col_start')}</th>
                    <th className="px-4 py-3 text-left font-medium">{t('col_end')}</th>
                    <th className="px-4 py-3 text-left font-medium">{t('col_distance')}</th>
                    <th className="px-4 py-3 text-left font-medium">{t('col_status')}</th>
                  </tr>
                </thead>
                <tbody>
                  {filteredBookings.map((b) => {
                    const hasDistance = completedBookingMap.has(b.booking_reference)
                    const km = completedBookingMap.get(b.booking_reference)
                    const status = getStatus(b, hasDistance)
                    return (
                      <tr key={b.booking_reference} className="border-b last:border-0">
                        <td className="px-4 py-3">{userMap.get(b.user_ref) ?? b.user_ref}</td>
                        <td className="px-4 py-3">{formatDate(b.start_date, language)}</td>
                        <td className="px-4 py-3">{formatTime(b.start_date, language)}</td>
                        <td className="px-4 py-3">{formatTime(b.end_date, language)}</td>
                        <td className="px-4 py-3">{km !== undefined ? km.toFixed(1) : '—'}</td>
                        <td className="px-4 py-3">{t(`status_${status}`)}</td>
                      </tr>
                    )
                  })}
                </tbody>
              </table>
            </div>

            <div className="md:hidden grid gap-3">
              {filteredBookings.map((b) => {
                const hasDistance = completedBookingMap.has(b.booking_reference)
                const km = completedBookingMap.get(b.booking_reference)
                const status = getStatus(b, hasDistance)
                return (
                  <div key={b.booking_reference} className="rounded-lg border bg-card p-4 space-y-1 text-sm">
                    <div className="flex justify-between items-start">
                      <span className="font-medium">{userMap.get(b.user_ref) ?? b.user_ref}</span>
                      <span className="text-xs text-muted-foreground">{t(`status_${status}`)}</span>
                    </div>
                    <p className="text-muted-foreground">{formatDate(b.start_date, language)}</p>
                    <p>{formatTime(b.start_date, language)} – {formatTime(b.end_date, language)}</p>
                    {km !== undefined && <p>{km.toFixed(1)} km</p>}
                  </div>
                )
              })}
            </div>
          </>
        )}
      </section>
    </div>
  )
}
