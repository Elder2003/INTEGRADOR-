import { useState, useEffect } from 'react'
import { Clock } from 'lucide-react'

interface Props {
  fechaLimite: string
  compact?: boolean
  onExpire?: () => void
  warningThresholdMinutes?: number
  dangerThresholdMinutes?: number
}

function getSecondsLeft(fechaLimite: string): number {
  return Math.max(0, Math.floor((new Date(fechaLimite).getTime() - Date.now()) / 1000))
}

function formatTime(seconds: number): string {
  const h = Math.floor(seconds / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  const s = seconds % 60
  if (h > 0) return `${h}h ${m.toString().padStart(2, '0')}m`
  return `${m.toString().padStart(2, '0')}:${s.toString().padStart(2, '0')}`
}

export default function TimerDisplay({
  fechaLimite,
  compact = false,
  onExpire,
  warningThresholdMinutes = 10,
  dangerThresholdMinutes = 5,
}: Props) {
  const [secondsLeft, setSecondsLeft] = useState(() => getSecondsLeft(fechaLimite))

  useEffect(() => {
    const interval = setInterval(() => {
      const secs = getSecondsLeft(fechaLimite)
      setSecondsLeft(secs)
      if (secs === 0 && onExpire) onExpire()
    }, 1000)
    return () => clearInterval(interval)
  }, [fechaLimite, onExpire])

  const minutesLeft = secondsLeft / 60
  const color =
    minutesLeft <= dangerThresholdMinutes
      ? 'var(--color-danger)'
      : minutesLeft <= warningThresholdMinutes
      ? 'var(--color-warning)'
      : 'var(--color-text-secondary)'

  const isExpired = secondsLeft === 0

  if (compact) {
    return (
      <div style={{ display: 'flex', alignItems: 'center', gap: 6, color }}>
        <Clock size={13} />
        <span
          style={{
            fontSize: 'var(--font-size-sm)',
            fontWeight: 600,
            fontFamily: 'var(--font-mono)',
          }}
        >
          {isExpired ? 'Vencida' : formatTime(secondsLeft)}
        </span>
      </div>
    )
  }

  return (
    <div
      style={{
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        gap: 4,
        padding: '14px 20px',
        backgroundColor: 'var(--color-surface-2)',
        borderRadius: 8,
        border: `1px solid ${isExpired ? 'var(--color-danger)' : color === 'var(--color-text-secondary)' ? 'var(--color-border)' : color}`,
      }}
    >
      <div
        style={{
          fontSize: 'var(--font-size-xs)',
          color: 'var(--color-text-muted)',
          letterSpacing: '0.08em',
          fontWeight: 500,
        }}
      >
        TIEMPO RESTANTE
      </div>
      <div
        style={{
          fontSize: 30,
          fontWeight: 700,
          color,
          fontFamily: 'var(--font-mono)',
          letterSpacing: '0.04em',
        }}
      >
        {isExpired ? '00:00' : formatTime(secondsLeft)}
      </div>
    </div>
  )
}
