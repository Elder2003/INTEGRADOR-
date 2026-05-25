import { useState, useEffect, type FormEvent } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { getCurso, updateCurso } from '../../api/academic'
import { ApiError } from '../../types'
import { LoadingSpinner } from '../../components/common/LoadingSpinner'
import { ErrorMessage } from '../../components/common/ErrorMessage'

export function EditCursoPage() {
  const { id: cursoId } = useParams<{ id: string }>()
  const navigate = useNavigate()

  const [materiaDisplay, setMateriaDisplay] = useState('')
  const [semester, setSemester] = useState('')
  const [section, setSection] = useState('')

  const [loading, setLoading] = useState(true)
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!cursoId) return
    // React 19 batches all setState calls automatically — no extra wrapper needed
    getCurso(cursoId)
      .then(curso => {
        setMateriaDisplay(curso.materia_nombre)
        setSemester(curso.semester)
        setSection(curso.section)
      })
      .catch(err => setError((err as ApiError).message))
      .finally(() => setLoading(false))
  }, [cursoId])

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    if (!cursoId) return
    setSubmitting(true)
    setError(null)
    try {
      await updateCurso(cursoId, { semester: semester.trim(), section: section.trim() })
      navigate(`/docente/cursos/${cursoId}`)
    } catch (err) {
      setError((err as ApiError).message)
    } finally {
      setSubmitting(false)
    }
  }

  if (loading) return <LoadingSpinner />

  return (
    <div className="p-8 max-w-lg">
      <h1 className="text-2xl font-semibold text-[#f0f0f0] mb-6">Editar Curso</h1>

      {error && <ErrorMessage message={error} onDismiss={() => setError(null)} />}

      <form onSubmit={handleSubmit} className="space-y-4">
        <div>
          <label htmlFor="materia-display" className="block text-sm text-[#888888] mb-1">Materia</label>
          <p
            id="materia-display"
            className="text-[#f0f0f0] bg-[#1a1a1a] border border-[#2e2e2e] rounded-lg px-3 py-2 text-sm opacity-60"
          >
            {materiaDisplay}
          </p>
        </div>

        <div>
          <label htmlFor="semester" className="block text-sm text-[#888888] mb-1">Semestre</label>
          <input
            id="semester"
            type="text"
            value={semester}
            onChange={e => setSemester(e.target.value)}
            required
            className="w-full bg-[#1a1a1a] border border-[#2e2e2e] text-[#f0f0f0] rounded-lg px-3 py-2 text-sm focus-visible:outline-none focus-visible:border-[#E85D24]"
          />
        </div>

        <div>
          <label htmlFor="section" className="block text-sm text-[#888888] mb-1">Sección</label>
          <input
            id="section"
            type="text"
            value={section}
            onChange={e => setSection(e.target.value)}
            required
            className="w-full bg-[#1a1a1a] border border-[#2e2e2e] text-[#f0f0f0] rounded-lg px-3 py-2 text-sm focus-visible:outline-none focus-visible:border-[#E85D24]"
          />
        </div>

        <div className="flex gap-3 pt-2">
          <button
            type="submit"
            disabled={submitting || !semester.trim() || !section.trim()}
            className="px-4 py-2 text-sm bg-[#E85D24] text-white rounded-lg hover:bg-[#d4521f] disabled:opacity-50 transition-colors"
          >
            {submitting ? 'Guardando...' : 'Guardar Cambios'}
          </button>
          <button
            type="button"
            onClick={() => navigate(`/docente/cursos/${cursoId}`)}
            className="px-4 py-2 text-sm border border-[#2e2e2e] text-[#f0f0f0] rounded-lg hover:border-[#888888] transition-colors"
          >
            Cancelar
          </button>
        </div>
      </form>
    </div>
  )
}
