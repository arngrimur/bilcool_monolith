import { useState, useEffect } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { useTranslation } from 'react-i18next'
import { useDeleteBooking, useEndBooking } from '../../hooks/useBookings'
import { useGpsTracking } from '../../hooks/useGpsTracking'
import { Button } from '../ui/button'
import { Input } from '../ui/input'
import { Label } from '../ui/label'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '../ui/dialog'
import type { BookingResponse } from '../../types/api'
import { formatDate, formatTime } from '../../utils/dateUtils'
import { useSettingsStore } from '../../stores/settingsStore'

const endSchema = z
  .object({
    startOdo: z.number().nonnegative(),
    endOdo: z.number().nonnegative(),
  })
  .refine((d) => d.endOdo >= d.startOdo, {
    message: 'odo_error',
    path: ['endOdo'],
  })

type EndFormValues = z.infer<typeof endSchema>

interface BookingStartEndDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  booking: BookingResponse
  hasDistance: boolean
  distanceKm?: number
  onEdit?: () => void
}

export default function BookingStartEndDialog({
  open,
  onOpenChange,
  booking,
  hasDistance,
  distanceKm,
  onEdit,
}: BookingStartEndDialogProps) {
  const { t } = useTranslation('bookings')
  const language = useSettingsStore((s) => s.language)
  const deleteBooking = useDeleteBooking()
  const endBooking = useEndBooking()
  const { isTracking, distanceMeters, currentPosition, error: gpsError, startTracking, stopTracking } =
    useGpsTracking()
  const [confirmCancel, setConfirmCancel] = useState(false)

  useEffect(() => {
    if (!open) setConfirmCancel(false)
  }, [open])

  const now = new Date()
  const startDate = new Date(booking.start_date)
  const isFuture = startDate > now
  const isStarted = startDate <= now && !hasDistance
  const canTrack = !hasDistance

  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<EndFormValues>({
    resolver: zodResolver(endSchema),
  })

  async function handleDelete() {
    await deleteBooking.mutateAsync(booking.booking_reference)
    onOpenChange(false)
  }

  async function handleStopGps() {
    stopTracking()
    await endBooking.mutateAsync({
      id: booking.booking_reference,
      body: {
        start_distance: 0,
        end_distance: Math.round(distanceMeters),
        position: currentPosition ?? undefined,
      },
    })
    onOpenChange(false)
  }

  async function handleEnd(data: EndFormValues) {
    await endBooking.mutateAsync({
      id: booking.booking_reference,
      body: {
        start_distance: data.startOdo * 1000,
        end_distance: data.endOdo * 1000,
      },
    })
    onOpenChange(false)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>
            {hasDistance ? t('status_completed') : isFuture ? t('status_upcoming') : t('end_booking_title')}
          </DialogTitle>
        </DialogHeader>

        <div className="space-y-2 text-sm">
          <p>
            <span className="font-medium">{t('form_start')}: </span>
            {formatDate(booking.start_date, language)} {formatTime(booking.start_date, language)}
          </p>
          <p>
            <span className="font-medium">{t('form_end')}: </span>
            {formatDate(booking.end_date, language)} {formatTime(booking.end_date, language)}
          </p>
          {hasDistance && distanceKm !== undefined && (
            <p>
              <span className="font-medium">{t('col_distance')}: </span>
              {distanceKm.toFixed(1)} km
            </p>
          )}
        </div>

        {canTrack && (
          <div className="space-y-4">
            {gpsError && (
              <p className="text-sm text-destructive">{gpsError}</p>
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
                  onClick={handleStopGps}
                  disabled={endBooking.isPending}
                >
                  {endBooking.isPending ? '...' : t('stop_tracking')}
                </Button>
              </div>
            ) : (
              <>
                <Button
                  className="w-full min-h-[52px] text-base"
                  onClick={startTracking}
                >
                  {t('start_tracking')}
                </Button>

                {isFuture && !confirmCancel && (
                  <DialogFooter className="flex gap-2">
                    {onEdit && (
                      <Button
                        variant="outline"
                        onClick={() => { onEdit(); onOpenChange(false) }}
                        className="min-h-[44px]"
                      >
                        {t('edit_booking')}
                      </Button>
                    )}
                    <Button
                      variant="destructive"
                      onClick={() => setConfirmCancel(true)}
                      className="min-h-[44px]"
                    >
                      {t('cancel_booking')}
                    </Button>
                  </DialogFooter>
                )}

                {isFuture && confirmCancel && (
                  <DialogFooter className="flex-col gap-2">
                    <p className="text-sm text-muted-foreground">{t('cancel_booking_confirm')}</p>
                    <div className="flex gap-2">
                      <Button variant="outline" onClick={() => setConfirmCancel(false)} className="min-h-[44px]">
                        {t('form_cancel')}
                      </Button>
                      <Button
                        variant="destructive"
                        onClick={handleDelete}
                        disabled={deleteBooking.isPending}
                        className="min-h-[44px]"
                      >
                        {deleteBooking.isPending ? '...' : t('cancel_booking')}
                      </Button>
                    </div>
                  </DialogFooter>
                )}

                {isStarted && (
                  <>
                    <div className="relative">
                      <div className="absolute inset-0 flex items-center">
                        <span className="w-full border-t" />
                      </div>
                      <div className="relative flex justify-center text-xs uppercase">
                        <span className="bg-background px-2 text-muted-foreground">or</span>
                      </div>
                    </div>

                    <form onSubmit={handleSubmit(handleEnd)} className="space-y-4" id="end-booking-form">
                      <div className="space-y-2">
                        <Label htmlFor="start-odo">{t('end_booking_start_odo')}</Label>
                        <Input
                          id="start-odo"
                          type="number"
                          min={0}
                          step={1}
                          {...register('startOdo', { valueAsNumber: true })}
                          aria-describedby={errors.startOdo ? 'start-odo-error' : undefined}
                        />
                        {errors.startOdo && (
                          <p id="start-odo-error" className="text-sm text-destructive" role="alert">
                            {errors.startOdo.message}
                          </p>
                        )}
                      </div>

                      <div className="space-y-2">
                        <Label htmlFor="end-odo">{t('end_booking_end_odo')}</Label>
                        <Input
                          id="end-odo"
                          type="number"
                          min={0}
                          step={1}
                          {...register('endOdo', { valueAsNumber: true })}
                          aria-describedby={errors.endOdo ? 'end-odo-error' : undefined}
                        />
                        {errors.endOdo && (
                          <p id="end-odo-error" className="text-sm text-destructive" role="alert">
                            {t('end_booking_error_odo')}
                          </p>
                        )}
                      </div>

                      <DialogFooter>
                        <Button type="submit" form="end-booking-form" disabled={isSubmitting} className="min-h-[44px]">
                          {isSubmitting ? '...' : t('end_booking_submit')}
                        </Button>
                      </DialogFooter>
                    </form>
                  </>
                )}
              </>
            )}
          </div>
        )}

      </DialogContent>
    </Dialog>
  )
}
