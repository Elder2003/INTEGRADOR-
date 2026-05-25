import { useState, useEffect, type FormEvent } from 'react'
import { useNavigate } from 'react-router-dom'
import { getMaterias, createCurso } from '../../api/academic'
import type { Materia } from '../../types/academic'
import { ApiError } from '../../types'
import { LoadingSpinner } from '../../components/common/LoadingSpinner'
import { ErrorMessage } from '../../components/common/ErrorMessage'

export function CreateCursoPage() {
  const navigate = useNavigate()
  const [materias, setMaterias] = useState<Materia[]>([])
  const [materiasLoading, setMateriasLoading] = useState(true)

  const [materiaId, setMateriaId] = useState('')
  const [semester, setSemester] = useState('')
  const [section, setSection] = useState('')

  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    getMaterias()
      .then(setMaterias)
      .finally(() => setMateriasLoading(false))
  }, [])

  const canSubmit = materiaId.trim() && semester.trim() && section.trim()

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    if (!canSubmit) return
    setSubmitting(true)
    setError(null)
    try {
      const curso = await createCurso({
        materia_id: materiaId.trim(),
        semester: semester.trim(),
        section: section.trim(),
      })
      navigate(`/docente/cursos/${curso.id}?tab=actividades`)
    } catch (err) {
      setError((err as ApiError).message)
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="p-8 max-w-lg">
      <h1 className="text-2xl font-semibold text-[#f0f0f0] mb-6">Nuevo Curso</h1>

      {error && <ErrorMessage message={error} onDismiss={() => setError(null)} />}

      <form onSubmit={handleSubmit} className="space-y-4">
        <div>
          <label htmlFor="materia" className="block text-sm text-[#888888] mb-1">Materia</label>
          {materiasLoading ? (
            <LoadingSpinner />
          ) : materias.length > 0 ? (
            <select
              id="materia"
              value={materiaId}
              onChange={e => setMateriaId(e.target.value)}
              required
              className="w-full bg-[#1a1a1a] border border-[#2e2e2e] text-[#f0f0f0] rounded-lg px-3 py-2 text-sm focus-visible:outline-none focus-visible:border-[#E85D24]"
            >
              <option value="" disabled>Seleccionar materia</option>
              {materias.map(m => (
                <option key={m.id} value={m.id}>{m.nombre} ({m.codigo})</option>
              ))}
            </select>
          ) : (
            <div>
              <input
                id="materia"
                type="text"
                value={materiaId}
                onChange={e => setMateriaId(e.target.value)}
                placeholder="UUID de la materia"
                required
                className="w-full bg-[#1a1a1a] border border-[#2e2e2e] text-[#f0f0f0] rounded-lg px-3 py-2 text-sm focus-visible:outline-none focus-visible:border-[#E85D24]"
              />
              <p className="text-xs text-[#888888] mt-1">
                Contacta a tu admin para obtener el ID de materia.
              </p>
            </div>
          )}
        </div>

        <div>
          <label htmlFor="semester" className="block text-sm text-[#888888] mb-1">Semestre</label>
          <input
            id="semester"
            type="text"
            value={semester}
            onChange={e => setSemester(e.target.value)}
            placeholder="Ej: 2026-1"
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
            placeholder="Ej: A"
            required
            className="w-full bg-[#1a1a1a] border border-[#2e2e2e] text-[#f0f0f0] rounded-lg px-3 py-2 text-sm focus-visible:outline-none focus-visible:border-[#E85D24]"
          />
        </div>

        <div className="flex gap-3 pt-2">
          <button
            type="submit"
            disabled={!canSubmit || submitting}
            className="px-4 py-2 text-sm bg-[#E85D24] text-white rounded-lg hover:bg-[#d4521f] disabled:opacity-50 transition-colors"
          >
            {submitting ? 'Creando...' : 'Crear Curso'}
          </button>
          <button
            type="button"
            onClick={() => navigate('/docente/dashboard')}
            className="px-4 py-2 text-sm border border-[#2e2e2e] text-[#f0f0f0] rounded-lg hover:border-[#888888] transition-colors"
          >
            Cancelar
          </button>
        </div>
      </form>
    </div>
  )
}
