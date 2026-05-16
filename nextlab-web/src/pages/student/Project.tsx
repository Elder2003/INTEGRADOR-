import { useNavigate } from 'react-router-dom'
import { CheckCircle, Circle, Clock, ChevronRight, Mic } from 'lucide-react'
import Topbar from '../../components/student/Topbar'
import TimerDisplay from '../../components/student/TimerDisplay'
import { proyecto } from '../../mocks/proyecto'
import type { EtapaEstado } from '../../types'

function EtapaIcon({ estado }: { estado: EtapaEstado }) {
  if (estado === 'entregada') return <CheckCircle size={18} color="var(--color-success)" />
  if (estado === 'activa') return <Clock size={18} color="var(--color-accent)" />
  return <Circle size={18} color="var(--color-text-muted)" />
}

function colorEstado(estado: EtapaEstado): string {
  if (estado === 'entregada') return 'var(--color-success)'
  if (estado === 'activa') return 'var(--color-accent)'
  return 'var(--color-text-muted)'
}

export default function Project() {
  const navigate = useNavigate()
  const etapas = proyecto.etapas ?? []
  const entregadas = etapas.filter(e => e.estado === 'entregada').length
  const pct = etapas.length > 0 ? (entregadas / etapas.length) * 100 : 0

  const defensaFutura = proyecto.defensa
    ? new Date(proyecto.defensa.fecha) > new Date()
    : false

  return (
    <div
      style={{
        minHeight: '100vh',
        backgroundColor: 'var(--color-bg)',
        display: 'flex',
        flexDirection: 'column',
      }}
    >
      <Topbar title="Vista general del proyecto" subtitle={proyecto.materia} />

      <main
        style={{
          flex: 1,
          maxWidth: 800,
          margin: '0 auto',
          padding: '32px 24px',
          width: '100%',
          borderLeft: '4px solid var(--color-stripe)',
        }}
      >
        {/* Project header */}
        <div className="card" style={{ marginBottom: 24 }}>
          <h1
            style={{
              fontSize: 'var(--font-size-lg)',
              fontWeight: 700,
              marginBottom: 6,
            }}
          >
            {proyecto.titulo}
          </h1>
          <p style={{ color: 'var(--color-text-secondary)', marginBottom: 12, lineHeight: 1.6 }}>
            {proyecto.descripcion}
          </p>
          <div
            style={{
              fontSize: 'var(--font-size-sm)',
              color: 'var(--color-text-secondary)',
              marginBottom: 14,
            }}
          >
            Docente: {proyecto.docente} — {proyecto.materia}
          </div>

          {/* Progress */}
          <div
            style={{
              display: 'flex',
              justifyContent: 'space-between',
              alignItems: 'center',
              marginBottom: 8,
            }}
          >
            <span style={{ fontSize: 'var(--font-size-sm)', color: 'var(--color-text-secondary)' }}>
              Progreso general
            </span>
            <span
              style={{
                fontSize: 'var(--font-size-sm)',
                fontWeight: 600,
                color: 'var(--color-text-primary)',
              }}
            >
              {entregadas}/{etapas.length} etapas
            </span>
          </div>
          <div
            style={{
              height: 6,
              backgroundColor: 'var(--color-surface-2)',
              borderRadius: 3,
              overflow: 'hidden',
            }}
          >
            <div
              style={{
                height: '100%',
                width: `${pct}%`,
                backgroundColor: 'var(--color-accent)',
                borderRadius: 3,
                transition: 'width 0.5s ease',
              }}
            />
          </div>
        </div>

        {/* Stages timeline */}
        <h2
          style={{
            fontWeight: 600,
            fontSize: 'var(--font-size-md)',
            marginBottom: 14,
          }}
        >
          Etapas del proyecto
        </h2>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 10, marginBottom: 28 }}>
          {etapas.map((etapa, i) => (
            <div
              key={etapa.id}
              className="card"
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 14,
                opacity: etapa.estado === 'pendiente' ? 0.65 : 1,
              }}
            >
              <div
                style={{
                  display: 'flex',
                  flexDirection: 'column',
                  alignItems: 'center',
                  gap: 0,
                }}
              >
                <EtapaIcon estado={etapa.estado} />
                {i < etapas.length - 1 && (
                  <div
                    style={{
                      width: 2,
                      height: 28,
                      backgroundColor: 'var(--color-border)',
                      marginTop: 4,
                    }}
                  />
                )}
              </div>
              <div style={{ flex: 1 }}>
                <div
                  style={{
                    fontWeight: 600,
                    color: colorEstado(etapa.estado),
                    marginBottom: 2,
                  }}
                >
                  {etapa.nombre}
                </div>
                <div
                  style={{
                    fontSize: 'var(--font-size-xs)',
                    color: 'var(--color-text-muted)',
                  }}
                >
                  Fecha límite:{' '}
                  {new Date(etapa.fechaLimite).toLocaleDateString('es-BO', {
                    day: 'numeric',
                    month: 'long',
                    year: 'numeric',
                  })}
                </div>
              </div>
              <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
                {etapa.estado === 'activa' && (
                  <TimerDisplay fechaLimite={etapa.fechaLimite} compact />
                )}
                {etapa.estado === 'activa' && (
                  <button
                    className="btn-primary"
                    onClick={() =>
                      navigate(`/student/project/${proyecto.id}/stage/${etapa.id}`)
                    }
                    style={{ padding: '6px 14px', fontSize: 'var(--font-size-xs)', gap: 4 }}
                  >
                    Entregar
                    <ChevronRight size={12} />
                  </button>
                )}
                {etapa.estado === 'entregada' && (
                  <span
                    style={{
                      fontSize: 'var(--font-size-xs)',
                      color: 'var(--color-success)',
                      fontWeight: 500,
                    }}
                  >
                    Entregada
                  </span>
                )}
              </div>
            </div>
          ))}
        </div>

        {/* Defense section */}
        {proyecto.defensa && (
          <div className="card">
            <div
              style={{
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'space-between',
                gap: 14,
              }}
            >
              <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
                <Mic size={18} color="var(--color-accent)" />
                <div>
                  <div style={{ fontWeight: 600, marginBottom: 2 }}>Defensa oral</div>
                  <div
                    style={{
                      fontSize: 'var(--font-size-sm)',
                      color: 'var(--color-text-secondary)',
                    }}
                  >
                    {new Date(proyecto.defensa.fecha).toLocaleString('es-BO', {
                      dateStyle: 'full',
                      timeStyle: 'short',
                    })}
                  </div>
                </div>
              </div>
              <button
                className="btn-primary"
                onClick={() =>
                  navigate(`/student/defense/${proyecto.defensa!.id}`)
                }
                disabled={defensaFutura}
                style={{ padding: '8px 16px', fontSize: 'var(--font-size-sm)' }}
              >
                {defensaFutura ? 'Aún no disponible' : 'Ingresar a defensa'}
              </button>
            </div>
          </div>
        )}
      </main>
    </div>
  )
}
