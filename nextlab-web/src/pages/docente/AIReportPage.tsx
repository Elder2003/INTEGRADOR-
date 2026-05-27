import { useEffect, useState } from 'react'
import { Link, useLocation, useParams } from 'react-router-dom'
import { getAIReport } from '../../api/gitAnalysis'
import type { AIReport } from '../../types/academic'
import { LoadingSpinner } from '../../components/common/LoadingSpinner'
import { ErrorMessage } from '../../components/common/ErrorMessage'

export function AIReportPage() {
  const { cid, aid, subid } = useParams<{ cid: string; aid: string; subid: string }>()
  const location = useLocation()
  const routePrefix = location.pathname.startsWith('/docente') ? '/docente' : '/codocente'
  const backPath = `${routePrefix}/cursos/${cid}/actividades/${aid}/submissions/${subid}`

  const [data, setData] = useState<AIReport | null>(null)
  const [loading, setLoading] = useState(true)
  const [fetchError, setFetchError] = useState<string | null>(null)

  useEffect(() => {
    if (!subid) return
    getAIReport(subid)
      .then(setData)
      .catch((err: Error) => setFetchError(err.message))
      .finally(() => setLoading(false))
  }, [subid])

  if (loading) return <LoadingSpinner />

  return (
    <div className="p-8 max-w-3xl space-y-6">
      <div>
        <Link to={backPath} className="text-sm text-[#888888] hover:text-[#E85D24] transition-colors">
          ← Entrega
        </Link>
        <h1 className="text-2xl font-semibold text-[#f0f0f0] mt-4">Informe IA</h1>
      </div>

      {fetchError && <ErrorMessage message={fetchError} />}

      {!fetchError && data && data.status === 'pending' && (
        <div className="bg-[#1a1a1a] border border-[#2e2e2e] rounded-lg p-4">
          <p className="text-[#888888] text-sm">Análisis IA en curso. Vuelve a revisar en unos minutos.</p>
        </div>
      )}

      {!fetchError && data && data.status === 'error' && (
        <div className="bg-red-900/20 border border-red-800 rounded-lg p-4">
          <p className="text-red-400 text-sm">{data.error_message ?? 'El análisis IA encontró un error.'}</p>
        </div>
      )}

      {!fetchError && data && data.status === 'done' && (
        <div className="space-y-4">
          {data.sections && data.sections.length > 0 ? (
            data.sections.map((section, idx) => (
              <div key={idx} className="bg-[#1a1a1a] border border-[#2e2e2e] rounded-lg p-4 space-y-2">
                <h2 className="text-sm font-semibold text-[#E85D24] uppercase tracking-wide">
                  {section.title}
                </h2>
                <pre className="whitespace-pre-wrap text-[#f0f0f0] text-sm leading-relaxed font-sans">
                  {section.content}
                </pre>
              </div>
            ))
          ) : (
            <p className="text-[#888888] text-sm">El informe IA no contiene secciones.</p>
          )}
        </div>
      )}
    </div>
  )
}
