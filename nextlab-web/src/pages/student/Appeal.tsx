import { useState } from 'react'
import { useParams } from 'react-router-dom'
import { CheckCircle, AlertCircle } from 'lucide-react'
import Topbar from '../../components/student/Topbar'
import ConfirmModal from '../../components/student/ConfirmModal'
import { calificaciones } from '../../mocks/calificaciones'

const MIN_CHARS = 100

export default function Appeal() {
  const { id } = useParams()
  const cal = calificaciones.find(c => c.id === id) ?? calificaciones[0]

  const [justificacion, setJustificacion] = useState('')
  const [showModal, setShowModal] = useState(false)
  const [enviado, setEnviado] = useState(false)

  function handleEnviar() {
    setEnviado(true)
    setShowModal(false)
  }

  const charsOk = justificacion.trim().length >= MIN_CHARS

  if (enviado) {
    return (
      <div
        style={{
          minHeight: '100vh',
          backgroundColor: 'var(--color-bg)',
          display: 'flex',
          flexDirection: 'column',
        }}
      >
        <Topbar title="Apelación enviada" />
        <main
          style={{
            flex: 1,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            padding: 24,
          }}
        >
          <div style={{ textAlign: 'center', maxWidth: 420 }}>
            <CheckCircle size={52} color="var(--color-success)" style={{ marginBottom: 16 }} />
            <h2
              style={{
                fontSize: 'var(--font-size-xl)',
                fontWeight: 700,
                color: 'var(--color-success)',
                marginBottom: 10,
              }}
            >
              Apelación enviada
            </h2>
            <p
              style={{
                color: 'var(--color-text-secondary)',
                lineHeight: 1.7,
                marginBottom: 16,
              }}
            >
              Tu apelación está <strong>En revisión</strong>. El docente la recibirá y podrá
              escalarla al administrador institucional si lo considera necesario.
            </p>
            <div
              style={{
                display: 'inline-flex',
                alignItems: 'center',
                gap: 6,
                padding: '6px 16px',
                borderRadius: 20,
                backgroundColor: 'rgba(240,165,0,0.1)',
                border: '1px solid rgba(240,165,0,0.3)',
                color: 'var(--color-warning)',
                fontSize: 'var(--font-size-sm)',
                fontWeight: 500,
              }}
            >
              <AlertCircle size={13} />
              En revisión
            </div>
          </div>
        </main>
      </div>
    )
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
      <Topbar title="Formulario de apelación" />

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
        {/* Grade summary */}
        <div className="card" style={{ marginBottom: 22 }}>
          <h2
            style={{
              fontWeight: 600,
              fontSize: 'var(--font-size-md)',
              marginBottom: 12,
            }}
          >
            Actividad a apelar
          </h2>
          <div
            style={{
              display: 'flex',
              justifyContent: 'space-between',
              alignItems: 'center',
              gap: 16,
            }}
          >
            <div>
              <div
                style={{
                  fontWeight: 500,
                  color: 'var(--color-text-primary)',
                  marginBottom: 4,
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
                {cal.materia} — {cal.fecha}
              </div>
            </div>
            <div style={{ textAlign: 'right' }}>
              <span
                style={{
                  fontFamily: 'var(--font-mono)',
                  fontSize: 28,
                  fontWeight: 800,
                  color:
                    cal.nota >= 51 ? 'var(--color-success)' : 'var(--color-danger)',
                }}
              >
                {cal.nota}
              </span>
              <span
                style={{
                  fontSize: 'var(--font-size-sm)',
                  color: 'var(--color-text-muted)',
                }}
              >
                /{cal.notaMaxima}
              </span>
            </div>
          </div>
        </div>

        {/* Justification */}
        <div style={{ marginBottom: 18 }}>
          <label
            style={{
              display: 'block',
              fontWeight: 600,
              marginBottom: 8,
              fontSize: 'var(--font-size-md)',
            }}
          >
            Justificación de la apelación
          </label>
          <p
            style={{
              fontSize: 'var(--font-size-sm)',
              color: 'var(--color-text-secondary)',
              marginBottom: 10,
              lineHeight: 1.5,
            }}
          >
            Explica detalladamente los motivos por los que consideras que la calificación debe ser
            revisada. Mínimo {MIN_CHARS} caracteres.
          </p>
          <textarea
            value={justificacion}
            onChange={e => setJustificacion(e.target.value)}
            placeholder="Describe aquí los motivos de tu apelación..."
            style={{
              width: '100%',
              minHeight: 160,
              padding: '12px 14px',
              backgroundColor: 'var(--color-surface)',
              border: `1px solid ${charsOk ? 'var(--color-success)' : 'var(--color-border)'}`,
              borderRadius: 6,
              color: 'var(--color-text-primary)',
              resize: 'vertical',
              outline: 'none',
              lineHeight: 1.65,
              transition: 'border-color 0.2s',
            }}
            onFocus={e => {
              if (!charsOk) e.target.style.borderColor = 'var(--color-accent)'
            }}
            onBlur={e => {
              e.target.style.borderColor = charsOk
                ? 'var(--color-success)'
                : 'var(--color-border)'
            }}
          />
          <div
            style={{
              display: 'flex',
              justifyContent: 'space-between',
              marginTop: 4,
              fontSize: 'var(--font-size-xs)',
            }}
          >
            <span style={{ color: charsOk ? 'var(--color-success)' : 'var(--color-text-muted)' }}>
              {justificacion.trim().length}/{MIN_CHARS} caracteres mínimos
            </span>
            <span style={{ color: 'var(--color-text-muted)' }}>
              {justificacion.length} caracteres
            </span>
          </div>
        </div>

        {/* Process explanation */}
        <div
          style={{
            backgroundColor: 'var(--color-surface)',
            border: '1px solid var(--color-border)',
            borderRadius: 8,
            padding: '14px 16px',
            marginBottom: 20,
          }}
        >
          <div
            style={{
              fontWeight: 600,
              marginBottom: 10,
              fontSize: 'var(--font-size-sm)',
              color: 'var(--color-text-secondary)',
            }}
          >
            Proceso de apelación
          </div>
          <ul
            style={{
              display: 'flex',
              flexDirection: 'column',
              gap: 6,
              paddingLeft: 4,
            }}
          >
            {[
              'El docente recibirá tu apelación y podrá escalarla al administrador institucional.',
              'El administrador tendrá acceso a la evidencia completa de tu sesión.',
              'La resolución final quedará registrada en el sistema.',
            ].map(item => (
              <li
                key={item}
                style={{
                  fontSize: 'var(--font-size-sm)',
                  color: 'var(--color-text-secondary)',
                  lineHeight: 1.6,
                  listStyle: 'none',
                  paddingLeft: 14,
                  position: 'relative',
                }}
              >
                <span
                  style={{
                    position: 'absolute',
                    left: 0,
                    color: 'var(--color-accent)',
                  }}
                >
                  ›
                </span>
                {item}
              </li>
            ))}
          </ul>
        </div>

        <button
          className="btn-primary"
          onClick={() => setShowModal(true)}
          disabled={!charsOk}
          style={{ width: '100%', padding: '12px 0' }}
        >
          Enviar apelación
        </button>
      </main>

      <ConfirmModal
        open={showModal}
        title="Confirmar apelación"
        message="¿Estás seguro de que deseas enviar esta apelación? El docente y el administrador institucional tendrán acceso a la evidencia completa de tu sesión."
        confirmLabel="Sí, enviar apelación"
        onConfirm={handleEnviar}
        onCancel={() => setShowModal(false)}
      />
    </div>
  )
}
