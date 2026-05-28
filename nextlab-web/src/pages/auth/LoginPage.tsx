import { useState } from 'react'
import { useNavigate, useLocation, Navigate } from 'react-router-dom'
import { useAuth } from '../../context/AuthContext'
import { ErrorMessage } from '../../components/common/ErrorMessage'
import { ApiError } from '../../types'

function roleHome(role: string): string {
  switch (role) {
    case 'docente': return '/docente/dashboard'
    case 'co_docente': return '/codocente/dashboard'
    case 'admin': return '/admin/dashboard'
    default: return '/student/dashboard'
  }
}

const TEST_ACCOUNTS = [
  { label: 'Admin', email: 'admin@nextlab.dev', password: 'test1234' },
  { label: 'Docente', email: 'docente@nextlab.dev', password: 'test1234' },
  { label: 'Co-docente', email: 'codocente@nextlab.dev', password: 'test1234' },
  { label: 'Estudiante', email: 'estudiante@nextlab.dev', password: 'test1234' },
]

export function LoginPage() {
  const { user, login } = useAuth()
  const navigate = useNavigate()
  const location = useLocation()

  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [submitting, setSubmitting] = useState(false)

  const sessionExpired =
    location.state?.sessionExpired === true ||
    new URLSearchParams(location.search).get('session_expired') === '1'

  if (user) {
    return <Navigate to={roleHome(user.role)} replace />
  }

  async function doLogin(emailVal: string, passwordVal: string) {
    setError('')
    setSubmitting(true)
    try {
      await login(emailVal, passwordVal)
      const from = (location.state as { from?: { pathname: string } })?.from?.pathname
      navigate(from ?? roleHome(''), { replace: true })
    } catch (err) {
      if (err instanceof ApiError) {
        if (err.code === 'rate_limit_exceeded') {
          setError(`Too many attempts. Try again in ${err.retryAfter} seconds.`)
        } else if (err.code === 'account_inactive') {
          setError('This account has been deactivated.')
        } else if (err.code === 'network_error') {
          setError('Unable to reach the server. Check your connection.')
        } else {
          setError(err.message || 'Invalid email or password.')
        }
      } else {
        setError('An unexpected error occurred.')
      }
    } finally {
      setSubmitting(false)
    }
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!email || !password) {
      setError('Please enter your email and password.')
      return
    }
    if (!email.includes('@')) {
      setError('Please enter a valid email address.')
      return
    }
    await doLogin(email, password)
  }

  return (
    <div className="min-h-screen bg-[#111111] flex items-center justify-center px-4">
      <div className="w-full max-w-sm">
        <h1 className="text-2xl font-semibold text-[#E85D24] mb-8 text-center">NextLab</h1>

        {sessionExpired && (
          <div className="mb-4 rounded bg-[#1a1a1a] border border-[#2e2e2e] px-4 py-3 text-sm text-[#888888]">
            Your session has ended. Please sign in again.
          </div>
        )}

        <form onSubmit={handleSubmit} noValidate className="space-y-4">
          <div>
            <label htmlFor="email" className="block text-sm text-[#888888] mb-1">
              Email
            </label>
            <input
              id="email"
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className="w-full rounded bg-[#1a1a1a] border border-[#2e2e2e] px-3 py-2 text-[#f0f0f0] text-sm focus-visible:outline-none focus-visible:border-[#E85D24]"
              autoComplete="email"
              disabled={submitting}
            />
          </div>

          <div>
            <label htmlFor="password" className="block text-sm text-[#888888] mb-1">
              Password
            </label>
            <input
              id="password"
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="w-full rounded bg-[#1a1a1a] border border-[#2e2e2e] px-3 py-2 text-[#f0f0f0] text-sm focus-visible:outline-none focus-visible:border-[#E85D24]"
              autoComplete="current-password"
              disabled={submitting}
            />
          </div>

          {error && <ErrorMessage message={error} onDismiss={() => setError('')} />}

          <button
            type="submit"
            disabled={submitting}
            className="w-full rounded bg-[#E85D24] py-2 text-sm font-semibold text-white hover:opacity-90 disabled:opacity-50 flex items-center justify-center gap-2"
          >
            {submitting && (
              <span className="h-4 w-4 rounded-full border-2 border-white border-t-transparent animate-spin" />
            )}
            {submitting ? 'Signing in…' : 'Sign in'}
          </button>
        </form>

        {/* Dev quick-login panel */}
        <div className="mt-8 border border-[#2e2e2e] rounded-lg p-4">
          <p className="text-xs text-[#888888] mb-3 uppercase tracking-wide">Test accounts</p>
          <div className="grid grid-cols-2 gap-2">
            {TEST_ACCOUNTS.map((acc) => (
              <button
                key={acc.label}
                type="button"
                disabled={submitting}
                onClick={() => {
                  setEmail(acc.email)
                  setPassword(acc.password)
                  doLogin(acc.email, acc.password)
                }}
                className="rounded border border-[#2e2e2e] bg-[#1a1a1a] px-3 py-2 text-xs text-[#f0f0f0] hover:border-[#E85D24] hover:text-[#E85D24] transition-colors disabled:opacity-50"
              >
                {acc.label}
              </button>
            ))}
          </div>
          <p className="mt-3 text-xs text-[#888888]">
            Password for all accounts: <span className="font-mono text-[#f0f0f0]">test1234</span>
          </p>
        </div>
      </div>
    </div>
  )
}
