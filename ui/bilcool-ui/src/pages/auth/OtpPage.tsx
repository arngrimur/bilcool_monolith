import { useState } from 'react'
import { useNavigate, useLocation, Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { startRegistration } from '@simplewebauthn/browser'
import { verifyToken, completeLogin, getUser } from '../../api/auth'
import { useAuthStore, storeAuthToken, decodeUserRefFromToken } from '../../stores/authStore'
import { Button } from '../../components/ui/button'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'

export default function OtpPage() {
  const { t } = useTranslation('common')
  const navigate = useNavigate()
  const location = useLocation()
  const setAuth = useAuthStore((s) => s.setAuth)

  const email = (location.state as { email?: string } | null)?.email

  const [otp, setOtp] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  if (!email) {
    navigate('/login', { replace: true })
    return null
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError(null)
    setLoading(true)

    try {
      const { session_id, options } = await verifyToken({ email: email!, token: otp })
      const { publicKey } = options as { publicKey: Parameters<typeof startRegistration>[0]['optionsJSON'] }
      const credential = await startRegistration({ optionsJSON: publicKey })
      const { token } = await completeLogin({ session_id, credential })
      storeAuthToken(token)
      const userRef = decodeUserRefFromToken(token)
      if (!userRef) throw new Error('Invalid token')
      const user = await getUser(userRef)
      setAuth({ token, userRef: user.user_ref, username: user.username, email: user.email, role: user.role })
      navigate('/')
    } catch (err: unknown) {
      if (err instanceof Error && err.name === 'InvalidStateError') {
        setError(t('auth.error_passkey_exists'))
      } else if (err instanceof Error && (err.name === 'NotAllowedError' || err.name === 'AbortError')) {
        setError(t('auth.error_passkey_cancelled'))
      } else {
        const apiErr = err as { status?: number }
        if (apiErr.status === 401 || apiErr.status === 400) {
          setError(t('auth.error_invalid_token'))
        } else {
          setError(t('auth.error_generic'))
        }
      }
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="flex min-h-svh items-center justify-center p-4 bg-background">
      <div className="w-full max-w-sm space-y-6">
        <div className="text-center space-y-2">
          <h1 className="text-2xl font-bold tracking-tight">{t('auth.otp_title')}</h1>
          <p className="text-sm text-muted-foreground">
            {t('auth.otp_description', { email })}
          </p>
        </div>

        <form onSubmit={handleSubmit} className="space-y-4" noValidate>
          <div className="space-y-2">
            <Label htmlFor="otp">{t('auth.otp_label')}</Label>
            <Input
              id="otp"
              type="text"
              inputMode="numeric"
              autoComplete="one-time-code"
              maxLength={6}
              value={otp}
              onChange={(e) => setOtp(e.target.value.replace(/\D/g, ''))}
              required
              disabled={loading}
              aria-label={t('auth.otp_label')}
              aria-describedby={error ? 'otp-error' : undefined}
              className="text-center text-lg tracking-widest"
            />
          </div>

          {error && (
            <p id="otp-error" role="alert" className="text-sm text-destructive">
              {error}
            </p>
          )}

          <Button type="submit" className="w-full min-h-[44px]" disabled={loading || otp.length < 6}>
            {loading ? '...' : t('auth.verify')}
          </Button>
        </form>

        <p className="text-center text-sm">
          <Link to="/login" className="underline underline-offset-4 hover:text-primary">
            {t('auth.back_to_login')}
          </Link>
        </p>
      </div>
    </div>
  )
}
