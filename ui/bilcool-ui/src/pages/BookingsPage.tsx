import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { useBookings } from '../hooks/useBookings'
import { useAllUsers } from '../hooks/useUsers'
import { useAuthStore } from '../stores/authStore'
import { useSettingsStore } from '../stores/settingsStore'
import type { BookingResponse } from '../types/api'
import { formatDate, formatTime, formatMonthYear, formatMonthKey } from '../utils/dateUtils'
import { listFinishedBookings } from '../api/events'

type BookingStatus = 'upcoming' | 'active' | 'overdue'

function getStatus(booking: BookingResponse): BookingStatus {
  const now = new Date()
  const start = new Date(booking.start_date)
  const end = new Date(booking.end_date)
  if (start <= now && now < end) return 'active'
  if (end < now) return 'overdue'
  return 'upcoming'
}

const currentYear = new Date().getFullYear()
const currentMonth = String(new Date().getMonth() + 1)

export default function BookingsPage() {
  const { t } = useTranslation('bookings')
  const language = useSettingsStore((s) => s.language)
  const userRef = useAuthStore((s) => s.userRef)

  const [selectedYear, setSelectedYear] = useState<number>(currentYear)
  const [selectedMonth, setSelectedMonth] = useState<string>(currentMonth)
  const [selectedUser, setSelectedUser] = useState<string>('all')

  const userFilter = selectedUser !== 'all' ? selectedUser : undefined

  const { data: allBookings = [] } = useBookings()
  const nonCompleted = allBookings.filter((b) => !b.distance)

  const summaryParams = { year: selectedYear, user_ref: userFilter }
  const { data: finishedBookingsAll = [] } = useQuery({
    queryKey: ['finishedBookings', summaryParams],
    queryFn: () => listFinishedBookings(summaryParams),
    staleTime: 30_000,
  })

  const listParams = {
    year: selectedYear,
    month: selectedMonth ? Number(selectedMonth) : undefined,
    user_ref: userFilter,
  }
  const { data: finishedBookings = [] } = useQuery({
    queryKey: ['finishedBookings', listParams],
    queryFn: () => listFinishedBookings(listParams),
    staleTime: 30_000,
  })

  const { data: allUsers = [] } = useAllUsers()
  const userMap = new Map(allUsers.map((u) => [u.user_ref, u.username]))

  const availableYears = [
    ...new Set([
      currentYear,
      ...allBookings.map((b) => new Date(b.start_date).getFullYear()),
      ...finishedBookingsAll.map((b) => new Date(b.start_date).getFullYear()),
    ]),
  ].sort((a, b) => b - a)

  const monthsInYear = [
    ...new Set([
      ...allBookings
        .filter((b) => new Date(b.start_date).getFullYear() === selectedYear)
        .map((b) => formatMonthKey(b.start_date).split('-')[1]),
      ...finishedBookingsAll.map((b) => formatMonthKey(b.start_date).split('-')[1]),
    ]),
  ].sort()

  const nonCompletedInYear = nonCompleted.filter(
    (b) => new Date(b.start_date).getFullYear() === selectedYear
  )

  const filteredNonCompleted = nonCompletedInYear.filter((b) => {
    if (selectedMonth && formatMonthKey(b.start_date).split('-')[1] !== selectedMonth) return false
    if (userFilter && b.user_ref !== userFilter) return false
    return true
  })

  const monthlySummary = monthsInYear.map((monthNum) => {
    const inMonth = (b: { start_date: string }) =>
      formatMonthKey(b.start_date).split('-')[1] === monthNum

    const myNonCompleted = nonCompletedInYear.filter(
      (b) => inMonth(b) && b.user_ref === userRef
    )
    const myFinished = finishedBookingsAll.filter(
      (b) => inMonth(b) && b.user_ref === (userRef ?? '')
    )
    const myKm = myFinished.reduce((sum, b) => sum + b.distance_meters / 1000, 0)

    return {
      monthKey: `${selectedYear}-${monthNum}`,
      label: formatMonthYear(new Date(`${selectedYear}-${monthNum}-01`), language),
      myCount: myNonCompleted.length + myFinished.length,
      totalCount: nonCompletedInYear.filter(inMonth).length + finishedBookingsAll.filter(inMonth).length,
      myKm,
    }
  })

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
            <label htmlFor="year-filter" className="text-sm font-medium mr-2">Year</label>
            <select
              id="year-filter"
              value={selectedYear}
              onChange={(e) => { setSelectedYear(Number(e.target.value)); setSelectedMonth(currentMonth) }}
              className="rounded-md border border-input bg-background px-3 py-1.5 text-sm"
            >
              {availableYears.map((y) => (
                <option key={y} value={y}>{y}</option>
              ))}
            </select>
          </div>

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
              <option value="">All months</option>
              {monthsInYear.map((m) => (
                <option key={m} value={m}>
                  {formatMonthYear(new Date(`${selectedYear}-${m}-01`), language)}
                </option>
              ))}
            </select>
          </div>

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
              {allUsers.map((u) => (
                <option key={u.user_ref} value={u.user_ref}>
                  {u.username}
                </option>
              ))}
            </select>
          </div>
        </div>

        {filteredNonCompleted.length === 0 && finishedBookings.length === 0 ? (
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
                  {finishedBookings.map((b) => (
                    <tr key={b.booking_reference} className="border-b last:border-0">
                      <td className="px-4 py-3">{userMap.get(b.user_ref) ?? b.user_ref}</td>
                      <td className="px-4 py-3">{formatDate(b.start_date, language)}</td>
                      <td className="px-4 py-3">{formatTime(b.start_date, language)}</td>
                      <td className="px-4 py-3">{formatTime(b.end_date, language)}</td>
                      <td className="px-4 py-3">{(b.distance_meters / 1000).toFixed(1)}</td>
                      <td className="px-4 py-3">{t('status_completed')}</td>
                    </tr>
                  ))}
                  {filteredNonCompleted
                    .sort((a, b) => new Date(b.start_date).getTime() - new Date(a.start_date).getTime())
                    .map((b) => (
                      <tr key={b.booking_reference} className="border-b last:border-0">
                        <td className="px-4 py-3">{userMap.get(b.user_ref) ?? b.user_ref}</td>
                        <td className="px-4 py-3">{formatDate(b.start_date, language)}</td>
                        <td className="px-4 py-3">{formatTime(b.start_date, language)}</td>
                        <td className="px-4 py-3">{formatTime(b.end_date, language)}</td>
                        <td className="px-4 py-3">—</td>
                        <td className="px-4 py-3">{t(`status_${getStatus(b)}`)}</td>
                      </tr>
                    ))}
                </tbody>
              </table>
            </div>

            <div className="md:hidden grid gap-3">
              {finishedBookings.map((b) => (
                <div key={b.booking_reference} className="rounded-lg border bg-card p-4 space-y-1 text-sm">
                  <div className="flex justify-between items-start">
                    <span className="font-medium">{userMap.get(b.user_ref) ?? b.user_ref}</span>
                    <span className="text-xs text-muted-foreground">{t('status_completed')}</span>
                  </div>
                  <p className="text-muted-foreground">{formatDate(b.start_date, language)}</p>
                  <p>{formatTime(b.start_date, language)} – {formatTime(b.end_date, language)}</p>
                  <p>{(b.distance_meters / 1000).toFixed(1)} km</p>
                </div>
              ))}
              {filteredNonCompleted
                .sort((a, b) => new Date(b.start_date).getTime() - new Date(a.start_date).getTime())
                .map((b) => (
                  <div key={b.booking_reference} className="rounded-lg border bg-card p-4 space-y-1 text-sm">
                    <div className="flex justify-between items-start">
                      <span className="font-medium">{userMap.get(b.user_ref) ?? b.user_ref}</span>
                      <span className="text-xs text-muted-foreground">{t(`status_${getStatus(b)}`)}</span>
                    </div>
                    <p className="text-muted-foreground">{formatDate(b.start_date, language)}</p>
                    <p>{formatTime(b.start_date, language)} – {formatTime(b.end_date, language)}</p>
                  </div>
                ))}
            </div>
          </>
        )}
      </section>
    </div>
  )
}
