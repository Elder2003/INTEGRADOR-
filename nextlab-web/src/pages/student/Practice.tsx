import { useState } from 'react'
import { BrainCircuit, ChevronDown, ChevronUp, CheckCircle, MessageSquare } from 'lucide-react'
import Topbar from '../../components/student/Topbar'
import { examen, feedbackIA } from '../../mocks/examen'
import type { FeedbackIA } from '../../types'

interface FeedbackEntry extends FeedbackIA {
  visible: boolean
}

export default function Practice() {
  const [preguntaIdx, setPreguntaIdx] = useState(0)
  const [respuestas, setRespuestas] = useState<Record<string, string>>({})
  const [feedbacks, setFeedbacks] = useState<FeedbackEntry[]>(
    feedbackIA.map(f => ({ ...f, visible: false }))
  )
  const [showReport, setShowReport] = useState(false)

  const pregunta = examen.preguntas[preguntaIdx]

  function pedirAyuda() {
    setFeedbacks(prev =>
      prev.map(f =>
        f.pregunta === pregunta.numero ? { ...f, visible: true } : f
      )
    )
  }

  const feedbackActual = feedbacks.find(f => f.pregunta === pregunta.numero && f.visible)
  const respondidas = examen.preguntas.filter(p => (respuestas[p.id] ?? '').trim()).length

  return (
    <div
      style={{
        minHeight: '100vh',
        backgroundColor: 'var(--color-bg)',
        display: 'flex',
        flexDirection: 'column',
      }}
    >
      <Topbar title={examen.titulo} subtitle={`${examen.materia} — Modo práctica`} />

      <main
        style={{
          flex: 1,
          maxWidth: 860,
          margin: '0 auto',
          padding: '28px 24px',
          width: '100%',
          borderLeft: '4px solid var(--color-stripe)',
        }}
      >
        {/* Question nav */}
        <div style={{ display: 'flex', gap: 8, marginBottom: 22 }}>
          {examen.preguntas.map((p, i) => {
            const respondida = (respuestas[p.id] ?? '').trim().length > 0
            return (
              <button
                key={p.id}
                onClick={() => setPreguntaIdx(i)}
                style={{
                  width: 36,
                  height: 36,
                  borderRadius: '50%',
                  border: '1px solid',
                  borderColor:
                    i === preguntaIdx
                      ? 'var(--color-accent)'
                      : respondida
                      ? 'var(--color-success)'
                      : 'var(--color-border)',
                  backgroundColor:
                    i === preguntaIdx ? 'var(--color-accent)' : 'transparent',
                  color:
                    i === preguntaIdx
                      ? '#fff'
                      : respondida
                      ? 'var(--color-success)'
                      : 'var(--color-text-secondary)',
                  fontWeight: 600,
                  fontSize: 'var(--font-size-sm)',
                }}
              >
                {i + 1}
              </button>
            )
          })}
        </div>

        {/* Question */}
        <div
          style={{
            backgroundColor: 'var(--color-surface)',
            border: '1px solid var(--color-border)',
            borderRadius: 8,
            padding: 20,
            marginBottom: 16,
          }}
        >
          <div
            style={{
              display: 'flex',
              justifyContent: 'space-between',
              alignItems: 'baseline',
              marginBottom: 10,
            }}
          >
            <span
              style={{
                fontWeight: 600,
                fontSize: 'var(--font-size-md)',
                color: 'var(--color-text-primary)',
              }}
            >
              Pregunta {pregunta.numero}
            </span>
            <span
              style={{
                fontSize: 'var(--font-size-sm)',
                color: 'var(--color-text-muted)',
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
            onChange={e =>
              setRespuestas(r => ({ ...r, [pregunta.id]: e.target.value }))
            }
            placeholder={pregunta.placeholder}
            maxLength={pregunta.maxCaracteres}
            style={{
              width: '100%',
              minHeight: pregunta.modalidad === 'codigo' ? 260 : 140,
              padding: '10px 12px',
              backgroundColor: 'var(--color-surface-2)',
              border: '1px solid var(--color-border)',
              borderRadius: 6,
              color: 'var(--color-text-primary)',
              fontFamily:
                pregunta.modalidad === 'codigo' ? 'var(--font-mono)' : 'var(--font-sans)',
              fontSize:
                pregunta.modalidad === 'codigo'
                  ? 'var(--font-size-sm)'
                  : 'var(--font-size-base)',
              resize: 'vertical',
              outline: 'none',
              lineHeight: 1.65,
            }}
            onFocus={e => (e.target.style.borderColor = 'var(--color-accent)')}
            onBlur={e => (e.target.style.borderColor = 'var(--color-border)')}
          />
        </div>

        {/* AI help button */}
        <div style={{ display: 'flex', gap: 10, marginBottom: 16 }}>
          <button className="btn-secondary" onClick={pedirAyuda} style={{ gap: 6 }}>
            <BrainCircuit size={14} />
            Pedir ayuda a la IA
          </button>
          <button
            className="btn-secondary"
            onClick={() => setPreguntaIdx(i => Math.max(0, i - 1))}
            disabled={preguntaIdx === 0}
          >
            ← Anterior
          </button>
          <button
            className="btn-secondary"
            onClick={() =>
              setPreguntaIdx(i => Math.min(examen.preguntas.length - 1, i + 1))
            }
            disabled={preguntaIdx === examen.preguntas.length - 1}
          >
            Siguiente →
          </button>
        </div>

        {/* AI feedback for current question */}
        {feedbackActual && (
          <div
            style={{
              backgroundColor: 'rgba(74,158,218,0.08)',
              border: '1px solid rgba(74,158,218,0.3)',
              borderRadius: 8,
              padding: '14px 16px',
              marginBottom: 16,
              display: 'flex',
              gap: 10,
            }}
          >
            <MessageSquare size={16} color="var(--color-info)" style={{ flexShrink: 0, marginTop: 2 }} />
            <div>
              <span
                style={{
                  fontSize: 'var(--font-size-xs)',
                  color: 'var(--color-info)',
                  fontWeight: 600,
                  textTransform: 'uppercase',
                  letterSpacing: '0.05em',
                }}
              >
                {feedbackActual.tipo}
              </span>
              <p
                style={{
                  marginTop: 4,
                  fontSize: 'var(--font-size-sm)',
                  color: 'var(--color-text-secondary)',
                  lineHeight: 1.6,
                }}
              >
                {feedbackActual.feedback}
              </p>
            </div>
          </div>
        )}

        {/* Report button */}
        <div style={{ borderTop: '1px solid var(--color-border)', paddingTop: 18 }}>
          <button
            className="btn-primary"
            onClick={() => setShowReport(s => !s)}
            style={{ gap: 8 }}
          >
            {showReport ? <ChevronUp size={14} /> : <ChevronDown size={14} />}
            Ver reporte de la sesión
          </button>

          {showReport && (
            <div
              style={{
                marginTop: 16,
                backgroundColor: 'var(--color-surface)',
                border: '1px solid var(--color-border)',
                borderRadius: 8,
                padding: 18,
              }}
            >
              <h3
                style={{
                  fontWeight: 600,
                  marginBottom: 14,
                  fontSize: 'var(--font-size-md)',
                }}
              >
                Reporte de práctica
              </h3>
              <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
                {examen.preguntas.map(p => {
                  const r = respuestas[p.id] ?? ''
                  const fb = feedbacks.find(f => f.pregunta === p.numero && f.visible)
                  return (
                    <div
                      key={p.id}
                      style={{
                        backgroundColor: 'var(--color-surface-2)',
                        borderRadius: 6,
                        padding: '12px 14px',
                      }}
                    >
                      <div
                        style={{
                          display: 'flex',
                          justifyContent: 'space-between',
                          alignItems: 'center',
                          marginBottom: 6,
                        }}
                      >
                        <span style={{ fontWeight: 500 }}>Pregunta {p.numero}</span>
                        {r.trim() ? (
                          <CheckCircle size={14} color="var(--color-success)" />
                        ) : (
                          <span
                            style={{
                              fontSize: 'var(--font-size-xs)',
                              color: 'var(--color-text-muted)',
                            }}
                          >
                            Sin respuesta
                          </span>
                        )}
                      </div>
                      {fb && (
                        <p
                          style={{
                            fontSize: 'var(--font-size-sm)',
                            color: 'var(--color-info)',
                            lineHeight: 1.5,
                          }}
                        >
                          {fb.feedback}
                        </p>
                      )}
                    </div>
                  )
                })}
              </div>
              <div
                style={{
                  marginTop: 14,
                  fontSize: 'var(--font-size-sm)',
                  color: 'var(--color-text-secondary)',
                }}
              >
                Preguntas respondidas: {respondidas}/{examen.preguntas.length}
              </div>
            </div>
          )}
        </div>
      </main>
    </div>
  )
}
