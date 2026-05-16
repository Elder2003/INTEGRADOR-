import { useState } from 'react'
import { AlertTriangle, CheckCircle } from 'lucide-react'
import Topbar from '../../components/student/Topbar'
import TimerDisplay from '../../components/student/TimerDisplay'
import FileUpload from '../../components/student/FileUpload'
import ConfirmModal from '../../components/student/ConfirmModal'
import { actividades } from '../../mocks/actividades'

const tarea = actividades.find(a => a.id === 'act-002')!

export default function Task() {
  const [archivo, setArchivo] = useState<File | null>(null)
  const [showModal, setShowModal] = useState(false)
  const [entregado, setEntregado] = useState(false)
  const [timestamp, setTimestamp] = useState('')

  const isLate = tarea.fechaLimite
    ? new Date() > new Date(tarea.fechaLimite)
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
      <Topbar title="Subida de tarea" subtitle={tarea.materia} />

      <main
        style={{
          flex: 1,
          maxWidth: 720,
          margin: '0 auto',
          padding: '32px 24px',
          width: '100%',
          borderLeft: '4px solid var(--color-stripe)',
        }}
      >
        {/* Late warning */}
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

        {/* Header */}
        <div style={{ marginBottom: 24 }}>
          <h1
            style={{
              fontSize: 'var(--font-size-lg)',
              fontWeight: 700,
              color: 'var(--color-text-primary)',
              marginBottom: 6,
            }}
          >
            {tarea.titulo}
          </h1>
          <p style={{ color: 'var(--color-text-secondary)', lineHeight: 1.6, marginBottom: 12 }}>
            {tarea.docente} — {tarea.materia}
          </p>
          {tarea.fechaLimite && (
            <TimerDisplay fechaLimite={tarea.fechaLimite} compact />
          )}
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
              <div
                style={{
                  fontWeight: 600,
                  color: 'var(--color-success)',
                  marginBottom: 4,
                }}
              >
                Tarea entregada
              </div>
              <div
                style={{
                  fontSize: 'var(--font-size-sm)',
                  color: 'var(--color-text-secondary)',
                }}
              >
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
            <div
              className="card"
              style={{ marginBottom: 16 }}
            >
              <h3
                style={{
                  fontWeight: 600,
                  marginBottom: 14,
                  color: 'var(--color-text-primary)',
                }}
              >
                Subir archivo
              </h3>
              <FileUpload
                acceptedTypes={['.pdf', '.doc', '.docx', '.zip', '.jpg', '.jpeg', '.png']}
                acceptedLabels={['PDF', 'Word', 'ZIP', 'Imágenes']}
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
        title="Confirmar entrega"
        message={`¿Estás seguro de que deseas entregar "${archivo?.name}"?${isLate ? ' Esta entrega quedará marcada como tardía.' : ''} No podrás reemplazar el archivo.`}
        confirmLabel="Sí, entregar"
        onConfirm={handleEntregar}
        onCancel={() => setShowModal(false)}
      />
    </div>
  )
}
