import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'

export default function NotFoundPage() {
  const { t } = useTranslation('common')

  return (
    <div className="flex min-h-svh items-center justify-center p-4">
      <div className="text-center space-y-4">
        <h1 className="text-4xl font-bold">404</h1>
        <p className="text-muted-foreground">{t('errors.unexpected')}</p>
        <Link to="/" className="underline underline-offset-4 hover:text-primary text-sm">
          {t('nav.calendar')}
        </Link>
      </div>
    </div>
  )
}
