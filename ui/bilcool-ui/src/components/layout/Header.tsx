import { Menu, LogOut } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Button } from '../ui/button'
import { useAuth } from '../../hooks/useAuth'
import SettingsDialog from './SettingsDialog'
import LanguageToggle from './LanguageToggle'

interface HeaderProps {
  onMenuClick: () => void
}

export default function Header({ onMenuClick }: HeaderProps) {
  const { t } = useTranslation('common')
  const { logout } = useAuth()

  return (
    <header className="flex h-14 items-center border-b bg-background px-4 gap-2">
      <Button
        variant="ghost"
        size="icon"
        onClick={onMenuClick}
        aria-label="Open navigation menu"
        className="min-h-[44px] min-w-[44px] md:hidden"
      >
        <Menu className="h-5 w-5" />
      </Button>
      <span className="flex-1 text-lg font-semibold md:hidden">{t('app_name')}</span>
      <div className="flex-1 hidden md:block" />
      <SettingsDialog />
      <LanguageToggle />
      <Button
        variant="ghost"
        size="icon"
        onClick={logout}
        className="min-h-[44px] min-w-[44px]"
        aria-label={t('nav.sign_out')}
      >
        <LogOut className="h-5 w-5" />
      </Button>
    </header>
  )
}
