import { useState } from 'react'
import { useParams } from 'react-router-dom'
import { AlertCircle, CheckCircle, Info, GitBranch } from 'lucide-react'
import Topbar from '../../components/student/Topbar'
import TimerDisplay from '../../components/student/TimerDisplay'
import FileUpload from '../../components/student/FileUpload'
import ConfirmModal from '../../components/student/ConfirmModal'
import { proyecto } from '../../mocks/proyecto'

export default function ProjectStage() {
  const { stageId } = useParams()
  const etapa = proyecto.etapas?.find(e => e.id === stageId) ?? proyecto.etapas?.[1]!

  const [zipFile, setZipFile] = useState<File | null>(null)
  const [gitUrl, setGitUrl] = useState('')
  const [showModal, setShowModal] = useState(false)
  const [entregado, setEntregado] = useState(false)
  const [timestamp, setTimestamp] = useState('')

  const isLate = new Date() > new Date(etapa.fechaLimite)

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
      <Topbar
        title={etapa.nombre}
        subtitle={`${proyecto.titulo} — Etapa`}
      />

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
            <AlertCircle size={15} color="var(--color-warning)" />
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
              marginBottom: 8,
            }}
          >
            {etapa.nombre}
          </h1>
          <TimerDisplay fechaLimite={etapa.fechaLimite} compact />
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
              <div style={{ fontWeight: 600, color: 'var(--color-success)', marginBottom: 6 }}>
                Etapa entregada
              </div>
              {zipFile && (
                <div
                  style={{
                    fontSize: 'var(--font-size-sm)',
                    color: 'var(--color-text-secondary)',
                    marginBottom: 2,
                  }}
                >
                  Archivo: <strong>{zipFile.name}</strong>
                </div>
              )}
              {gitUrl && (
                <div
                  style={{
                    fontSize: 'var(--font-size-sm)',
                    color: 'var(--color-text-secondary)',
                    marginBottom: 2,
                  }}
                >
                  Repositorio: <strong>{gitUrl}</strong>
                </div>
              )}
              <div
                style={{
                  fontSize: 'var(--font-size-sm)',
                  color: 'var(--color-text-secondary)',
                  marginTop: 4,
                }}
              >
                Entregado el {timestamp}
              </div>
            </div>
          </div>
        ) : (
          <>
            {/* Section 1: ZIP */}
            <div className="card" style={{ marginBottom: 16 }}>
              <h3
                style={{
                  fontWeight: 600,
                  marginBottom: 8,
                  color: 'var(--color-text-primary)',
                }}
              >
                1. Entregable formal (.zip)
              </h3>
              <p
                style={{
                  fontSize: 'var(--font-size-sm)',
                  color: 'var(--color-text-secondary)',
                  marginBottom: 14,
                  lineHeight: 1.6,
                }}
              >
                Solo se aceptan archivos .zip.
              </p>
              <FileUpload
                acceptedTypes={['.zip']}
                acceptedLabels={['ZIP']}
                onFileSelect={setZipFile}
                selectedFile={zipFile}
                onClear={() => setZipFile(null)}
              />
              <div
                style={{
                  marginTop: 10,
                  fontSize: 'var(--font-size-xs)',
                  color: 'var(--color-text-muted)',
                  lineHeight: 1.5,
                }}
              >
                ⚠ Una vez subido, este archivo no puede reemplazarse. Si necesitas corregir algo,
                podrás subir una nueva versión que quedará marcada como entrega tardía.
              </div>
            </div>

            {/* Section 2: Git */}
            <div className="card" style={{ marginBottom: 20 }}>
              <h3
                style={{
                  fontWeight: 600,
                  marginBottom: 8,
                  color: 'var(--color-text-primary)',
                  display: 'flex',
                  alignItems: 'center',
                  gap: 8,
                }}
              >
                <GitBranch size={16} />
                2. Repositorio Git (opcional)
              </h3>
              <p
                style={{
                  fontSize: 'var(--font-size-sm)',
                  color: 'var(--color-text-secondary)',
                  marginBottom: 10,
                  lineHeight: 1.6,
                }}
              >
                El sistema analizará el historial de commits de tu repositorio como evidencia del
                proceso de desarrollo.
              </p>
              <input
                type="url"
                placeholder="https://github.com/usuario/repositorio"
                value={gitUrl}
                onChange={e => setGitUrl(e.target.value)}
                style={{
                  width: '100%',
                  padding: '10px 12px',
                  backgroundColor: 'var(--color-surface-2)',
                  border: '1px solid var(--color-border)',
                  borderRadius: 6,
                  color: 'var(--color-text-primary)',
                  outline: 'none',
                  marginBottom: 10,
                }}
                onFocus={e => (e.target.style.borderColor = 'var(--color-accent)')}
                onBlur={e => (e.target.style.borderColor = 'var(--color-border)')}
              />
              <div
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: 6,
                  fontSize: 'var(--font-size-xs)',
                  color: 'var(--color-info)',
                }}
              >
                <Info size={11} />
                Solo se analiza el historial — no se modifica nada en tu repositorio.
              </div>
            </div>

            <button
              className="btn-primary"
              onClick={() => setShowModal(true)}
              disabled={!zipFile}
              style={{ width: '100%', padding: '12px 0' }}
            >
              Confirmar entrega de etapa
            </button>
          </>
        )}
      </main>

      <ConfirmModal
        open={showModal}
        title="Confirmar entrega de etapa"
        message={`¿Confirmas la entrega de la etapa "${etapa.nombre}" con el archivo "${zipFile?.name}"?${isLate ? ' Quedará marcada como tardía.' : ''} Esta acción no se puede deshacer.`}
        confirmLabel="Sí, entregar"
        onConfirm={handleEntregar}
        onCancel={() => setShowModal(false)}
      />
    </div>
  )
}
