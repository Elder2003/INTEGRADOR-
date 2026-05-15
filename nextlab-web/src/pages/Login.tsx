import { useState, type FormEvent } from 'react'
import { useNavigate } from 'react-router-dom'
import { apiLogin } from '../api/auth'

type Mode = 'idle' | 'loading' | 'error'

export default function Login() {
  const navigate = useNavigate()
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [mode, setMode] = useState<Mode>('idle')
  const [errorMsg, setErrorMsg] = useState('')

  const isLoading = mode === 'loading'

  async function handleSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault()
    setMode('loading')
    setErrorMsg('')
    try {
      const result = await apiLogin({ username, password })
      localStorage.setItem('access_token', result.access_token)
      navigate('/chat')
    } catch (err) {
      setMode('error')
      setErrorMsg(err instanceof Error ? err.message : 'Something went wrong')
    }
  }

  return (
    <div style={{ width: '100%', maxWidth: 400, padding: '0 16px', margin: '0 auto' }}>
      <form
        onSubmit={handleSubmit}
        noValidate
        style={{
          background: 'var(--card-bg)',
          border: '1px solid var(--border)',
          borderRadius: 12,
          padding: '40px 36px',
        }}
      >
        {/* Logo mark */}
        <div style={{ textAlign: 'center', marginBottom: 28 }}>
          <div
            style={{
              display: 'inline-block',
              width: 36,
              height: 36,
              borderRadius: 8,
              background: 'var(--accent)',
              marginBottom: 16,
            }}
          />
          <h1
            style={{
              color: 'var(--text-h)',
              fontSize: 22,
              fontWeight: 700,
              letterSpacing: '-0.02em',
              margin: 0,
            }}
          >
            Sign in to Nextlab
          </h1>
          <p style={{ color: 'var(--text)', fontSize: 14, marginTop: 6 }}>
            Enter your credentials below
          </p>
        </div>

        {/* Error banner */}
        {mode === 'error' && errorMsg && (
          <div
            role="alert"
            style={{
              background: 'rgba(239, 68, 68, 0.1)',
              border: '1px solid rgba(239, 68, 68, 0.3)',
              borderRadius: 8,
              padding: '10px 14px',
              marginBottom: 20,
              color: 'var(--error)',
              fontSize: 14,
            }}
          >
            {errorMsg}
          </div>
        )}

        {/* Username */}
        <div style={{ marginBottom: 16 }}>
          <label
            htmlFor="username"
            style={{
              display: 'block',
              color: 'var(--text-h)',
              fontSize: 13,
              fontWeight: 500,
              marginBottom: 6,
            }}
          >
            Username
          </label>
          <input
            id="username"
            type="text"
            autoComplete="username"
            required
            value={username}
            onChange={e => setUsername(e.target.value)}
            disabled={isLoading}
            placeholder="elder"
            style={{
              width: '100%',
              padding: '10px 12px',
              background: 'var(--input-bg)',
              border: '1px solid var(--input-border)',
              borderRadius: 8,
              color: 'var(--text-h)',
              fontSize: 14,
              outline: 'none',
              transition: 'border-color 0.15s',
            }}
            onFocus={e => (e.currentTarget.style.borderColor = 'var(--input-focus)')}
            onBlur={e => (e.currentTarget.style.borderColor = 'var(--input-border)')}
          />
        </div>

        {/* Password */}
        <div style={{ marginBottom: 24 }}>
          <label
            htmlFor="password"
            style={{
              display: 'block',
              color: 'var(--text-h)',
              fontSize: 13,
              fontWeight: 500,
              marginBottom: 6,
            }}
          >
            Password
          </label>
          <input
            id="password"
            type="password"
            autoComplete="current-password"
            required
            value={password}
            onChange={e => setPassword(e.target.value)}
            disabled={isLoading}
            placeholder="••••••••"
            style={{
              width: '100%',
              padding: '10px 12px',
              background: 'var(--input-bg)',
              border: '1px solid var(--input-border)',
              borderRadius: 8,
              color: 'var(--text-h)',
              fontSize: 14,
              outline: 'none',
              transition: 'border-color 0.15s',
            }}
            onFocus={e => (e.currentTarget.style.borderColor = 'var(--input-focus)')}
            onBlur={e => (e.currentTarget.style.borderColor = 'var(--input-border)')}
          />
        </div>

        {/* Submit */}
        <button
          type="submit"
          disabled={isLoading}
          style={{
            width: '100%',
            padding: '11px 0',
            background: isLoading ? '#7c3e14' : 'var(--accent)',
            color: 'var(--accent-text)',
            border: 'none',
            borderRadius: 8,
            fontSize: 15,
            fontWeight: 600,
            cursor: isLoading ? 'not-allowed' : 'pointer',
            transition: 'background 0.15s',
            letterSpacing: '-0.01em',
          }}
          onMouseEnter={e => {
            if (!isLoading) e.currentTarget.style.background = 'var(--accent-hover)'
          }}
          onMouseLeave={e => {
            if (!isLoading) e.currentTarget.style.background = 'var(--accent)'
          }}
        >
          {isLoading ? 'Signing in…' : 'Sign in'}
        </button>
      </form>
    </div>
  )
}
