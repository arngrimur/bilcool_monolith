import { NavLink } from 'react-router-dom'
import { CalendarDays, ClipboardList, User, Users, LogOut } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useAuthStore } from '../../stores/authStore'
import { useAuth } from '../../hooks/useAuth'
import { cn } from '../../lib/utils'
import ThemeToggle from './ThemeToggle'
import LanguageToggle from './LanguageToggle'

interface SidebarProps {
  onNavigate: () => void
}

const navLinkClass = ({ isActive }: { isActive: boolean }) =>
  cn(
    'flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-colors min-h-[44px]',
    isActive
      ? 'bg-accent text-accent-foreground'
      : 'text-muted-foreground hover:bg-accent hover:text-accent-foreground'
  )

export default function Sidebar({ onNavigate }: SidebarProps) {
  const { t } = useTranslation('common')
  const role = useAuthStore((s) => s.role)
  const { logout } = useAuth()

  return (
    <div className="flex h-full flex-col border-r bg-background">
      <div className="flex h-14 items-center border-b px-4">
        <span className="text-lg font-semibold">{t('app_name')}</span>
      </div>

      <nav className="flex-1 space-y-1 p-3" aria-label="Main navigation">
        <NavLink to="/" end className={navLinkClass} onClick={onNavigate}>
          <CalendarDays className="h-4 w-4" aria-hidden="true" />
          {t('nav.calendar')}
        </NavLink>
        <NavLink to="/bookings" className={navLinkClass} onClick={onNavigate}>
          <ClipboardList className="h-4 w-4" aria-hidden="true" />
          {t('nav.bookings')}
        </NavLink>
        <NavLink to="/profile" className={navLinkClass} onClick={onNavigate}>
          <User className="h-4 w-4" aria-hidden="true" />
          {t('nav.profile')}
        </NavLink>
        {role === 'admin' && (
          <NavLink to="/admin/users" className={navLinkClass} onClick={onNavigate}>
            <Users className="h-4 w-4" aria-hidden="true" />
            {t('nav.admin_users')}
          </NavLink>
        )}
      </nav>

      <div className="border-t p-3 space-y-2">
        <div className="flex items-center gap-2">
          <ThemeToggle />
          <LanguageToggle />
        </div>
        <button
          onClick={logout}
          className="flex w-full items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium text-muted-foreground transition-colors hover:bg-accent hover:text-accent-foreground min-h-[44px]"
          aria-label={t('nav.sign_out')}
        >
          <LogOut className="h-4 w-4" aria-hidden="true" />
          {t('nav.sign_out')}
        </button>
      </div>
    </div>
  )
}
