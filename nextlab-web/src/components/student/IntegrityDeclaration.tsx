interface Props {
  checked: boolean
  onChange: (checked: boolean) => void
}

export default function IntegrityDeclaration({ checked, onChange }: Props) {
  return (
    <label
      style={{
        display: 'flex',
        alignItems: 'flex-start',
        gap: 10,
        cursor: 'pointer',
        padding: '12px 14px',
        borderRadius: 6,
        border: `1px solid ${checked ? 'var(--color-accent)' : 'var(--color-border)'}`,
        backgroundColor: checked ? 'var(--color-accent-dim)' : 'transparent',
        transition: 'border-color 0.2s, background-color 0.2s',
      }}
    >
      <input
        type="checkbox"
        checked={checked}
        onChange={e => onChange(e.target.checked)}
        style={{
          marginTop: 2,
          accentColor: 'var(--color-accent)',
          width: 16,
          height: 16,
          flexShrink: 0,
          cursor: 'pointer',
        }}
      />
      <span
        style={{
          fontSize: 'var(--font-size-sm)',
          color: 'var(--color-text-secondary)',
          lineHeight: 1.6,
        }}
      >
        Declaro que este trabajo es de mi autoría y que no he utilizado herramientas de inteligencia
        artificial generativa no declaradas en su elaboración. Entiendo que el sistema de auditoría
        ha registrado mi proceso de resolución y que cualquier anomalía detectada será revisada por
        mi docente.
      </span>
    </label>
  )
}
