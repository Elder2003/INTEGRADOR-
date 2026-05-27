import { useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { getStudentActividades } from '../../api/student'
import type { StudentActividadEntry } from '../../types/academic'
import { LoadingSpinner } from '../../components/common/LoadingSpinner'
import { ErrorMessage } from '../../components/common/ErrorMessage'

const tipoLabel: Record<string, string> = {
  practica: 'Práctica',
  tarea: 'Tarea',
  informe: 'Informe',
  examen: 'Examen',
  defensa_oral: 'Defensa Oral',
}

function formatDeadline(iso: string): { label: string; isPast: boolean } {
  const d = new Date(iso)
  const isPast = d < new Date()
  const label = d.toLocaleString('es-BO', {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
  return { label, isPast }
}

export function ActividadesPage() {
  const { id: cursoId } = useParams<{ id: string }>()
  const [items, setItems] = useState<StudentActividadEntry[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [isAccessDenied, setIsAccessDenied] = useState(false)

  useEffect(() => {
    if (!cursoId) return
    getStudentActividades(cursoId)
      .then(setItems)
      .catch((err: Error) => {
        if (err.message.toLowerCase().includes('not enrolled') || err.message.toLowerCase().includes('forbidden')) {
          setIsAccessDenied(true)
          setError('No tienes acceso a este curso.')
        } else {
          setError(err.message)
        }
      })
      .finally(() => setLoading(false))
  }, [cursoId])

  if (loading) return <LoadingSpinner />

  return (
    <div className="p-8">
      <div className="flex items-center gap-4 mb-6">
        <Link to="/student/dashboard" className="text-sm text-[#888888] hover:text-[#E85D24] transition-colors">
          ← Mis Cursos
        </Link>
        <h1 className="text-2xl font-semibold text-[#f0f0f0]">Actividades del Curso</h1>
      </div>

      {error && (
        <div className="space-y-3">
          <ErrorMessage message={error} />
          {isAccessDenied && (
            <Link to="/student/dashboard" className="text-sm text-[#E85D24] hover:underline">
              ← Volver a Mis Cursos
            </Link>
          )}
        </div>
      )}

      {!error && items.length === 0 && (
        <p className="text-[#888888]">Este curso aún no tiene actividades publicadas.</p>
      )}

      {!error && items.length > 0 && (
        <ul className="space-y-2">
          {items.map((act) => {
            const { label, isPast } = formatDeadline(act.deadline)
            return (
              <li key={act.id}>
                <Link
                  to={`/student/cursos/${cursoId}/actividades/${act.id}`}
                  className="block bg-[#1a1a1a] border border-[#2e2e2e] rounded-lg p-4 hover:border-[#E85D24] transition-colors"
                >
                  <div className="flex items-start justify-between gap-4">
                    <div className="flex-1 min-w-0">
                      <p className="text-[#f0f0f0] font-medium">{act.title}</p>
                      <div className="flex flex-wrap items-center gap-2 mt-1">
                        <span className="bg-[#2e2e2e] text-[#888888] text-xs px-2 py-0.5 rounded">
                          {tipoLabel[act.tipo] ?? act.tipo}
                        </span>
                        <span className={`text-xs ${isPast ? 'text-[#888888]' : 'text-[#888888]'}`}>
                          {label}
                        </span>
                        {isPast && (
                          <span className="text-xs text-[#888888] font-medium">Vencido</span>
                        )}
                        {act.trace_enabled && (
                          <span className="bg-[#E85D24]/20 text-[#E85D24] text-xs px-2 py-0.5 rounded">
                            Requiere monitoreo
                          </span>
                        )}
                      </div>
                    </div>
                  </div>
                </Link>
              </li>
            )
          })}
        </ul>
      )}
    </div>
  )
}
