import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '../../context/AuthContext'

export function Topbar() {
  const { user, logout } = useAuth()
  const navigate = useNavigate()
  const [signingOut, setSigningOut] = useState(false)

  async function handleSignOut() {
    setSigningOut(true)
    try {
      await logout()
    } finally {
      setSigningOut(false)
    }
    navigate('/login', { replace: true })
  }

  return (
    <header className="flex items-center justify-between bg-[#1a1a1a] border-b border-[#2e2e2e] px-6 h-14">
      <span className="text-[#E85D24] font-bold text-lg tracking-tight">NextLab</span>
      <div className="flex items-center gap-4">
        <span className="text-sm text-[#888888]">{user?.email}</span>
        <button
          type="button"
          onClick={handleSignOut}
          disabled={signingOut}
          className="text-sm text-[#888888] hover:text-[#f0f0f0] disabled:opacity-50"
        >
          {signingOut ? 'Signing out…' : 'Sign out'}
        </button>
      </div>
    </header>
  )
}
