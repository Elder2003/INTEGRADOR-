import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { AlertCircle, X } from 'lucide-react'
import Topbar from '../../components/student/Topbar'
import ActivityTypeBadge from '../../components/student/ActivityTypeBadge'
import { calificaciones } from '../../mocks/calificaciones'
import type { Calificacion } from '../../types'

function NotaBar({ nota, notaMaxima }: { nota: number; notaMaxima: number }) {
  const pct = (nota / notaMaxima) * 100
  const color =
    nota >= 51 ? 'var(--color-success)' : 'var(--color-danger)'

  return (
    <div style={{ display: 'flex', alignItems: 'center', gap: 10, minWidth: 120 }}>
      <div
        style={{
          flex: 1,
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
            backgroundColor: color,
            borderRadius: 3,
          }}
        />
      </div>
      <span
        style={{
          fontFamily: 'var(--font-mono)',
          fontWeight: 700,
          color,
          fontSize: 'var(--font-size-md)',
          minWidth: 28,
          textAlign: 'right',
        }}
      >
        {nota}
      </span>
      <span style={{ fontSize: 'var(--font-size-xs)', color: 'var(--color-text-muted)' }}>
        /{notaMaxima}
      </span>
    </div>
  )
}

function promedio(cals: Calificacion[]): number {
  if (!cals.length) return 0
  return Math.round(cals.reduce((acc, c) => acc + (c.nota / c.notaMaxima) * 100, 0) / cals.length)
}

export default function Grades() {
  const navigate = useNavigate()
  const [alertaCal, setAlertaCal] = useState<Calificacion | null>(null)

  const materias = [...new Set(calificaciones.map(c => c.materia))]

  return (
    <div
      style={{
        minHeight: '100vh',
        backgroundColor: 'var(--color-bg)',
        display: 'flex',
        flexDirection: 'column',
      }}
    >
      <Topbar title="Calificaciones y feedback" />

      <main
        style={{
          flex: 1,
          maxWidth: 860,
          margin: '0 auto',
          padding: '32px 24px',
          width: '100%',
          borderLeft: '4px solid var(--color-stripe)',
        }}
      >
        {/* General average */}
        <div
          className="card"
          style={{
            marginBottom: 28,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            gap: 20,
          }}
        >
          <div>
            <div
              style={{
                fontSize: 'var(--font-size-xs)',
                color: 'var(--color-text-muted)',
                letterSpacing: '0.06em',
                marginBottom: 4,
              }}
            >
              PROMEDIO GENERAL
            </div>
            <div
              style={{
                fontSize: 36,
                fontWeight: 800,
                fontFamily: 'var(--font-mono)',
                color:
                  promedio(calificaciones) >= 51
                    ? 'var(--color-success)'
                    : 'var(--color-danger)',
              }}
            >
              {promedio(calificaciones)}
            </div>
          </div>
          <div
            style={{
              display: 'flex',
              gap: 20,
              flexWrap: 'wrap',
            }}
          >
            {materias.map(m => {
              const cals = calificaciones.filter(c => c.materia === m)
              return (
                <div key={m} style={{ textAlign: 'center' }}>
                  <div
                    style={{
                      fontSize: 'var(--font-size-xs)',
                      color: 'var(--color-text-muted)',
                      marginBottom: 2,
                    }}
                  >
                    {m.split(' ')[0]}
                  </div>
                  <div
                    style={{
                      fontFamily: 'var(--font-mono)',
                      fontWeight: 700,
                      fontSize: 'var(--font-size-lg)',
                      color:
                        promedio(cals) >= 51 ? 'var(--color-success)' : 'var(--color-danger)',
                    }}
                  >
                    {promedio(cals)}
                  </div>
                </div>
              )
            })}
          </div>
        </div>

        {/* By subject */}
        {materias.map(materia => {
          const cals = calificaciones.filter(c => c.materia === materia)
          return (
            <section key={materia} style={{ marginBottom: 28 }}>
              <h2
                style={{
                  fontWeight: 700,
                  fontSize: 'var(--font-size-md)',
                  marginBottom: 12,
                  color: 'var(--color-text-primary)',
                }}
              >
                {materia}
              </h2>
              <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
                {cals.map(cal => (
                  <div
                    key={cal.id}
                    className="card"
                    style={{
                      display: 'flex',
                      alignItems: 'center',
                      gap: 14,
                      flexWrap: 'wrap',
                    }}
                  >
                    <ActivityTypeBadge tipo={cal.tipo} />
                    <div style={{ flex: 1, minWidth: 150 }}>
                      <div
                        style={{
                          fontWeight: 500,
                          color: 'var(--color-text-primary)',
                          marginBottom: 2,
                        }}
                      >
                        {cal.actividad}
                      </div>
                      <div
                        style={{
                          fontSize: 'var(--font-size-xs)',
                          color: 'var(--color-text-muted)',
                        }}
                      >
                        {cal.fecha}
                      </div>
                    </div>
                    {cal.tieneAlerta && (
                      <button
                        onClick={() => setAlertaCal(cal)}
                        style={{
                          display: 'flex',
                          alignItems: 'center',
                          gap: 4,
                          color: 'var(--color-warning)',
                          fontSize: 'var(--font-size-xs)',
                          fontWeight: 500,
                        }}
                        title="Ver mensaje del docente"
                      >
                        <AlertCircle size={14} />
                        Alerta
                      </button>
                    )}
                    <NotaBar nota={cal.nota} notaMaxima={cal.notaMaxima} />
                    {cal.puedeApelar && (
                      <button
                        className="btn-secondary"
                        onClick={() => navigate(`/student/appeal/${cal.id}`)}
                        style={{
                          fontSize: 'var(--font-size-xs)',
                          padding: '4px 10px',
                          whiteSpace: 'nowrap',
                        }}
                      >
                        Apelar
                      </button>
                    )}
                  </div>
                ))}
              </div>
            </section>
          )
        })}
      </main>

      {/* Alert modal */}
      {alertaCal && (
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
          onClick={() => setAlertaCal(null)}
        >
          <div
            onClick={e => e.stopPropagation()}
            style={{
              backgroundColor: 'var(--color-surface)',
              border: '1px solid var(--color-border)',
              borderRadius: 12,
              padding: 24,
              maxWidth: 460,
              width: '90%',
            }}
          >
            <div
              style={{
                display: 'flex',
                justifyContent: 'space-between',
                alignItems: 'center',
                marginBottom: 14,
              }}
            >
              <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                <AlertCircle size={18} color="var(--color-warning)" />
                <h3 style={{ fontWeight: 600, fontSize: 'var(--font-size-md)' }}>
                  Mensaje del docente
                </h3>
              </div>
              <button onClick={() => setAlertaCal(null)}>
                <X size={16} color="var(--color-text-muted)" />
              </button>
            </div>
            <p
              style={{
                fontSize: 'var(--font-size-sm)',
                color: 'var(--color-text-secondary)',
                lineHeight: 1.7,
                marginBottom: 16,
              }}
            >
              {alertaCal.mensajeDocente}
            </p>
            <div
              style={{
                fontSize: 'var(--font-size-xs)',
                color: 'var(--color-text-muted)',
                borderTop: '1px solid var(--color-border)',
                paddingTop: 12,
              }}
            >
              El análisis detallado del sistema de auditoría es confidencial y solo visible para
              el docente y el administrador institucional.
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
