import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { useTranslation } from 'react-i18next'
import { useAuthStore } from '../../stores/authStore'
import { useUpsertBooking } from '../../hooks/useBookings'
import { snapToQuarterHour, hasOverlap } from '../../utils/bookingUtils'
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

const formSchema = z
  .object({
    start: z.string().min(1),
    end: z.string().min(1),
  })
  .refine((d) => new Date(d.end) > new Date(d.start), {
    message: 'end_before_start',
    path: ['end'],
  })
  .refine(
    (d) => new Date(d.end).getTime() - new Date(d.start).getTime() >= 15 * 60 * 1000,
    { message: 'min_duration', path: ['end'] }
  )

type FormValues = z.infer<typeof formSchema>

function toLocalDatetimeString(date: Date) {
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`
}

interface BookingFormProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  initialStart?: Date
  initialEnd?: Date
  editingBooking?: BookingResponse
  allBookings: BookingResponse[]
}

export default function BookingForm({
  open,
  onOpenChange,
  initialStart,
  initialEnd,
  editingBooking,
  allBookings,
}: BookingFormProps) {
  const { t } = useTranslation('bookings')
  const userRef = useAuthStore((s) => s.userRef)
  const upsert = useUpsertBooking()

  const defaultStart = initialStart ? toLocalDatetimeString(snapToQuarterHour(initialStart)) : ''
  const defaultEnd = initialEnd ? toLocalDatetimeString(snapToQuarterHour(initialEnd)) : ''

  const {
    register,
    handleSubmit,
    setError,
    formState: { errors, isSubmitting },
  } = useForm<FormValues>({
    resolver: zodResolver(formSchema),
    defaultValues: { start: defaultStart, end: defaultEnd },
  })

  async function onSubmit(data: FormValues) {
    const start = snapToQuarterHour(new Date(data.start))
    const end = snapToQuarterHour(new Date(data.end))

    if (hasOverlap(start, end, allBookings, editingBooking?.booking_reference)) {
      setError('end', { message: 'overlap' })
      return
    }

    const bookingRef = editingBooking?.booking_reference ?? crypto.randomUUID()

    await upsert.mutateAsync({
      user_ref: userRef!,
      booking_reference: bookingRef,
      start_date: start.toISOString(),
      end_date: end.toISOString(),
    })

    onOpenChange(false)
  }

  const title = editingBooking ? t('form_title_edit') : t('form_title_new')

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
        </DialogHeader>

        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4" id="booking-form">
          <div className="space-y-2">
            <Label htmlFor="booking-start">{t('form_start')}</Label>
            <Input
              id="booking-start"
              type="datetime-local"
              step={900}
              {...register('start')}
              aria-describedby={errors.start ? 'start-error' : undefined}
            />
            {errors.start && (
              <p id="start-error" className="text-sm text-destructive" role="alert">
                {errors.start.message}
              </p>
            )}
          </div>

          <div className="space-y-2">
            <Label htmlFor="booking-end">{t('form_end')}</Label>
            <Input
              id="booking-end"
              type="datetime-local"
              step={900}
              {...register('end')}
              aria-describedby={errors.end ? 'end-error' : undefined}
            />
            {errors.end && (
              <p id="end-error" className="text-sm text-destructive" role="alert">
                {errors.end.message === 'overlap'
                  ? t('form_error_overlap')
                  : errors.end.message === 'min_duration'
                    ? t('form_error_min_duration')
                    : t('form_error_end_before_start')}
              </p>
            )}
          </div>

          {upsert.error && (
            <p className="text-sm text-destructive" role="alert">
              {(upsert.error as { body?: { message?: string } }).body?.message ?? t('form_cancel')}
            </p>
          )}
        </form>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)} className="min-h-[44px]">
            {t('form_cancel')}
          </Button>
          <Button type="submit" form="booking-form" disabled={isSubmitting} className="min-h-[44px]">
            {isSubmitting ? '...' : t('form_save')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
