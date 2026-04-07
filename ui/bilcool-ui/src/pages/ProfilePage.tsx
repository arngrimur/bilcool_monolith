import { useTranslation } from 'react-i18next'
import { useAuthStore } from '../stores/authStore'
import { Badge } from '../components/ui/badge'

export default function ProfilePage() {
  const { t } = useTranslation('common')
  const { username, email, role } = useAuthStore()

  return (
    <div className="max-w-md space-y-6">
      <h1 className="text-2xl font-bold">{t('profile.title')}</h1>

      <div className="rounded-lg border bg-card p-6 space-y-4">
        <div className="space-y-1">
          <p className="text-sm font-medium text-muted-foreground">{t('profile.username')}</p>
          <p className="text-base">{username}</p>
        </div>

        <div className="space-y-1">
          <p className="text-sm font-medium text-muted-foreground">{t('profile.email')}</p>
          <p className="text-base">{email}</p>
        </div>

        <div className="space-y-1">
          <p className="text-sm font-medium text-muted-foreground">{t('profile.role')}</p>
          <Badge variant={role === 'admin' ? 'default' : 'secondary'}>
            {role ?? '—'}
          </Badge>
        </div>

        <div className="space-y-1">
          <p className="text-sm font-medium text-muted-foreground">{t('profile.passkeys')}</p>
          <p className="text-base">{t('profile.passkeys_registered_one', { count: 1 })}</p>
        </div>
      </div>
    </div>
  )
}
