import { useState, useEffect } from 'react'
import { Link, useLocation, useParams } from 'react-router-dom'
import { getActivityAnalytics } from '../../api/analytics'
import type { ActivityAnalytics } from '../../types/academic'
import { ApiError } from '../../types'
import { LoadingSpinner } from '../../components/common/LoadingSpinner'
import { ErrorMessage } from '../../components/common/ErrorMessage'

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

  const gradeBuckets = [
    { label: '0 – 49', count: data.grade_distribution.range_0_49 },
    { label: '50 – 69', count: data.grade_distribution.range_50_69 },
    { label: '70 – 89', count: data.grade_distribution.range_70_89 },
    { label: '90 – 100', count: data.grade_distribution.range_90_100 },
  ]

  const hasGrades = gradeBuckets.some(b => b.count > 0)

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
          <p className="text-2xl font-semibold text-[#f0f0f0]">{data.submission_count}</p>
        </div>
        <div className="bg-[#1a1a1a] border border-[#2e2e2e] rounded-xl p-4 flex-1">
          <p className="text-xs text-[#888888] uppercase tracking-wide mb-1">Entregas Tardías</p>
          <p className="text-2xl font-semibold text-[#f0f0f0]">{data.late_count}</p>
        </div>
        <div className="bg-[#1a1a1a] border border-[#2e2e2e] rounded-xl p-4 flex-1">
          <p className="text-xs text-[#888888] uppercase tracking-wide mb-1">Marcadas</p>
          <p className="text-2xl font-semibold text-[#f0f0f0]">{data.flagged_count}</p>
        </div>
        {data.avg_grade !== null && (
          <div className="bg-[#1a1a1a] border border-[#2e2e2e] rounded-xl p-4 flex-1">
            <p className="text-xs text-[#888888] uppercase tracking-wide mb-1">Nota Promedio</p>
            <p className="text-2xl font-semibold text-[#f0f0f0]">{data.avg_grade.toFixed(1)}</p>
          </div>
        )}
      </div>

      {/* Grade distribution */}
      <div className="bg-[#1a1a1a] border border-[#2e2e2e] rounded-xl p-5">
        <h2 className="text-[#f0f0f0] font-medium mb-4">Distribución de Calificaciones</h2>
        {!hasGrades ? (
          <p className="text-[#888888] text-sm">Sin calificaciones registradas aún.</p>
        ) : (
          <table className="w-full text-sm">
            <thead>
              <tr className="text-[#888888] border-b border-[#2e2e2e] text-left">
                <th className="py-2 pr-4 font-medium">Rango</th>
                <th className="py-2 font-medium">Cantidad</th>
              </tr>
            </thead>
            <tbody>
              {gradeBuckets.map(bucket => (
                <tr key={bucket.label} className="border-b border-[#2e2e2e] last:border-0">
                  <td className="py-2.5 pr-4 text-[#f0f0f0]">{bucket.label}</td>
                  <td className="py-2.5 text-[#f0f0f0]">{bucket.count}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  )
}
