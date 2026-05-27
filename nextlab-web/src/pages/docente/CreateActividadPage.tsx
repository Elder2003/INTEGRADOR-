import { useState, type FormEvent } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { createActividad } from '../../api/academic'
import type { ActividadTipo } from '../../types/academic'
import { ApiError } from '../../types'
import { ErrorMessage } from '../../components/common/ErrorMessage'
import { ActividadTypeSelect } from '../../components/docente/ActividadTypeSelect'

export function CreateActividadPage() {
  const { id: cursoId } = useParams<{ id: string }>()
  const navigate = useNavigate()

  const [title, setTitle] = useState('')
  const [tipo, setTipo] = useState<ActividadTipo | ''>('')
  const [deadline, setDeadline] = useState('')
  const [instructions, setInstructions] = useState('')

  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const canSubmit = title.trim() && tipo && deadline && instructions.trim()

  function toISODeadline(local: string): string {
    return local + ':00Z'
  }

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    if (!cursoId || !canSubmit || !tipo) return
    setSubmitting(true)
    setError(null)
    try {
      await createActividad(cursoId, {
        title: title.trim(),
        tipo,
        deadline: toISODeadline(deadline),
        instructions: instructions.trim(),
      })
      navigate(`/docente/cursos/${cursoId}?tab=actividades`)
    } catch (err) {
      setError((err as ApiError).message)
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="p-8 max-w-lg">
      <h1 className="text-2xl font-semibold text-[#f0f0f0] mb-6">Nueva Actividad</h1>

      {error && <ErrorMessage message={error} onDismiss={() => setError(null)} />}

      <form onSubmit={handleSubmit} className="space-y-4">
        <div>
          <label htmlFor="title" className="block text-sm text-[#888888] mb-1">Título</label>
          <input
            id="title"
            type="text"
            value={title}
            onChange={e => setTitle(e.target.value)}
            placeholder="Ej: Tarea 1 — Recursividad"
            required
            className="w-full bg-[#1a1a1a] border border-[#2e2e2e] text-[#f0f0f0] rounded-lg px-3 py-2 text-sm focus-visible:outline-none focus-visible:border-[#E85D24]"
          />
        </div>

        <div>
          <label htmlFor="tipo" className="block text-sm text-[#888888] mb-1">Tipo</label>
          <ActividadTypeSelect id="tipo" value={tipo} onChange={setTipo} />
        </div>

        <div>
          <label htmlFor="deadline" className="block text-sm text-[#888888] mb-1">Fecha límite</label>
          <input
            id="deadline"
            type="datetime-local"
            value={deadline}
            onChange={e => setDeadline(e.target.value)}
            required
            className="w-full bg-[#1a1a1a] border border-[#2e2e2e] text-[#f0f0f0] rounded-lg px-3 py-2 text-sm focus-visible:outline-none focus-visible:border-[#E85D24]"
          />
        </div>

        <div>
          <label htmlFor="instructions" className="block text-sm text-[#888888] mb-1">Instrucciones</label>
          <textarea
            id="instructions"
            value={instructions}
            onChange={e => setInstructions(e.target.value)}
            placeholder="Describe lo que el estudiante debe entregar..."
            required
            rows={4}
            className="w-full bg-[#1a1a1a] border border-[#2e2e2e] text-[#f0f0f0] rounded-lg px-3 py-2 text-sm focus-visible:outline-none focus-visible:border-[#E85D24] resize-none"
          />
        </div>

        <div className="flex gap-3 pt-2">
          <button
            type="submit"
            disabled={!canSubmit || submitting}
            className="px-4 py-2 text-sm bg-[#E85D24] text-white rounded-lg hover:bg-[#d4521f] disabled:opacity-50 transition-colors"
          >
            {submitting ? 'Creando...' : 'Crear Actividad'}
          </button>
          <button
            type="button"
            onClick={() => navigate(`/docente/cursos/${cursoId}?tab=actividades`)}
            className="px-4 py-2 text-sm border border-[#2e2e2e] text-[#f0f0f0] rounded-lg hover:border-[#888888] transition-colors"
          >
            Cancelar
          </button>
        </div>
      </form>
    </div>
  )
}
