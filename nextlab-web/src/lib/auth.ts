import type { JWTClaims } from '../types'

const REFRESH_KEY = 'nextlab_refresh_token'

export function getRefreshToken(): string | null {
  return localStorage.getItem(REFRESH_KEY)
}

export function setTokens(_accessToken: string, refreshToken: string): void {
  localStorage.setItem(REFRESH_KEY, refreshToken)
}

export function clearTokens(): void {
  localStorage.removeItem(REFRESH_KEY)
}

export function decodeJWT(token: string): JWTClaims {
  const payload = token.split('.')[1]
  const decoded = atob(payload.replace(/-/g, '+').replace(/_/g, '/'))
  return JSON.parse(decoded) as JWTClaims
}
