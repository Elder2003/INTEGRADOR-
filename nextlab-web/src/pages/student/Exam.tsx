import { useState, useEffect, useRef } from 'react'
import Topbar from '../../components/student/Topbar'
import TimerDisplay from '../../components/student/TimerDisplay'
import TracingPanel from '../../components/student/TracingPanel'
import ConfirmModal from '../../components/student/ConfirmModal'
import IntegrityDeclaration from '../../components/student/IntegrityDeclaration'
import { examen } from '../../mocks/examen'
import { CheckCircle } from 'lucide-react'

export default function Exam() {
  const [preguntaIdx, setPreguntaIdx] = useState(0)
  const [respuestas, setRespuestas] = useState<Record<string, string>>({})
  const [declaracion, setDeclaracion] = useState(false)
  const [showModal, setShowModal] = useState(false)
  const [enviado, setEnviado] = useState(false)

  const [wpm, setWpm] = useState(0)
  const [activeTime, setActiveTime] = useState(0)
  const [pasteCount, setPasteCount] = useState(0)
  const [corrections, setCorrections] = useState(0)
  const [activityHistory, setActivityHistory] = useState<number[]>(Array(10).fill(0))
  const keystrokesRef = useRef<number[]>([])

  const [fechaLimite] = useState(
    () => new Date(Date.now() + examen.duracionMinutos * 60 * 1000).toISOString()
  )

  useEffect(() => {
    const t = setInterval(() => setActiveTime(s => s + 1), 1000)
    return () => clearInterval(t)
  }, [])

  useEffect(() => {
    const t = setInterval(() => {
      const now = Date.now()
      const recent = keystrokesRef.current.filter(ts => now - ts < 60_000)
      const currentWpm = Math.round(recent.length / 5)
      setWpm(currentWpm)
      setActivityHistory(prev => [...prev.slice(1), currentWpm])
      keystrokesRef.current = recent
    }, 3000)
    return () => clearInterval(t)
  }, [])

  const pregunta = examen.preguntas[preguntaIdx]
  const totalPalabras = Object.values(respuestas).reduce(
    (acc, r) => acc + r.split(/\s+/).filter(Boolean).length,
    0
  )
  const preguntasRespondidas = examen.preguntas.filter(
    p => (respuestas[p.id] ?? '').trim().length > 0
  ).length

  function handleChange(pregId: string, value: string) {
    const prev = respuestas[pregId] ?? ''
    if (prev.length > value.length) setCorrections(c => c + 1)
    setRespuestas(r => ({ ...r, [pregId]: value }))
  }

  function handleEnviar() {
    setEnviado(true)
    setShowModal(false)
  }

  if (enviado) {
    return (
      <div
        style={{
          minHeight: '100vh',
          backgroundColor: 'var(--color-bg)',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
        }}
      >
        <div style={{ textAlign: 'center', maxWidth: 400, padding: 24 }}>
          <CheckCircle size={52} color="var(--color-success)" style={{ marginBottom: 16 }} />
          <h2
            style={{
              color: 'var(--color-success)',
              fontSize: 'var(--font-size-xl)',
              marginBottom: 10,
            }}
          >
            Examen enviado
          </h2>
          <p style={{ color: 'var(--color-text-secondary)', lineHeight: 1.7 }}>
            Tu examen ha sido entregado exitosamente. El docente revisará tu proceso de resolución
            registrado por el sistema de auditoría.
          </p>
        </div>
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
      <Topbar
        title={examen.titulo}
        subtitle={examen.materia}
        rightSlot={
          <TimerDisplay
            fechaLimite={fechaLimite}
            compact
            onExpire={() => setShowModal(true)}
            warningThresholdMinutes={10}
            dangerThresholdMinutes={5}
          />
        }
      />

      <div style={{ flex: 1, display: 'flex', overflow: 'hidden' }}>
        {/* Left: question area */}
        <main
          style={{
            flex: 1,
            padding: 24,
            overflowY: 'auto',
            borderLeft: '4px solid var(--color-stripe)',
          }}
        >
          {/* Question pills */}
          <div style={{ display: 'flex', gap: 8, marginBottom: 22, flexWrap: 'wrap' }}>
            {examen.preguntas.map((p, i) => {
              const respondida = (respuestas[p.id] ?? '').trim().length > 0
              const activa = i === preguntaIdx
              return (
                <button
                  key={p.id}
                  onClick={() => setPreguntaIdx(i)}
                  style={{
                    width: 36,
                    height: 36,
                    borderRadius: '50%',
                    border: '1px solid',
                    borderColor: activa
                      ? 'var(--color-accent)'
                      : respondida
                      ? 'var(--color-success)'
                      : 'var(--color-border)',
                    backgroundColor: activa ? 'var(--color-accent)' : 'transparent',
                    color: activa
                      ? '#fff'
                      : respondida
                      ? 'var(--color-success)'
                      : 'var(--color-text-secondary)',
                    fontWeight: 600,
                    fontSize: 'var(--font-size-sm)',
                    cursor: 'pointer',
                    transition: 'all 0.15s',
                  }}
                >
                  {i + 1}
                </button>
              )
            })}
          </div>

          {/* Question */}
          <div style={{ marginBottom: 20 }}>
            <div
              style={{
                display: 'flex',
                justifyContent: 'space-between',
                alignItems: 'baseline',
                marginBottom: 10,
              }}
            >
              <h3 style={{ fontSize: 'var(--font-size-md)', fontWeight: 600 }}>
                Pregunta {pregunta.numero}
              </h3>
              <span
                style={{
                  fontSize: 'var(--font-size-sm)',
                  color: 'var(--color-text-secondary)',
                  fontFamily: 'var(--font-mono)',
                }}
              >
                {pregunta.puntos} pts
              </span>
            </div>
            <p
              style={{
                color: 'var(--color-text-secondary)',
                lineHeight: 1.7,
                marginBottom: 14,
              }}
            >
              {pregunta.enunciado}
            </p>
            <textarea
              value={respuestas[pregunta.id] ?? ''}
              onChange={e => handleChange(pregunta.id, e.target.value)}
              onKeyDown={() => keystrokesRef.current.push(Date.now())}
              onPaste={() => setPasteCount(c => c + 1)}
              placeholder={pregunta.placeholder}
              maxLength={pregunta.maxCaracteres}
              style={{
                width: '100%',
                minHeight: pregunta.modalidad === 'codigo' ? 280 : 160,
                padding: '12px 14px',
                backgroundColor: 'var(--color-surface)',
                border: '1px solid var(--color-border)',
                borderRadius: 6,
                color: 'var(--color-text-primary)',
                fontFamily:
                  pregunta.modalidad === 'codigo' ? 'var(--font-mono)' : 'var(--font-sans)',
                fontSize:
                  pregunta.modalidad === 'codigo' ? 'var(--font-size-sm)' : 'var(--font-size-base)',
                resize: 'vertical',
                outline: 'none',
                lineHeight: 1.65,
                transition: 'border-color 0.2s',
              }}
              onFocus={e => (e.target.style.borderColor = 'var(--color-accent)')}
              onBlur={e => (e.target.style.borderColor = 'var(--color-border)')}
            />
            <div
              style={{
                textAlign: 'right',
                fontSize: 'var(--font-size-xs)',
                color: 'var(--color-text-muted)',
                marginTop: 4,
              }}
            >
              {(respuestas[pregunta.id] ?? '').length}/{pregunta.maxCaracteres}
            </div>
          </div>

          {/* Nav buttons */}
          <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 28 }}>
            <button
              className="btn-secondary"
              onClick={() => setPreguntaIdx(i => Math.max(0, i - 1))}
              disabled={preguntaIdx === 0}
            >
              ← Anterior
            </button>
            <button
              className="btn-secondary"
              onClick={() => setPreguntaIdx(i => Math.min(examen.preguntas.length - 1, i + 1))}
              disabled={preguntaIdx === examen.preguntas.length - 1}
            >
              Siguiente →
            </button>
          </div>

          {/* Integrity + submit */}
          <div
            style={{ borderTop: '1px solid var(--color-border)', paddingTop: 20 }}
          >
            <IntegrityDeclaration checked={declaracion} onChange={setDeclaracion} />
            <button
              className="btn-primary"
              onClick={() => setShowModal(true)}
              disabled={!declaracion}
              style={{ width: '100%', padding: '12px 0', marginTop: 14 }}
            >
              Enviar examen
            </button>
          </div>
        </main>

        {/* Right: tracing panel */}
        <aside
          style={{
            width: 280,
            padding: '20px 16px',
            borderLeft: '1px solid var(--color-border)',
            overflowY: 'auto',
            flexShrink: 0,
            display: 'flex',
            flexDirection: 'column',
            gap: 16,
          }}
        >
          <TimerDisplay
            fechaLimite={fechaLimite}
            onExpire={() => setShowModal(true)}
            warningThresholdMinutes={10}
            dangerThresholdMinutes={5}
          />
          <TracingPanel
            data={{ wpm, activeTime, pasteCount, corrections, activityHistory }}
            totalPreguntas={examen.preguntas.length}
            preguntasRespondidas={preguntasRespondidas}
            totalPalabras={totalPalabras}
          />
        </aside>
      </div>

      <ConfirmModal
        open={showModal}
        title="Enviar examen"
        message={`¿Estás seguro de que deseas enviar el examen? Has respondido ${preguntasRespondidas} de ${examen.preguntas.length} preguntas. Esta acción no se puede deshacer.`}
        confirmLabel="Sí, enviar"
        onConfirm={handleEnviar}
        onCancel={() => setShowModal(false)}
      />
    </div>
  )
}
