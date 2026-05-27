import { useState, useEffect } from 'react'
import { Link, useLocation, useParams } from 'react-router-dom'
import { getActivityAnalytics } from '../../api/analytics'
import type { ActivityAnalytics } from '../../types/academic'
import { ApiError } from '../../types'
import { LoadingSpinner } from '../../components/common/LoadingSpinner'
import { ErrorMessage } from '../../components/common/ErrorMessage'

const STAGE_LABELS: Record<string, string> = {
  propuesta: 'Propuesta',
  desarrollo: 'Desarrollo',
  revisión: 'Revisión',
  aprobado: 'Aprobado',
}

export function ActivityAnalyticsPage() {
  const { cid, aid } = useParams<{ cid: string; aid: string }>()
  const location = useLocation()
  const routePrefix = location.pathname.startsWith('/codocente') ? '/codocente' : '/docente'

  const [data, setData] = useState<ActivityAnalytics | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!aid) return
    getActivityAnalytics(aid)
      .then(setData)
      .catch(err => setError((err as ApiError).message))
      .finally(() => setLoading(false))
  }, [aid])

  if (loading) return <LoadingSpinner />
  if (error) return <ErrorMessage message={error} />
  if (!data) return null

  return (
    <div className="p-8 max-w-2xl">
      <Link
        to={`${routePrefix}/cursos/${cid}/actividades/${aid}`}
        className="text-sm text-[#888888] hover:text-[#f0f0f0] transition-colors mb-4 inline-block"
      >
        ← Actividad
      </Link>

      <h1 className="text-2xl font-semibold text-[#f0f0f0] mb-6">Analíticas de Actividad</h1>

      {/* Stats row */}
      <div className="flex gap-4 mb-6">
        <div className="bg-[#1a1a1a] border border-[#2e2e2e] rounded-xl p-4 flex-1">
          <p className="text-xs text-[#888888] uppercase tracking-wide mb-1">Total Entregas</p>
          <p className="text-2xl font-semibold text-[#f0f0f0]">{data.total_submissions}</p>
        </div>
        <div className="bg-[#1a1a1a] border border-[#2e2e2e] rounded-xl p-4 flex-1">
          <p className="text-xs text-[#888888] uppercase tracking-wide mb-1">Entregas Tardías</p>
          <p className="text-2xl font-semibold text-[#f0f0f0]">{data.late_submissions}</p>
        </div>
      </div>

      {/* Grade distribution */}
      <div className="bg-[#1a1a1a] border border-[#2e2e2e] rounded-xl p-5 mb-4">
        <h2 className="text-[#f0f0f0] font-medium mb-4">Distribución de Calificaciones</h2>
        {data.grade_distribution.length === 0 ? (
          <p className="text-[#888888] text-sm">Sin datos de calificaciones aún.</p>
        ) : (
          <table className="w-full text-sm">
            <thead>
              <tr className="text-[#888888] border-b border-[#2e2e2e] text-left">
                <th className="py-2 pr-4 font-medium">Rango</th>
                <th className="py-2 font-medium">Cantidad</th>
              </tr>
            </thead>
            <tbody>
              {data.grade_distribution.map(bucket => (
                <tr key={bucket.range_label} className="border-b border-[#2e2e2e]">
                  <td className="py-2.5 pr-4 text-[#f0f0f0]">{bucket.range_label}</td>
                  <td className="py-2.5 text-[#f0f0f0]">{bucket.count}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      {/* Stage breakdown (informe only) */}
      {data.stage_breakdown !== null && (
        <div className="bg-[#1a1a1a] border border-[#2e2e2e] rounded-xl p-5">
          <h2 className="text-[#f0f0f0] font-medium mb-4">Etapas de Investigación</h2>
          <table className="w-full text-sm">
            <thead>
              <tr className="text-[#888888] border-b border-[#2e2e2e] text-left">
                <th className="py-2 pr-4 font-medium">Etapa</th>
                <th className="py-2 font-medium">Cantidad</th>
              </tr>
            </thead>
            <tbody>
              {(['propuesta', 'desarrollo', 'revisión', 'aprobado'] as const).map(stage => (
                <tr key={stage} className="border-b border-[#2e2e2e]">
                  <td className="py-2.5 pr-4 text-[#f0f0f0]">{STAGE_LABELS[stage]}</td>
                  <td className="py-2.5 text-[#f0f0f0]">{data.stage_breakdown![stage]}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}
