import { useState, useEffect } from 'react'
import { Link, useParams } from 'react-router-dom'
import { getCourseAnalytics } from '../../api/analytics'
import type { CourseAnalytics } from '../../types/academic'
import { ApiError } from '../../types'
import { LoadingSpinner } from '../../components/common/LoadingSpinner'
import { ErrorMessage } from '../../components/common/ErrorMessage'

export function CourseAnalyticsPage() {
  const { cid } = useParams<{ cid: string }>()

  const [data, setData] = useState<CourseAnalytics | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!cid) return
    getCourseAnalytics(cid)
      .then(setData)
      .catch(err => setError((err as ApiError).message))
      .finally(() => setLoading(false))
  }, [cid])

  if (loading) return <LoadingSpinner />
  if (error) return <ErrorMessage message={error} />
  if (!data) return null

  const activities = data.activities ?? []
  const isEmpty = activities.length === 0

  return (
    <div className="p-8 max-w-3xl">
      <Link
        to={`/docente/cursos/${cid}`}
        className="text-sm text-[#888888] hover:text-[#f0f0f0] transition-colors mb-4 inline-block"
      >
        ← Curso
      </Link>

      <h1 className="text-2xl font-semibold text-[#f0f0f0] mb-6">Analíticas del Curso</h1>

      {/* Overall stats */}
      <div className="grid grid-cols-3 gap-4 mb-6">
        <div className="bg-[#1a1a1a] border border-[#2e2e2e] rounded-xl p-4">
          <p className="text-[#888888] text-xs mb-1">Entregas Totales</p>
          <p className="text-2xl font-semibold text-[#f0f0f0]">{data.total_submissions}</p>
        </div>
        <div className="bg-[#1a1a1a] border border-[#2e2e2e] rounded-xl p-4">
          <p className="text-[#888888] text-xs mb-1">Nota Promedio</p>
          <p className="text-2xl font-semibold text-[#f0f0f0]">
            {data.avg_grade !== null ? data.avg_grade.toFixed(1) : '—'}
          </p>
        </div>
        <div className="bg-[#1a1a1a] border border-[#2e2e2e] rounded-xl p-4">
          <p className="text-[#888888] text-xs mb-1">Tasa de Alerta</p>
          <p className="text-2xl font-semibold text-[#f0f0f0]">
            {(data.flag_rate * 100).toFixed(0)}%
          </p>
        </div>
      </div>

      {/* Per-activity summary */}
      <div className="bg-[#1a1a1a] border border-[#2e2e2e] rounded-xl p-5 mb-4">
        <h2 className="text-[#f0f0f0] font-medium mb-4">Resumen por Actividad</h2>
        {isEmpty ? (
          <p className="text-[#888888] text-sm">Sin actividades registradas aún.</p>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="text-[#888888] border-b border-[#2e2e2e] text-left">
                  <th className="py-2 pr-4 font-medium">ID Actividad</th>
                  <th className="py-2 pr-4 font-medium">Entregas</th>
                  <th className="py-2 pr-4 font-medium">Nota Prom.</th>
                  <th className="py-2 pr-4 font-medium">Tardías</th>
                  <th className="py-2 font-medium">Alertas</th>
                </tr>
              </thead>
              <tbody>
                {activities.map(a => (
                  <tr key={a.actividad_id} className="border-b border-[#2e2e2e]">
                    <td className="py-2.5 pr-4 text-[#f0f0f0] font-mono text-xs">
                      {a.actividad_id.slice(0, 8)}…
                    </td>
                    <td className="py-2.5 pr-4 text-[#f0f0f0]">{a.submission_count}</td>
                    <td className="py-2.5 pr-4 text-[#f0f0f0]">
                      {a.avg_grade !== null ? a.avg_grade.toFixed(1) : '—'}
                    </td>
                    <td className="py-2.5 pr-4 text-[#f0f0f0]">{a.late_count}</td>
                    <td className="py-2.5 text-[#f0f0f0]">{a.flagged_count}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Grade distribution */}
      {!isEmpty && activities[0] && (
        <div className="bg-[#1a1a1a] border border-[#2e2e2e] rounded-xl p-5">
          <h2 className="text-[#f0f0f0] font-medium mb-4">Distribución de Notas</h2>
          <table className="w-full text-sm">
            <thead>
              <tr className="text-[#888888] border-b border-[#2e2e2e] text-left">
                <th className="py-2 pr-4 font-medium">Rango</th>
                <th className="py-2 font-medium">Cantidad</th>
              </tr>
            </thead>
            <tbody>
              {[
                { label: '0 – 49', count: activities.reduce((s, a) => s + a.grade_distribution.range_0_49, 0) },
                { label: '50 – 69', count: activities.reduce((s, a) => s + a.grade_distribution.range_50_69, 0) },
                { label: '70 – 89', count: activities.reduce((s, a) => s + a.grade_distribution.range_70_89, 0) },
                { label: '90 – 100', count: activities.reduce((s, a) => s + a.grade_distribution.range_90_100, 0) },
              ].map(bucket => (
                <tr key={bucket.label} className="border-b border-[#2e2e2e]">
                  <td className="py-2.5 pr-4 text-[#f0f0f0]">{bucket.label}</td>
                  <td className="py-2.5 text-[#f0f0f0]">{bucket.count}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}
