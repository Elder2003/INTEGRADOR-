import { useNavigate } from 'react-router-dom'
import { AlertCircle } from 'lucide-react'
import Topbar from '../../components/student/Topbar'
import ActivityCard from '../../components/student/ActivityCard'
import { actividades, calificacionesRecientes } from '../../mocks/actividades'

const S: Record<string, React.CSSProperties> = {
  page: { minHeight: '100vh', backgroundColor: 'var(--color-bg)', display: 'flex', flexDirection: 'column' },
  main: { maxWidth: 1100, margin: '0 auto', padding: '32px 24px', width: '100%', flex: 1 },
  alertBar: {
    display: 'flex',
    alignItems: 'center',
    gap: 8,
    padding: '10px 14px',
    borderRadius: 6,
    backgroundColor: 'rgba(224,82,82,0.08)',
    border: '1px solid rgba(224,82,82,0.25)',
    marginBottom: 28,
  },
  section: {
    marginBottom: 36,
    borderLeft: '4px solid var(--color-stripe)',
    paddingLeft: 18,
  },
  sectionTitle: {
    fontWeight: 700,
    fontSize: 'var(--font-size-md)',
    color: 'var(--color-text-primary)',
    marginBottom: 16,
  },
  grid: {
    display: 'grid',
    gridTemplateColumns: 'repeat(auto-fill, minmax(270px, 1fr))',
    gap: 14,
  },
}

export default function Dashboard() {
  const navigate = useNavigate()
  const activas = actividades.filter(a => a.estado === 'activa')
  const proximas = actividades.filter(a => a.estado === 'proxima')
  const sinEntregar = activas.length

  return (
    <div style={S.page}>
      <Topbar title="Panel del estudiante" />
      <main style={S.main}>
        {sinEntregar > 0 && (
          <div style={S.alertBar}>
            <AlertCircle size={15} color="var(--color-danger)" />
            <span style={{ fontSize: 'var(--font-size-sm)', color: 'var(--color-danger)' }}>
              Tienes {sinEntregar} actividad{sinEntregar > 1 ? 'es' : ''} pendiente
              {sinEntregar > 1 ? 's' : ''} de entrega
            </span>
          </div>
        )}

        <section style={S.section}>
          <h2 style={S.sectionTitle}>Actividades activas</h2>
          <div style={S.grid}>
            {activas.map(a => (
              <ActivityCard key={a.id} actividad={a} />
            ))}
          </div>
        </section>

        {proximas.length > 0 && (
          <section style={S.section}>
            <h2 style={S.sectionTitle}>Próximamente</h2>
            <div style={S.grid}>
              {proximas.map(a => (
                <ActivityCard key={a.id} actividad={a} />
              ))}
            </div>
          </section>
        )}

        <section style={S.section}>
          <h2 style={S.sectionTitle}>Calificaciones recientes</h2>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
            {calificacionesRecientes.map(cal => (
              <div
                key={cal.id}
                className="card"
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'space-between',
                  gap: 16,
                }}
              >
                <div>
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
                      fontSize: 'var(--font-size-sm)',
                      color: 'var(--color-text-secondary)',
                    }}
                  >
                    {cal.materia}
                  </div>
                </div>
                <div style={{ display: 'flex', alignItems: 'center', gap: 14 }}>
                  {cal.tieneAlerta && (
                    <AlertCircle size={15} color="var(--color-warning)" />
                  )}
                  <div style={{ textAlign: 'right' }}>
                    <span
                      style={{
                        fontSize: 'var(--font-size-lg)',
                        fontWeight: 700,
                        color:
                          cal.nota >= 51 ? 'var(--color-success)' : 'var(--color-danger)',
                        fontFamily: 'var(--font-mono)',
                      }}
                    >
                      {cal.nota}
                    </span>
                    <span
                      style={{
                        fontSize: 'var(--font-size-xs)',
                        color: 'var(--color-text-muted)',
                      }}
                    >
                      /{cal.notaMaxima}
                    </span>
                  </div>
                  <button
                    className="btn-secondary"
                    onClick={() => navigate('/student/grades')}
                    style={{ fontSize: 'var(--font-size-xs)', padding: '4px 10px' }}
                  >
                    Ver detalle
                  </button>
                </div>
              </div>
            ))}
          </div>
        </section>
      </main>
    </div>
  )
}
