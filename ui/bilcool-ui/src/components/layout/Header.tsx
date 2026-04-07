import { Menu } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Button } from '../ui/button'
import ThemeToggle from './ThemeToggle'
import LanguageToggle from './LanguageToggle'

interface HeaderProps {
  onMenuClick: () => void
}

export default function Header({ onMenuClick }: HeaderProps) {
  const { t } = useTranslation('common')

  return (
    <header className="flex h-14 items-center border-b bg-background px-4 gap-2 md:hidden">
      <Button
        variant="ghost"
        size="icon"
        onClick={onMenuClick}
        aria-label="Open navigation menu"
        className="min-h-[44px] min-w-[44px]"
      >
        <Menu className="h-5 w-5" />
      </Button>
      <span className="flex-1 text-lg font-semibold">{t('app_name')}</span>
      <ThemeToggle />
      <LanguageToggle />
    </header>
  )
}
