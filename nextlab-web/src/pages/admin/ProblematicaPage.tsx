import { useState } from 'react'
import { Link } from 'react-router-dom'
import { solveProblem } from '../../api/rag'
import type { SolveProblemResponse } from '../../api/rag'
import { ErrorMessage } from '../../components/common/ErrorMessage'

export function ProblematicaPage() {
  const [problem, setProblem] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [result, setResult] = useState<SolveProblemResponse | null>(null)

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!problem.trim()) return
    setError(null)
    setResult(null)
    setLoading(true)
    try {
      const res = await solveProblem(problem.trim())
      setResult(res)
    } catch (err: unknown) {
      setError((err as Error).message)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="p-8 max-w-3xl space-y-6">
      <div>
        <Link
          to="/admin/dashboard"
          className="text-sm text-[#888888] hover:text-[#E85D24] transition-colors"
        >
          ← Panel de administración
        </Link>
        <h1 className="text-2xl font-semibold text-[#f0f0f0] mt-4">Resolver Problemáticas</h1>
        <p className="text-sm text-[#888888] mt-1">
          Describe un problema o desafío. El sistema buscará en el currículo qué materias pueden
          aportar soluciones y mostrará una sugerencia generada por IA.
        </p>
      </div>

      <form onSubmit={handleSubmit} className="space-y-3">
        <textarea
          value={problem}
          onChange={(e) => setProblem(e.target.value)}
          placeholder="Ej: Los estudiantes tienen dificultades para entender la complejidad algorítmica y su impacto en el rendimiento de aplicaciones reales..."
          rows={5}
          disabled={loading}
          className="w-full bg-[#1a1a1a] border border-[#2e2e2e] rounded-lg p-3 text-[#f0f0f0] text-sm placeholder-[#888888] focus:outline-none focus:border-[#E85D24] resize-none disabled:opacity-50"
        />
        <button
          type="submit"
          disabled={loading || !problem.trim()}
          className="px-5 py-2 bg-[#E85D24] text-white text-sm font-medium rounded-lg hover:bg-[#d4521f] disabled:opacity-50 transition-colors flex items-center gap-2"
        >
          {loading && (
            <span className="h-4 w-4 rounded-full border-2 border-white border-t-transparent animate-spin" />
          )}
          {loading ? 'Analizando...' : 'Analizar problemática'}
        </button>
      </form>

      {error && <ErrorMessage message={error} onDismiss={() => setError(null)} />}

      {result && (
        <div className="space-y-6">
          {/* AI suggestion */}
          <div className="bg-[#1a1a1a] border border-[#2e2e2e] rounded-xl p-5 space-y-3">
            <h2 className="text-sm font-semibold text-[#E85D24] uppercase tracking-wide">
              Sugerencia
            </h2>
            <p className="text-[#f0f0f0] text-sm leading-relaxed whitespace-pre-wrap">
              {result.suggestion}
            </p>
          </div>

          {/* Materia cards */}
          {result.materias.length > 0 ? (
            <div className="space-y-3">
              <h2 className="text-base font-semibold text-[#f0f0f0]">
                Materias relevantes ({result.materias.length})
              </h2>
              <div className="space-y-3">
                {result.materias.map((m, i) => (
                  <div
                    key={`${m.codigo_materia}-${i}`}
                    className="bg-[#1a1a1a] border border-[#2e2e2e] rounded-lg p-4 space-y-2"
                  >
                    <div className="flex items-center justify-between gap-3">
                      <div className="flex items-center gap-3">
                        <span className="text-[#f0f0f0] font-medium text-sm">{m.codigo_materia}</span>
                        <span className="text-xs text-[#888888]">{m.carrera} · Semestre {m.semestre}</span>
                      </div>
                      <RelevanceBar score={m.relevance_score} />
                    </div>
                    <p className="text-xs text-[#888888] leading-relaxed">{m.excerpt}</p>
                  </div>
                ))}
              </div>
            </div>
          ) : (
            <p className="text-sm text-[#888888]">
              No se encontraron materias con contenido curricular indexado. Sube documentos desde la
              sección de curriculum para activar esta función.
            </p>
          )}
        </div>
      )}
    </div>
  )
}

function RelevanceBar({ score }: { score: number }) {
  const pct = Math.round(score * 100)
  const color =
    pct >= 70 ? 'bg-green-500' : pct >= 45 ? 'bg-yellow-500' : 'bg-[#E85D24]'
  return (
    <div className="flex items-center gap-2 shrink-0">
      <div className="w-20 bg-[#2e2e2e] rounded-full h-1.5">
        <div className={`${color} h-1.5 rounded-full`} style={{ width: `${pct}%` }} />
      </div>
      <span className="text-xs text-[#888888] w-8 text-right">{pct}%</span>
    </div>
  )
}
