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

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError('')

    if (!email || !password) {
      setError('Please enter your email and password.')
      return
    }
    if (!email.includes('@')) {
      setError('Please enter a valid email address.')
      return
    }

    setSubmitting(true)
    try {
      await login(email, password)
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

        <div className="mt-6">
          <a
            href={`${import.meta.env.VITE_API_URL}/auth/google`}
            className="flex w-full items-center justify-center gap-2 rounded border border-[#2e2e2e] bg-[#1a1a1a] py-2 text-sm text-[#f0f0f0] hover:border-[#4285F4] hover:text-[#4285F4]"
          >
            <svg width="16" height="16" viewBox="0 0 24 24" aria-hidden="true">
              <path
                fill="#4285F4"
                d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z"
              />
              <path
                fill="#34A853"
                d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"
              />
              <path
                fill="#FBBC05"
                d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l3.66-2.84z"
              />
              <path
                fill="#EA4335"
                d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z"
              />
            </svg>
            Sign in with Google
          </a>
          <p className="mt-2 text-center text-xs text-[#888888]">
            Google sign-in is available for students
          </p>
        </div>
      </div>
    </div>
  )
}
