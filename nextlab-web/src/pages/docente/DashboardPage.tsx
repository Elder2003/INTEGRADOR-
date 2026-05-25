import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { getCursos } from '../../api/academic'
import type { Curso } from '../../types/academic'
import { ApiError } from '../../types'
import { LoadingSpinner } from '../../components/common/LoadingSpinner'
import { ErrorMessage } from '../../components/common/ErrorMessage'
import { CourseCard } from '../../components/docente/CourseCard'

export function DashboardPage() {
  const navigate = useNavigate()
  const [cursos, setCursos] = useState<Curso[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    getCursos()
      .then(setCursos)
      .catch(err => setError((err as ApiError).message))
      .finally(() => setLoading(false))
  }, [])

  if (loading) return <LoadingSpinner />

  return (
    <div className="p-8">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-semibold text-[#f0f0f0]">Mis Cursos</h1>
        <button
          type="button"
          onClick={() => navigate('/docente/cursos/new')}
          className="px-4 py-2 text-sm bg-[#E85D24] text-white rounded-lg hover:bg-[#d4521f] transition-colors"
        >
          Nuevo Curso
        </button>
      </div>

      {error && <ErrorMessage message={error} />}

      {!error && cursos.length === 0 && (
        <div className="text-center py-20">
          <p className="text-[#888888] mb-4">No tienes cursos aún. ¡Crea tu primer curso!</p>
          <button
            type="button"
            onClick={() => navigate('/docente/cursos/new')}
            className="text-[#E85D24] hover:underline text-sm"
          >
            Crear curso
          </button>
        </div>
      )}

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {cursos.map(curso => (
          <CourseCard
            key={curso.id}
            curso={curso}
            onClick={() => navigate(`/docente/cursos/${curso.id}`)}
          />
        ))}
      </div>
    </div>
  )
}
