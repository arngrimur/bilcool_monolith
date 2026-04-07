import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { startAuthentication } from '@simplewebauthn/browser'
import { beginLogin, completeLogin, getUser } from '../../api/auth'
import { useAuthStore, storeAuthToken, decodeUserRefFromToken } from '../../stores/authStore'
import { Button } from '../../components/ui/button'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'

export default function LoginPage() {
  const { t } = useTranslation('common')
  const navigate = useNavigate()
  const setAuth = useAuthStore((s) => s.setAuth)

  const [email, setEmail] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError(null)
    setLoading(true)

    try {
      const result = await beginLogin({ email })

      if (result.next_step === 'verify_token') {
        navigate('/login/otp', { state: { email } })
        return
      }

      if (result.next_step === 'passkey_assertion') {
        const { publicKey } = result.options as { publicKey: Parameters<typeof startAuthentication>[0]['optionsJSON'] }
        const credential = await startAuthentication({ optionsJSON: publicKey })
        const { token } = await completeLogin({ session_id: result.session_id!, credential })
        await finalizeLogin(token)
      }
    } catch (err: unknown) {
      const apiErr = err as { status?: number }
      if (apiErr.status === 404) {
        setError(t('auth.error_user_not_found'))
      } else if (apiErr.status === 401) {
        setError(t('auth.error_passkey_failed'))
      } else {
        setError(t('auth.error_generic'))
      }
    } finally {
      setLoading(false)
    }
  }

  async function finalizeLogin(token: string) {
    storeAuthToken(token)
    const userRef = decodeUserRefFromToken(token)
    if (!userRef) throw new Error('Invalid token')
    const user = await getUser(userRef)
    setAuth({ token, userRef: user.user_ref, username: user.username, email: user.email, role: user.role })
    navigate('/')
  }

  return (
    <div className="flex min-h-svh items-center justify-center p-4 bg-background">
      <div className="w-full max-w-sm space-y-6">
        <div className="text-center">
          <h1 className="text-3xl font-bold tracking-tight">{t('app_name')}</h1>
        </div>

        <form onSubmit={handleSubmit} className="space-y-4" noValidate>
          <div className="space-y-2">
            <Label htmlFor="email">{t('auth.email_label')}</Label>
            <Input
              id="email"
              type="email"
              autoComplete="email"
              placeholder={t('auth.email_placeholder')}
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              required
              disabled={loading}
              aria-describedby={error ? 'login-error' : undefined}
            />
          </div>

          {error && (
            <p id="login-error" role="alert" className="text-sm text-destructive">
              {error}
            </p>
          )}

          <Button type="submit" className="w-full min-h-[44px]" disabled={loading || !email}>
            {loading ? '...' : t('auth.sign_in')}
          </Button>
        </form>
      </div>
    </div>
  )
}
