import { AlertTriangle } from 'lucide-react'

interface Props {
  open: boolean
  title: string
  message: string
  confirmLabel?: string
  cancelLabel?: string
  onConfirm: () => void
  onCancel: () => void
  danger?: boolean
}

export default function ConfirmModal({
  open,
  title,
  message,
  confirmLabel = 'Confirmar',
  cancelLabel = 'Cancelar',
  onConfirm,
  onCancel,
  danger = false,
}: Props) {
  if (!open) return null

  return (
    <div
      style={{
        position: 'fixed',
        inset: 0,
        backgroundColor: 'rgba(0,0,0,0.75)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        zIndex: 1000,
      }}
      onClick={onCancel}
    >
      <div
        onClick={e => e.stopPropagation()}
        style={{
          backgroundColor: 'var(--color-surface)',
          border: '1px solid var(--color-border)',
          borderRadius: 12,
          padding: 24,
          maxWidth: 440,
          width: '90%',
        }}
      >
        <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 12 }}>
          <AlertTriangle
            size={18}
            color={danger ? 'var(--color-danger)' : 'var(--color-warning)'}
          />
          <h3 style={{ fontWeight: 600, fontSize: 'var(--font-size-md)' }}>{title}</h3>
        </div>
        <p
          style={{
            color: 'var(--color-text-secondary)',
            marginBottom: 22,
            lineHeight: 1.6,
            fontSize: 'var(--font-size-sm)',
          }}
        >
          {message}
        </p>
        <div style={{ display: 'flex', gap: 10, justifyContent: 'flex-end' }}>
          <button className="btn-secondary" onClick={onCancel}>
            {cancelLabel}
          </button>
          <button
            className="btn-primary"
            onClick={onConfirm}
            style={danger ? { backgroundColor: 'var(--color-danger)' } : undefined}
          >
            {confirmLabel}
          </button>
        </div>
      </div>
    </div>
  )
}
