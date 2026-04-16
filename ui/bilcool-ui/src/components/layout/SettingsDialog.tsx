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
import ColorSelector from './ColorSelector'
import ThemeToggle from './ThemeToggle'

export default function SettingsDialog() {
  const { t } = useTranslation('common')

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
          <div className="flex items-center justify-between">
            <span className="text-sm font-medium">{t('color.label')}</span>
            <ColorSelector />
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}
