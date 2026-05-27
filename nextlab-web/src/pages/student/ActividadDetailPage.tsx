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

export function ActividadDetailPage() {
  const { id: cursoId, aid } = useParams<{ id: string; aid: string }>()
  const [actividad, setActividad] = useState<StudentActividadEntry | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!cursoId || !aid) return
    getStudentActividades(cursoId)
      .then((list) => {
        const found = list.find((a) => a.id === aid)
        if (!found) {
          setError('Actividad no encontrada.')
        } else {
          setActividad(found)
        }
      })
      .catch((err: Error) => {
        if (err.message.toLowerCase().includes('not enrolled') || err.message.toLowerCase().includes('forbidden')) {
          setError('No tienes acceso a este curso.')
        } else {
          setError(err.message)
        }
      })
      .finally(() => setLoading(false))
  }, [cursoId, aid])

  if (loading) return <LoadingSpinner />

  if (error) {
    return (
      <div className="p-8 space-y-4">
        <ErrorMessage message={error} />
        <Link
          to={`/student/cursos/${cursoId}/actividades`}
          className="text-sm text-[#E85D24] hover:underline"
        >
          ← Volver al curso
        </Link>
      </div>
    )
  }

  if (!actividad) return null

  const { label, isPast } = formatDeadline(actividad.deadline)

  return (
    <div className="p-8 max-w-3xl">
      <Link
        to={`/student/cursos/${cursoId}/actividades`}
        className="text-sm text-[#888888] hover:text-[#E85D24] transition-colors"
      >
        ← Volver al curso
      </Link>

      <h1 className="text-2xl font-semibold text-[#f0f0f0] mt-4 mb-4">{actividad.title}</h1>

      <div className="flex flex-wrap items-center gap-2 mb-6">
        <span className="bg-[#2e2e2e] text-[#888888] text-xs px-2 py-0.5 rounded">
          {tipoLabel[actividad.tipo] ?? actividad.tipo}
        </span>
        <span className="text-sm text-[#888888]">{label}</span>
        {isPast && (
          <span className="text-xs text-[#888888] font-medium">Vencido</span>
        )}
        {actividad.trace_enabled && (
          <span className="bg-[#E85D24]/20 text-[#E85D24] text-xs px-2 py-0.5 rounded">
            Requiere monitoreo
          </span>
        )}
      </div>

      <div className="bg-[#1a1a1a] border border-[#2e2e2e] rounded-lg p-4">
        <h2 className="text-sm font-medium text-[#888888] mb-2">Instrucciones</h2>
        <pre className="whitespace-pre-wrap text-[#f0f0f0] text-sm">{actividad.instructions}</pre>
      </div>
    </div>
  )
}
