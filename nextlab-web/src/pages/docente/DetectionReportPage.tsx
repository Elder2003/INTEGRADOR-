import { useEffect, useState } from 'react'
import { Link, useLocation, useParams } from 'react-router-dom'
import { getSessionDetectionReport } from '../../api/gitAnalysis'
import type { DetectionReport } from '../../types/academic'
import { LoadingSpinner } from '../../components/common/LoadingSpinner'
import { ErrorMessage } from '../../components/common/ErrorMessage'

function scoreColor(score: number): string {
  if (score <= 30) return 'text-green-400'
  if (score <= 60) return 'text-yellow-400'
  return 'text-red-400'
}

function cleanAdvisory(text: string): string {
  // Strip ```json ... ``` fenced block if LLM wrapped its response
  const jsonMatch = text.match(/```(?:json)?\s*(\{[\s\S]*?\})\s*```/)
  if (jsonMatch) {
    try {
      const parsed = JSON.parse(jsonMatch[1])
      if (parsed.advisory_text) return parsed.advisory_text
    } catch { /* ignore */ }
  }
  // Strip leading/trailing backtick fences without JSON parsing
  return text.replace(/^```(?:json)?\s*/i, '').replace(/\s*```\s*$/, '').trim()
}

export function DetectionReportPage() {
  const { cid, aid, subid } = useParams<{ cid: string; aid: string; subid: string }>()
  const location = useLocation()
  const routePrefix = location.pathname.startsWith('/docente') ? '/docente' : '/codocente'
  const backPath = `${routePrefix}/cursos/${cid}/actividades/${aid}/submissions/${subid}`

  const sessionId = (location.state as { sessionId?: string } | null)?.sessionId

  const [data, setData] = useState<DetectionReport | null>(null)
  const [loading, setLoading] = useState(Boolean(sessionId))
  const [fetchError, setFetchError] = useState<string | null>(
    sessionId ? null : 'No se encontró el ID de sesión. Navega desde la página de la entrega.',
  )

  useEffect(() => {
    if (!sessionId) return
    getSessionDetectionReport(sessionId)
      .then(setData)
      .catch((err: Error) => setFetchError(err.message))
      .finally(() => setLoading(false))
  }, [sessionId])

  if (loading) return <LoadingSpinner />

  return (
    <div className="p-8 max-w-4xl space-y-6">
      <div>
        <Link to={backPath} className="text-sm text-[#888888] hover:text-[#E85D24] transition-colors">
          ← Entrega
        </Link>
        <h1 className="text-2xl font-semibold text-[#f0f0f0] mt-4">Informe de Detección e IA</h1>
      </div>

      {fetchError && <ErrorMessage message={fetchError} />}

      {!fetchError && data && (
        <div className="space-y-6">

          {/* Heuristic score */}
          {data.heuristic && (
            <div className="bg-[#1a1a1a] border border-[#2e2e2e] rounded-xl p-5 space-y-4">
              <h2 className="text-[#f0f0f0] font-medium">Análisis de Comportamiento</h2>
              <div className="flex items-center gap-4">
                <span className={`text-5xl font-bold ${scoreColor(data.heuristic.score)}`}>
                  {data.heuristic.score}
                </span>
                <div>
                  <p className="text-[#f0f0f0] text-sm font-medium">Puntaje de sospecha (0–100)</p>
                  <p className="text-[#888888] text-xs mt-0.5">
                    {data.heuristic.event_count} eventos capturados
                  </p>
                </div>
              </div>
              {data.heuristic.flagged_categories.length > 0 ? (
                <div>
                  <p className="text-xs text-[#888888] uppercase tracking-wide mb-2">Indicadores detectados</p>
                  <div className="flex flex-wrap gap-2">
                    {data.heuristic.flagged_categories.map(cat => (
                      <span key={cat} className="bg-red-400/10 text-red-400 text-xs px-2 py-0.5 rounded">
                        {cat}
                      </span>
                    ))}
                  </div>
                </div>
              ) : (
                <p className="text-[#888888] text-xs">Sin indicadores de integridad detectados.</p>
              )}
            </div>
          )}

          {/* LLM advisory */}
          {data.llm_status === 'success' && data.llm ? (
            <div className="bg-[#1a1a1a] border border-[#2e2e2e] rounded-xl p-5 space-y-4">
              <div className="flex items-center justify-between">
                <h2 className="text-[#f0f0f0] font-medium">Análisis de IA (Groq)</h2>
                <span className="text-xs bg-[#E85D24]/20 text-[#E85D24] px-2 py-0.5 rounded">
                  {data.llm.backend_used}
                </span>
              </div>

              {data.llm.suggested_grade_range && data.llm.suggested_grade_range !== 'unknown' && (
                <div className="bg-[#242424] rounded-lg px-4 py-3">
                  <p className="text-xs text-[#888888] uppercase tracking-wide mb-1">Rango sugerido</p>
                  <p className="text-[#f0f0f0] font-semibold text-lg">{data.llm.suggested_grade_range}</p>
                  <p className="text-xs text-[#888888] mt-0.5">Solo orientativo — la nota final la asigna el docente</p>
                </div>
              )}

              <div>
                <p className="text-xs text-[#888888] uppercase tracking-wide mb-2">Análisis advisory</p>
                <pre className="whitespace-pre-wrap text-[#f0f0f0] text-sm leading-relaxed font-sans bg-[#111111] rounded-lg p-4">
                  {cleanAdvisory(data.llm.advisory_text)}
                </pre>
              </div>
            </div>
          ) : data.llm_status === 'failed' ? (
            <div className="bg-red-900/20 border border-red-800 rounded-lg p-4">
              <p className="text-red-400 text-sm">El análisis de IA no se pudo completar.</p>
            </div>
          ) : data.llm_status === 'unavailable' ? (
            <div className="bg-[#1a1a1a] border border-[#2e2e2e] rounded-lg p-4">
              <p className="text-[#888888] text-sm">Análisis de IA en proceso. Vuelve en unos minutos.</p>
            </div>
          ) : null}

          {/* Grade decision */}
          {data.grade_decision && (
            <div className="bg-[#1a1a1a] border border-[#2e2e2e] rounded-xl p-5 space-y-2">
              <h2 className="text-[#f0f0f0] font-medium">Calificación Final</h2>
              <p className="text-3xl font-bold text-[#E85D24]">{data.grade_decision.final_grade}</p>
              {data.grade_decision.override_reason && (
                <p className="text-[#888888] text-sm">
                  Motivo de ajuste: {data.grade_decision.override_reason}
                </p>
              )}
              <p className="text-[#888888] text-xs">
                Decidido el {new Date(data.grade_decision.decided_at).toLocaleString('es-BO')}
              </p>
            </div>
          )}

        </div>
      )}
    </div>
  )
}
