import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { getStudentCursos } from '../../api/student'
import type { StudentCursoEntry } from '../../types/academic'
import { LoadingSpinner } from '../../components/common/LoadingSpinner'
import { ErrorMessage } from '../../components/common/ErrorMessage'

export function DashboardPage() {
  const [items, setItems] = useState<StudentCursoEntry[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    getStudentCursos()
      .then(setItems)
      .catch((err: Error) => setError(err.message))
      .finally(() => setLoading(false))
  }, [])

  if (loading) return <LoadingSpinner />
  if (error) return <ErrorMessage message={error} />

  return (
    <div className="p-8">
      <h1 className="text-2xl font-semibold text-[#f0f0f0] mb-6">Mis Cursos</h1>

      {items.length === 0 ? (
        <p className="text-[#888888]">Aún no estás inscrito en ningún curso.</p>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {items.map((curso) => (
            <Link
              key={curso.id}
              to={`/student/cursos/${curso.id}/actividades`}
              className="bg-[#1a1a1a] border border-[#2e2e2e] rounded-lg p-4 hover:border-[#E85D24] transition-colors"
            >
              <h2 className="text-[#f0f0f0] font-semibold">{curso.materia_name}</h2>
              <p className="text-sm text-[#888888]">{curso.docente_email}</p>
              <p className="text-sm text-[#888888]">{curso.semester_label} · Sección {curso.section}</p>
              <p className="text-sm text-[#888888]">{curso.grupo?.name ?? 'Sin grupo'}</p>
            </Link>
          ))}
        </div>
      )}
    </div>
  )
}
