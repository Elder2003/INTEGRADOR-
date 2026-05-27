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

  const isEmpty = data.activity_summaries.length === 0

  return (
    <div className="p-8 max-w-3xl">
      <Link
        to={`/docente/cursos/${cid}`}
        className="text-sm text-[#888888] hover:text-[#f0f0f0] transition-colors mb-4 inline-block"
      >
        ← Curso
      </Link>

      <h1 className="text-2xl font-semibold text-[#f0f0f0] mb-6">Analíticas del Curso</h1>

      {/* Per-activity summary */}
      <div className="bg-[#1a1a1a] border border-[#2e2e2e] rounded-xl p-5 mb-4">
        <h2 className="text-[#f0f0f0] font-medium mb-4">Resumen por Actividad</h2>
        {isEmpty ? (
          <p className="text-[#888888] text-sm">Sin datos de calificaciones aún.</p>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="text-[#888888] border-b border-[#2e2e2e] text-left">
                  <th className="py-2 pr-4 font-medium">Actividad</th>
                  <th className="py-2 pr-4 font-medium">Entregas</th>
                  <th className="py-2 pr-4 font-medium">Nota Prom.</th>
                  <th className="py-2 font-medium">Tardías</th>
                </tr>
              </thead>
              <tbody>
                {data.activity_summaries.map(summary => (
                  <tr key={summary.actividad_id} className="border-b border-[#2e2e2e]">
                    <td className="py-2.5 pr-4 text-[#f0f0f0]">{summary.activity_name}</td>
                    <td className="py-2.5 pr-4 text-[#f0f0f0]">{summary.submissions}</td>
                    <td className="py-2.5 pr-4 text-[#f0f0f0]">
                      {summary.average_grade !== null
                        ? summary.average_grade.toFixed(1)
                        : '—'}
                    </td>
                    <td className="py-2.5 text-[#f0f0f0]">{summary.late_count}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Overall grade distribution */}
      <div className="bg-[#1a1a1a] border border-[#2e2e2e] rounded-xl p-5">
        <h2 className="text-[#f0f0f0] font-medium mb-4">Distribución General</h2>
        {isEmpty || data.overall_grade_distribution.length === 0 ? (
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
              {data.overall_grade_distribution.map(bucket => (
                <tr key={bucket.range_label} className="border-b border-[#2e2e2e]">
                  <td className="py-2.5 pr-4 text-[#f0f0f0]">{bucket.range_label}</td>
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
