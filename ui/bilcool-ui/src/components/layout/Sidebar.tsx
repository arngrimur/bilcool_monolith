import { NavLink } from 'react-router-dom'
import { CalendarDays, ClipboardList, User, Users, ChevronLeft, ChevronRight } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useAuthStore } from '../../stores/authStore'
import { cn } from '../../lib/utils'

interface SidebarProps {
  onNavigate: () => void
  collapsed?: boolean
  onToggleCollapse?: () => void
}

export default function Sidebar({ onNavigate, collapsed = false, onToggleCollapse }: SidebarProps) {
  const { t } = useTranslation('common')
  const role = useAuthStore((s) => s.role)

  const navLinkClass = ({ isActive }: { isActive: boolean }) =>
    cn(
      'flex items-center rounded-lg px-3 py-2 text-sm font-medium transition-colors min-h-[44px]',
      collapsed ? 'justify-center gap-0' : 'gap-3',
      isActive
        ? 'bg-accent text-accent-foreground'
        : 'text-muted-foreground hover:bg-accent hover:text-accent-foreground'
    )

  return (
    <div className="flex h-full w-full flex-col border-r bg-background overflow-hidden">
      <div className="flex h-14 items-center border-b px-3 gap-2">
        {!collapsed && <span className="text-lg font-semibold flex-1 truncate">{t('app_name')}</span>}
        {onToggleCollapse && (
          <button
            onClick={onToggleCollapse}
            className={cn(
              'flex items-center justify-center rounded-lg p-1.5 text-muted-foreground hover:bg-accent hover:text-accent-foreground transition-colors',
              collapsed && 'w-full'
            )}
            aria-label={collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
          >
            {collapsed ? <ChevronRight className="h-4 w-4" /> : <ChevronLeft className="h-4 w-4" />}
          </button>
        )}
      </div>

      <nav className="flex-1 space-y-1 p-2" aria-label="Main navigation">
        <NavLink to="/" end className={navLinkClass} onClick={onNavigate} title={collapsed ? t('nav.calendar') : undefined}>
          <CalendarDays className="h-4 w-4 shrink-0" aria-hidden="true" />
          {!collapsed && t('nav.calendar')}
        </NavLink>
        <NavLink to="/bookings" className={navLinkClass} onClick={onNavigate} title={collapsed ? t('nav.bookings') : undefined}>
          <ClipboardList className="h-4 w-4 shrink-0" aria-hidden="true" />
          {!collapsed && t('nav.bookings')}
        </NavLink>
        <NavLink to="/profile" className={navLinkClass} onClick={onNavigate} title={collapsed ? t('nav.profile') : undefined}>
          <User className="h-4 w-4 shrink-0" aria-hidden="true" />
          {!collapsed && t('nav.profile')}
        </NavLink>
        {role === 'admin' && (
          <NavLink to="/admin/users" className={navLinkClass} onClick={onNavigate} title={collapsed ? t('nav.admin_users') : undefined}>
            <Users className="h-4 w-4 shrink-0" aria-hidden="true" />
            {!collapsed && t('nav.admin_users')}
          </NavLink>
        )}
      </nav>
    </div>
  )
}
