import { useState, useEffect, useRef } from 'react'
import { Send, MessageSquare } from 'lucide-react'
import Topbar from '../../components/student/Topbar'
import TracingPanel from '../../components/student/TracingPanel'
import { defensa as defensaMock } from '../../mocks/defensa'

interface HistorialItem {
  pregunta: string
  respuesta: string
  timestamp: string
}

export default function Defense() {
  const [respuesta, setRespuesta] = useState('')
  const [historial, setHistorial] = useState<HistorialItem[]>([])

  const [wpm, setWpm] = useState(0)
  const [activeTime, setActiveTime] = useState(0)
  const [pasteCount, setPasteCount] = useState(0)
  const [corrections, setCorrections] = useState(0)
  const [activityHistory, setActivityHistory] = useState<number[]>(Array(10).fill(0))
  const keystrokesRef = useRef<number[]>([])

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

  function enviarRespuesta() {
    if (!respuesta.trim()) return
    setHistorial(h => [
      ...h,
      {
        pregunta: defensaMock.preguntaActual,
        respuesta,
        timestamp: new Date().toLocaleTimeString('es-BO'),
      },
    ])
    setRespuesta('')
  }

  const totalPalabras = historial.reduce(
    (acc, h) => acc + h.respuesta.split(/\s+/).filter(Boolean).length,
    0
  ) + respuesta.split(/\s+/).filter(Boolean).length

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
        title="Sesión de defensa oral"
        subtitle={`${defensaMock.proyecto} — ${defensaMock.docente}`}
      />

      <div style={{ flex: 1, display: 'flex', overflow: 'hidden' }}>
        {/* Left panel */}
        <main
          style={{
            flex: 1,
            padding: 24,
            overflowY: 'auto',
            borderLeft: '4px solid var(--color-stripe)',
          }}
        >
          {/* Current question */}
          <div
            style={{
              backgroundColor: 'var(--color-surface)',
              border: '1px solid var(--color-border)',
              borderLeft: '3px solid var(--color-accent)',
              borderRadius: 8,
              padding: '16px 18px',
              marginBottom: 20,
            }}
          >
            <div
              style={{
                fontSize: 'var(--font-size-xs)',
                color: 'var(--color-accent)',
                fontWeight: 600,
                letterSpacing: '0.06em',
                marginBottom: 8,
              }}
            >
              PREGUNTA ACTUAL — {defensaMock.docente}
            </div>
            <p
              style={{
                color: 'var(--color-text-primary)',
                lineHeight: 1.7,
                fontSize: 'var(--font-size-md)',
              }}
            >
              {defensaMock.preguntaActual}
            </p>
          </div>

          {/* Answer area */}
          <div style={{ marginBottom: 16 }}>
            <textarea
              value={respuesta}
              onChange={e => {
                const prev = respuesta
                if (prev.length > e.target.value.length) setCorrections(c => c + 1)
                setRespuesta(e.target.value)
              }}
              onKeyDown={() => keystrokesRef.current.push(Date.now())}
              onPaste={() => setPasteCount(c => c + 1)}
              placeholder="Escribe tu respuesta aquí..."
              style={{
                width: '100%',
                minHeight: 200,
                padding: '12px 14px',
                backgroundColor: 'var(--color-surface)',
                border: '1px solid var(--color-border)',
                borderRadius: 6,
                color: 'var(--color-text-primary)',
                resize: 'vertical',
                outline: 'none',
                lineHeight: 1.65,
                fontSize: 'var(--font-size-base)',
              }}
              onFocus={e => (e.target.style.borderColor = 'var(--color-accent)')}
              onBlur={e => (e.target.style.borderColor = 'var(--color-border)')}
            />
          </div>

          <button
            className="btn-primary"
            onClick={enviarRespuesta}
            disabled={!respuesta.trim()}
            style={{ gap: 6, marginBottom: 28 }}
          >
            <Send size={14} />
            Enviar respuesta
          </button>

          {/* History */}
          {historial.length > 0 && (
            <div>
              <h3
                style={{
                  fontWeight: 600,
                  fontSize: 'var(--font-size-md)',
                  marginBottom: 14,
                  color: 'var(--color-text-secondary)',
                }}
              >
                Historial de la sesión
              </h3>
              <div style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
                {historial.map((item, i) => (
                  <div
                    key={i}
                    style={{
                      backgroundColor: 'var(--color-surface)',
                      border: '1px solid var(--color-border)',
                      borderRadius: 8,
                      overflow: 'hidden',
                    }}
                  >
                    <div
                      style={{
                        backgroundColor: 'var(--color-surface-2)',
                        padding: '8px 14px',
                        fontSize: 'var(--font-size-xs)',
                        color: 'var(--color-text-muted)',
                        display: 'flex',
                        justifyContent: 'space-between',
                      }}
                    >
                      <span style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
                        <MessageSquare size={10} />
                        Pregunta {i + 1}
                      </span>
                      <span>{item.timestamp}</span>
                    </div>
                    <div style={{ padding: '10px 14px' }}>
                      <p
                        style={{
                          fontSize: 'var(--font-size-sm)',
                          color: 'var(--color-text-secondary)',
                          marginBottom: 6,
                          fontStyle: 'italic',
                        }}
                      >
                        {item.pregunta}
                      </p>
                      <p
                        style={{
                          fontSize: 'var(--font-size-sm)',
                          color: 'var(--color-text-primary)',
                          lineHeight: 1.6,
                        }}
                      >
                        {item.respuesta}
                      </p>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}
        </main>

        {/* Right: tracing */}
        <aside
          style={{
            width: 280,
            padding: '20px 16px',
            borderLeft: '1px solid var(--color-border)',
            overflowY: 'auto',
            flexShrink: 0,
          }}
        >
          <TracingPanel
            data={{ wpm, activeTime, pasteCount, corrections, activityHistory }}
            totalPreguntas={Math.max(1, historial.length + 1)}
            preguntasRespondidas={historial.length}
            totalPalabras={totalPalabras}
          />
        </aside>
      </div>
    </div>
  )
}
