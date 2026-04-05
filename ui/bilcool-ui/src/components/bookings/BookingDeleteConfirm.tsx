import { useTranslation } from 'react-i18next'
import { useDeleteBooking } from '../../hooks/useBookings'
import { Button } from '../ui/button'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '../ui/dialog'

interface BookingDeleteConfirmProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  bookingReference: string
}

export default function BookingDeleteConfirm({
  open,
  onOpenChange,
  bookingReference,
}: BookingDeleteConfirmProps) {
  const { t } = useTranslation('bookings')
  const deleteBooking = useDeleteBooking()

  async function handleConfirm() {
    await deleteBooking.mutateAsync(bookingReference)
    onOpenChange(false)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t('cancel_booking')}</DialogTitle>
        </DialogHeader>
        <p className="text-sm text-muted-foreground">{t('cancel_booking_confirm')}</p>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)} className="min-h-[44px]">
            {t('form_cancel')}
          </Button>
          <Button
            variant="destructive"
            onClick={handleConfirm}
            disabled={deleteBooking.isPending}
            className="min-h-[44px]"
          >
            {deleteBooking.isPending ? '...' : t('cancel_booking')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
