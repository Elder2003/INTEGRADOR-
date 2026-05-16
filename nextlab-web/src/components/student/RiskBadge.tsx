type Risk = 'alto' | 'medio' | 'bajo'

interface Props {
  nivel: Risk
}

const config: Record<Risk, { label: string; color: string; bg: string }> = {
  alto:  { label: 'Riesgo alto',  color: 'var(--color-danger)',  bg: 'rgba(224,82,82,0.1)' },
  medio: { label: 'Riesgo medio', color: 'var(--color-warning)', bg: 'rgba(240,165,0,0.1)' },
  bajo:  { label: 'Riesgo bajo',  color: 'var(--color-success)', bg: 'rgba(63,166,110,0.1)' },
}

export default function RiskBadge({ nivel }: Props) {
  const { label, color, bg } = config[nivel]
  return (
    <span
      style={{
        display: 'inline-flex',
        alignItems: 'center',
        gap: 4,
        padding: '2px 8px',
        borderRadius: 12,
        fontSize: 'var(--font-size-xs)',
        fontWeight: 500,
        color,
        backgroundColor: bg,
      }}
    >
      <span
        style={{
          width: 6,
          height: 6,
          borderRadius: '50%',
          backgroundColor: color,
          display: 'inline-block',
        }}
      />
      {label}
    </span>
  )
}
