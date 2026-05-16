import { useNavigate } from 'react-router-dom'
import { Shield } from 'lucide-react'
import ActivityTypeBadge from './ActivityTypeBadge'
import TimerDisplay from './TimerDisplay'
import type { Actividad } from '../../types'

interface Props {
  actividad: Actividad
}

function routeFor(a: Actividad): string {
  switch (a.tipo) {
    case 'examen':   return `/student/exam/${a.id}`
    case 'practica': return `/student/practice/${a.id}`
    case 'tarea':    return `/student/task/${a.id}`
    case 'informe':  return `/student/report/${a.id}`
    case 'proyecto': return `/student/project/${a.id}`
  }
}

export default function ActivityCard({ actividad }: Props) {
  const navigate = useNavigate()

  return (
    <div
      onClick={() => navigate(routeFor(actividad))}
      className="card"
      style={{
        cursor: 'pointer',
        borderLeft: '3px solid var(--color-accent)',
        transition: 'border-color 0.2s, box-shadow 0.2s',
      }}
      onMouseEnter={e => {
        ;(e.currentTarget as HTMLElement).style.boxShadow =
          '0 0 0 1px var(--color-accent)'
      }}
      onMouseLeave={e => {
        ;(e.currentTarget as HTMLElement).style.boxShadow = 'none'
      }}
    >
      <div
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'flex-start',
          marginBottom: 10,
          gap: 8,
          flexWrap: 'wrap',
        }}
      >
        <ActivityTypeBadge tipo={actividad.tipo} />
        {actividad.auditoria && (
          <span
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: 4,
              fontSize: 'var(--font-size-xs)',
              color: 'var(--color-accent)',
              backgroundColor: 'var(--color-accent-dim)',
              padding: '2px 8px',
              borderRadius: 4,
              fontWeight: 500,
            }}
          >
            <Shield size={10} />
            Auditoría activa
          </span>
        )}
      </div>

      <div
        style={{
          fontWeight: 600,
          color: 'var(--color-text-primary)',
          marginBottom: 4,
          fontSize: 'var(--font-size-base)',
          lineHeight: 1.4,
        }}
      >
        {actividad.titulo}
      </div>
      <div
        style={{
          fontSize: 'var(--font-size-sm)',
          color: 'var(--color-text-secondary)',
          marginBottom: 12,
        }}
      >
        {actividad.materia} — {actividad.docente}
      </div>

      {actividad.fechaLimite ? (
        <TimerDisplay fechaLimite={actividad.fechaLimite} compact />
      ) : (
        <span
          style={{
            fontSize: 'var(--font-size-xs)',
            color: 'var(--color-text-muted)',
            fontStyle: 'italic',
          }}
        >
          Sin fecha límite
        </span>
      )}
    </div>
  )
}
