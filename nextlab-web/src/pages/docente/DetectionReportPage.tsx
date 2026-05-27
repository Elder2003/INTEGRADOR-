import { useEffect, useState } from 'react'
import { Link, useLocation, useParams } from 'react-router-dom'
import { getDetectionReport } from '../../api/gitAnalysis'
import type { DetectionReport } from '../../types/academic'
import { LoadingSpinner } from '../../components/common/LoadingSpinner'
import { ErrorMessage } from '../../components/common/ErrorMessage'

function scoreColorClass(score: number): string {
  if (score <= 20) return 'text-green-400'
  if (score <= 50) return 'text-yellow-400'
  return 'text-red-400'
}

export function DetectionReportPage() {
  const { cid, aid, subid } = useParams<{ cid: string; aid: string; subid: string }>()
  const location = useLocation()
  const routePrefix = location.pathname.startsWith('/docente') ? '/docente' : '/codocente'
  const backPath = `${routePrefix}/cursos/${cid}/actividades/${aid}/submissions/${subid}`

  const [data, setData] = useState<DetectionReport | null>(null)
  const [loading, setLoading] = useState(true)
  const [fetchError, setFetchError] = useState<string | null>(null)

  useEffect(() => {
    if (!subid) return
    getDetectionReport(subid)
      .then(setData)
      .catch((err: Error) => setFetchError(err.message))
      .finally(() => setLoading(false))
  }, [subid])

  if (loading) return <LoadingSpinner />

  return (
    <div className="p-8 max-w-4xl space-y-6">
      <div>
        <Link to={backPath} className="text-sm text-[#888888] hover:text-[#E85D24] transition-colors">
          ← Entrega
        </Link>
        <h1 className="text-2xl font-semibold text-[#f0f0f0] mt-4">Informe de Detección</h1>
      </div>

      {fetchError && <ErrorMessage message={fetchError} />}

      {!fetchError && data && data.status === 'pending' && (
        <div className="bg-[#1a1a1a] border border-[#2e2e2e] rounded-lg p-4">
          <p className="text-[#888888] text-sm">Detección en curso. Vuelve a revisar en unos minutos.</p>
        </div>
      )}

      {!fetchError && data && data.status === 'error' && (
        <div className="bg-red-900/20 border border-red-800 rounded-lg p-4">
          <p className="text-red-400 text-sm">{data.error_message ?? 'La detección encontró un error.'}</p>
        </div>
      )}

      {!fetchError && data && data.status === 'done' && (
        <div className="space-y-6">
          {data.similarity_score !== undefined && (
            <div className="bg-[#1a1a1a] border border-[#2e2e2e] rounded-lg p-6 flex items-center gap-4">
              <span className={`text-5xl font-bold ${scoreColorClass(data.similarity_score)}`}>
                {data.similarity_score}%
              </span>
              <div>
                <p className="text-[#f0f0f0] font-medium">Similitud global</p>
                <p className="text-[#888888] text-xs mt-0.5">
                  {data.similarity_score <= 20 ? 'Nivel bajo' : data.similarity_score <= 50 ? 'Nivel medio' : 'Nivel alto'}
                </p>
              </div>
            </div>
          )}

          {data.flagged_segments && data.flagged_segments.length > 0 ? (
            <div className="bg-[#1a1a1a] border border-[#2e2e2e] rounded-xl overflow-hidden">
              <p className="px-4 py-3 text-xs text-[#888888] uppercase tracking-wide border-b border-[#2e2e2e]">
                Segmentos detectados ({data.flagged_segments.length})
              </p>
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-[#2e2e2e] text-left">
                    <th className="px-4 py-2 text-[#888888] font-medium">Fuente</th>
                    <th className="px-4 py-2 text-[#888888] font-medium">Coincidencia</th>
                    {data.flagged_segments.some(s => s.content) && (
                      <th className="px-4 py-2 text-[#888888] font-medium">Fragmento</th>
                    )}
                  </tr>
                </thead>
                <tbody>
                  {data.flagged_segments.map((seg, idx) => (
                    <tr key={idx} className="border-b border-[#2e2e2e] last:border-0">
                      <td className="px-4 py-3 text-[#f0f0f0] text-xs font-mono break-all max-w-[200px]">
                        {seg.source}
                      </td>
                      <td className={`px-4 py-3 font-medium ${scoreColorClass(seg.match_percentage)}`}>
                        {seg.match_percentage}%
                      </td>
                      {data.flagged_segments!.some(s => s.content) && (
                        <td className="px-4 py-3 text-[#888888] text-xs max-w-[300px]">
                          {seg.content ?? '—'}
                        </td>
                      )}
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : (
            <p className="text-[#888888] text-sm">Sin segmentos detectados.</p>
          )}
        </div>
      )}
    </div>
  )
}
