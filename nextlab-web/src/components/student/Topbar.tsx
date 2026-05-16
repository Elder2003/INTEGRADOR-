import type { ReactNode } from 'react'
import { User } from 'lucide-react'
import { useStudent } from '../../context/StudentContext'

interface TopbarProps {
  title: string
  subtitle?: string
  rightSlot?: ReactNode
}

export default function Topbar({ title, subtitle, rightSlot }: TopbarProps) {
  const { usuario } = useStudent()

  return (
    <header
      style={{
        backgroundColor: 'var(--color-surface)',
        borderBottom: '1px solid var(--color-border)',
        padding: '0 24px',
        height: 56,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        position: 'sticky',
        top: 0,
        zIndex: 100,
        flexShrink: 0,
      }}
    >
      <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
        <span
          style={{
            color: 'var(--color-accent)',
            fontWeight: 700,
            fontSize: 'var(--font-size-md)',
            letterSpacing: '0.12em',
          }}
        >
          UNIFRANZ
        </span>
        <div style={{ width: 1, height: 22, backgroundColor: 'var(--color-border)' }} />
        <div>
          <div
            style={{
              fontWeight: 600,
              color: 'var(--color-text-primary)',
              fontSize: 'var(--font-size-base)',
              lineHeight: subtitle ? 1.2 : undefined,
            }}
          >
            {title}
          </div>
          {subtitle && (
            <div
              style={{
                fontSize: 'var(--font-size-xs)',
                color: 'var(--color-text-secondary)',
              }}
            >
              {subtitle}
            </div>
          )}
        </div>
      </div>

      <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
        {rightSlot}
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <div
            style={{
              width: 30,
              height: 30,
              borderRadius: '50%',
              backgroundColor: 'var(--color-surface-2)',
              border: '1px solid var(--color-border)',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              flexShrink: 0,
            }}
          >
            <User size={14} color="var(--color-text-secondary)" />
          </div>
          <div>
            <div
              style={{
                fontSize: 'var(--font-size-sm)',
                color: 'var(--color-text-primary)',
                fontWeight: 500,
                whiteSpace: 'nowrap',
              }}
            >
              {usuario.nombre}
            </div>
            <div
              style={{
                fontSize: 'var(--font-size-xs)',
                color: 'var(--color-text-secondary)',
              }}
            >
              {usuario.semestre}
            </div>
          </div>
        </div>
      </div>
    </header>
  )
}
