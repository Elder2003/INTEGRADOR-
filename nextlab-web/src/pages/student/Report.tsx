import { useState } from 'react'
import { CheckCircle, AlertTriangle, Info } from 'lucide-react'
import Topbar from '../../components/student/Topbar'
import TimerDisplay from '../../components/student/TimerDisplay'
import FileUpload from '../../components/student/FileUpload'
import ConfirmModal from '../../components/student/ConfirmModal'
import { actividades } from '../../mocks/actividades'

const informe = actividades.find(a => a.id === 'act-003')!

const formatoRequisitos = [
  'Portada con campos requeridos',
  'Formato de citación APA 7',
  'Secciones mínimas: Introducción, Desarrollo, Conclusión, Bibliografía',
  'Mínimo 5 fuentes bibliográficas',
  'Extensión: 8 a 15 páginas',
  'Fuente: Times New Roman 12pt o Arial 11pt',
]

export default function Report() {
  const [archivo, setArchivo] = useState<File | null>(null)
  const [showModal, setShowModal] = useState(false)
  const [entregado, setEntregado] = useState(false)
  const [timestamp, setTimestamp] = useState('')

  const isLate = informe.fechaLimite
    ? new Date() > new Date(informe.fechaLimite)
    : false

  function handleEntregar() {
    setEntregado(true)
    setTimestamp(new Date().toLocaleString('es-BO'))
    setShowModal(false)
  }

  return (
    <div
      style={{
        minHeight: '100vh',
        backgroundColor: 'var(--color-bg)',
        display: 'flex',
        flexDirection: 'column',
      }}
    >
      <Topbar title="Subida de informe" subtitle={informe.materia} />

      <main
        style={{
          flex: 1,
          maxWidth: 760,
          margin: '0 auto',
          padding: '32px 24px',
          width: '100%',
          borderLeft: '4px solid var(--color-stripe)',
        }}
      >
        {isLate && !entregado && (
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: 8,
              padding: '10px 14px',
              borderRadius: 6,
              backgroundColor: 'rgba(240,165,0,0.08)',
              border: '1px solid rgba(240,165,0,0.3)',
              marginBottom: 22,
            }}
          >
            <AlertTriangle size={15} color="var(--color-warning)" />
            <span style={{ fontSize: 'var(--font-size-sm)', color: 'var(--color-warning)' }}>
              Esta entrega será marcada como tardía. El docente decidirá si la acepta.
            </span>
          </div>
        )}

        <div style={{ marginBottom: 24 }}>
          <h1
            style={{
              fontSize: 'var(--font-size-lg)',
              fontWeight: 700,
              color: 'var(--color-text-primary)',
              marginBottom: 6,
            }}
          >
            {informe.titulo}
          </h1>
          <p
            style={{
              color: 'var(--color-text-secondary)',
              marginBottom: 12,
            }}
          >
            {informe.docente} — {informe.materia}
          </p>
          {informe.fechaLimite && (
            <TimerDisplay fechaLimite={informe.fechaLimite} compact />
          )}
        </div>

        {/* Format requirements */}
        <div
          className="card"
          style={{ marginBottom: 18 }}
        >
          <h3
            style={{
              fontWeight: 600,
              marginBottom: 12,
              color: 'var(--color-text-primary)',
              display: 'flex',
              alignItems: 'center',
              gap: 6,
            }}
          >
            Parámetros de formato (validación automática)
          </h3>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
            {formatoRequisitos.map(req => (
              <div
                key={req}
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: 8,
                  fontSize: 'var(--font-size-sm)',
                  color: 'var(--color-text-secondary)',
                }}
              >
                <CheckCircle size={13} color="var(--color-success)" style={{ flexShrink: 0 }} />
                {req}
              </div>
            ))}
          </div>
          <div
            style={{
              marginTop: 14,
              padding: '10px 12px',
              borderRadius: 6,
              backgroundColor: 'rgba(74,158,218,0.07)',
              border: '1px solid rgba(74,158,218,0.2)',
              display: 'flex',
              gap: 8,
              alignItems: 'flex-start',
            }}
          >
            <Info size={13} color="var(--color-info)" style={{ flexShrink: 0, marginTop: 1 }} />
            <span style={{ fontSize: 'var(--font-size-xs)', color: 'var(--color-info)' }}>
              La validación de formato es automática. El resultado lo verá tu docente. Tu entrega
              siempre será aceptada.
            </span>
          </div>
        </div>

        {entregado ? (
          <div
            style={{
              backgroundColor: 'rgba(63,166,110,0.06)',
              border: '1px solid var(--color-success)',
              borderRadius: 8,
              padding: '20px 22px',
              display: 'flex',
              gap: 14,
              alignItems: 'flex-start',
            }}
          >
            <CheckCircle size={22} color="var(--color-success)" style={{ flexShrink: 0 }} />
            <div>
              <div style={{ fontWeight: 600, color: 'var(--color-success)', marginBottom: 4 }}>
                Informe entregado
              </div>
              <div style={{ fontSize: 'var(--font-size-sm)', color: 'var(--color-text-secondary)' }}>
                Archivo: <strong>{archivo?.name}</strong>
              </div>
              <div
                style={{
                  fontSize: 'var(--font-size-sm)',
                  color: 'var(--color-text-secondary)',
                  marginTop: 2,
                }}
              >
                Entregado el {timestamp}
              </div>
            </div>
          </div>
        ) : (
          <>
            <div className="card" style={{ marginBottom: 16 }}>
              <h3 style={{ fontWeight: 600, marginBottom: 14 }}>Subir informe</h3>
              <FileUpload
                acceptedTypes={['.pdf', '.docx']}
                acceptedLabels={['PDF', 'Word (.docx)']}
                onFileSelect={setArchivo}
                selectedFile={archivo}
                onClear={() => setArchivo(null)}
              />
            </div>
            <button
              className="btn-primary"
              onClick={() => setShowModal(true)}
              disabled={!archivo}
              style={{ width: '100%', padding: '12px 0' }}
            >
              Confirmar entrega
            </button>
          </>
        )}
      </main>

      <ConfirmModal
        open={showModal}
        title="Confirmar entrega de informe"
        message={`¿Confirmas la entrega de "${archivo?.name}"?${isLate ? ' Esta entrega quedará marcada como tardía.' : ''} El sistema validará el formato automáticamente.`}
        confirmLabel="Sí, entregar"
        onConfirm={handleEntregar}
        onCancel={() => setShowModal(false)}
      />
    </div>
  )
}
