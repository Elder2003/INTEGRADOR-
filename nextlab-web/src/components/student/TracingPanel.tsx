import { Activity, AlertTriangle, Edit3, Clock } from 'lucide-react'
import MiniBarChart from './MiniBarChart'

export interface TracingData {
  wpm: number
  activeTime: number
  pasteCount: number
  corrections: number
  activityHistory: number[]
}

interface Props {
  data: TracingData
  totalPreguntas: number
  preguntasRespondidas: number
  totalPalabras: number
}

function formatActiveTime(seconds: number): string {
  const m = Math.floor(seconds / 60)
  const s = seconds % 60
  return `${m}m ${s.toString().padStart(2, '0')}s`
}

function MetricBox({
  icon,
  label,
  value,
  highlight = false,
}: {
  icon: React.ReactNode
  label: string
  value: string
  highlight?: boolean
}) {
  return (
    <div
      style={{
        backgroundColor: 'var(--color-surface-2)',
        borderRadius: 6,
        padding: '8px 10px',
        border: `1px solid ${highlight ? 'var(--color-warning)' : 'var(--color-border)'}`,
      }}
    >
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: 4,
          color: 'var(--color-text-muted)',
          marginBottom: 4,
        }}
      >
        {icon}
        <span style={{ fontSize: 'var(--font-size-xs)' }}>{label}</span>
      </div>
      <div
        style={{
          fontSize: 'var(--font-size-md)',
          fontWeight: 700,
          color: highlight ? 'var(--color-warning)' : 'var(--color-text-primary)',
          fontFamily: 'var(--font-mono)',
        }}
      >
        {value}
      </div>
    </div>
  )
}

export default function TracingPanel({
  data,
  totalPreguntas,
  preguntasRespondidas,
  totalPalabras,
}: Props) {
  const pct = totalPreguntas > 0 ? (preguntasRespondidas / totalPreguntas) * 100 : 0

  return (
    <div
      style={{
        backgroundColor: 'var(--color-surface)',
        border: '1px solid var(--color-border)',
        borderRadius: 8,
        padding: 16,
        display: 'flex',
        flexDirection: 'column',
        gap: 14,
      }}
    >
      {/* Active badge */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
        <span
          style={{
            display: 'inline-block',
            width: 8,
            height: 8,
            borderRadius: '50%',
            backgroundColor: 'var(--color-success)',
            boxShadow: '0 0 6px var(--color-success)',
            animation: 'pulse-dot 2s ease-in-out infinite',
          }}
        />
        <span
          style={{
            fontSize: 'var(--font-size-sm)',
            color: 'var(--color-success)',
            fontWeight: 500,
          }}
        >
          Trazabilidad activa
        </span>
      </div>

      {/* Metrics */}
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 8 }}>
        <MetricBox icon={<Activity size={11} />} label="Velocidad" value={`${data.wpm} ppm`} />
        <MetricBox
          icon={<Clock size={11} />}
          label="Tiempo activo"
          value={formatActiveTime(data.activeTime)}
        />
        <MetricBox
          icon={<AlertTriangle size={11} />}
          label="Pastes"
          value={String(data.pasteCount)}
          highlight={data.pasteCount > 0}
        />
        <MetricBox
          icon={<Edit3 size={11} />}
          label="Correcciones"
          value={String(data.corrections)}
        />
      </div>

      {/* Activity chart */}
      <div>
        <div
          style={{
            fontSize: 'var(--font-size-xs)',
            color: 'var(--color-text-muted)',
            marginBottom: 6,
          }}
        >
          Actividad por minuto
        </div>
        <MiniBarChart data={data.activityHistory} />
      </div>

      {/* Progress */}
      <div>
        <div
          style={{
            display: 'flex',
            justifyContent: 'space-between',
            marginBottom: 6,
          }}
        >
          <span style={{ fontSize: 'var(--font-size-xs)', color: 'var(--color-text-muted)' }}>
            {preguntasRespondidas}/{totalPreguntas} preguntas
          </span>
          <span style={{ fontSize: 'var(--font-size-xs)', color: 'var(--color-text-muted)' }}>
            {totalPalabras} palabras
          </span>
        </div>
        <div
          style={{
            height: 4,
            backgroundColor: 'var(--color-surface-2)',
            borderRadius: 2,
            overflow: 'hidden',
          }}
        >
          <div
            style={{
              height: '100%',
              width: `${pct}%`,
              backgroundColor: 'var(--color-accent)',
              borderRadius: 2,
              transition: 'width 0.4s ease',
            }}
          />
        </div>
      </div>
    </div>
  )
}
