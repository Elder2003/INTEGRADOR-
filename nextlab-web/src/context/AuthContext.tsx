import { createContext, useContext, useState, useEffect, type ReactNode } from 'react'
import { login as apiLogin, logout as apiLogout, refreshToken } from '../api/auth'
import { clearTokens, decodeJWT, getRefreshToken, setTokens } from '../lib/auth'
import { setAccessTokenGetter } from '../lib/api'
import type { User } from '../types'

interface AuthContextValue {
  user: User | null
  accessToken: string | null
  isLoading: boolean
  login: (email: string, password: string) => Promise<void>
  logout: () => Promise<void>
}

const AuthContext = createContext<AuthContextValue | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [accessToken, setAccessToken] = useState<string | null>(null)
  const [isLoading, setIsLoading] = useState(() => Boolean(getRefreshToken()))

  useEffect(() => {
    setAccessTokenGetter(() => accessToken)
  }, [accessToken])

  useEffect(() => {
    const rt = getRefreshToken()
    if (!rt) {
      return
    }
    refreshToken(rt)
      .then((data) => {
        const claims = decodeJWT(data.access_token)
        setUser({ id: claims.sub, email: claims.email, role: claims.role })
        setAccessToken(data.access_token)
        setTokens(data.access_token, data.refresh_token)
      })
      .catch(() => {
        clearTokens()
      })
      .finally(() => {
        setIsLoading(false)
      })
  }, [])

  async function login(email: string, password: string): Promise<void> {
    const data = await apiLogin(email, password)
    const claims = decodeJWT(data.access_token)
    const newUser: User = { id: claims.sub, email: claims.email, role: claims.role }
    setUser(newUser)
    setAccessToken(data.access_token)
    setTokens(data.access_token, data.refresh_token)
  }

  async function logout(): Promise<void> {
    const rt = getRefreshToken()
    if (rt) {
      try {
        await apiLogout(rt)
      } catch {
        // proceed with local cleanup regardless
      }
    }
    clearTokens()
    setUser(null)
    setAccessToken(null)
  }

  return (
    <AuthContext.Provider value={{ user, accessToken, isLoading, login, logout }}>
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used within AuthProvider')
  return ctx
}
