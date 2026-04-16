import { Settings } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Button } from '../ui/button'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '../ui/dialog'
import ThemeToggle from './ThemeToggle'
import { useSettingsStore } from '../../stores/settingsStore'
import { BOOKING_COLORS } from '../../utils/bookingColors'

export default function SettingsDialog() {
  const { t } = useTranslation('common')
  const bookingColor = useSettingsStore((s) => s.bookingColor)
  const setBookingColor = useSettingsStore((s) => s.setBookingColor)

  return (
    <Dialog>
      <DialogTrigger asChild>
        <Button
          variant="ghost"
          size="icon"
          className="min-h-[44px] min-w-[44px]"
          aria-label={t('nav.settings')}
        >
          <Settings className="h-5 w-5" />
        </Button>
      </DialogTrigger>
      <DialogContent className="sm:max-w-xs">
        <DialogHeader>
          <DialogTitle>{t('nav.settings')}</DialogTitle>
        </DialogHeader>
        <div className="flex flex-col gap-4 py-2">
          <div className="flex items-center justify-between">
            <span className="text-sm font-medium">{t('settings.theme')}</span>
            <ThemeToggle />
          </div>
          <div className="flex flex-col gap-2">
            <span className="text-sm font-medium">{t('settings.bookingColor')}</span>
            <div className="flex gap-2 flex-wrap">
              {BOOKING_COLORS.map((color) => (
                <button
                  key={color}
                  onClick={() => setBookingColor(color)}
                  className="w-8 h-8 rounded-full transition-all"
                  style={{
                    backgroundColor: color,
                    outline: bookingColor === color ? `2px solid ${color}` : 'none',
                    outlineOffset: '2px',
                  }}
                  aria-label={color}
                />
              ))}
            </div>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}
