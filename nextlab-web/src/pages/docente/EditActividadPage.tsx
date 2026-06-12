import { useState, useEffect, type FormEvent } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { getActividad, updateActividad } from '../../api/academic'
import type { ActividadTipo } from '../../types/academic'
import { ApiError } from '../../types'
import { LoadingSpinner } from '../../components/common/LoadingSpinner'
import { ErrorMessage } from '../../components/common/ErrorMessage'
import { ActividadTypeSelect } from '../../components/docente/ActividadTypeSelect'

function toDatetimeLocal(iso: string): string {
  // API returns "2026-05-20T23:59:00Z" — strip seconds+Z for datetime-local
  return iso.slice(0, 16)
}

function toISODeadline(local: string): string {
  return local + ':00Z'
}

export function EditActividadPage() {
  const { id: cursoId, aid } = useParams<{ id: string; aid: string }>()
  const navigate = useNavigate()

  const [title, setTitle] = useState('')
  const [tipo, setTipo] = useState<ActividadTipo | ''>('')
  const [deadline, setDeadline] = useState('')
  const [instructions, setInstructions] = useState('')
  const [lateAllowed, setLateAllowed] = useState(false)
  const [locked, setLocked] = useState(false)

  const [loading, setLoading] = useState(true)
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!aid) return
    getActividad(cursoId!, aid)
      .then(act => {
        setTitle(act.title)
        setTipo(act.tipo)
        setDeadline(toDatetimeLocal(act.deadline))
        setInstructions(act.instructions)
        setLateAllowed(act.late_submission_allowed)
        setLocked(act.has_submissions)
      })
      .catch(err => setError((err as ApiError).message))
      .finally(() => setLoading(false))
  }, [aid])

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    if (!aid || !tipo) return
    setSubmitting(true)
    setError(null)
    try {
      await updateActividad(cursoId!, aid, {
        title: title.trim(),
        tipo,
        deadline: toISODeadline(deadline),
        instructions: instructions.trim(),
        late_submission_allowed: lateAllowed,
      })
      navigate(`/docente/cursos/${cursoId}/actividades/${aid}`)
    } catch (err) {
      setError((err as ApiError).message)
    } finally {
      setSubmitting(false)
    }
  }

  if (loading) return <LoadingSpinner />

  return (
    <div className="p-8 max-w-lg">
      <h1 className="text-2xl font-semibold text-[#f0f0f0] mb-6">Editar Actividad</h1>

      {error && <ErrorMessage message={error} onDismiss={() => setError(null)} />}

      <form onSubmit={handleSubmit} className="space-y-4">
        <div>
          <label htmlFor="title" className="block text-sm text-[#888888] mb-1">Título</label>
          <input
            id="title"
            type="text"
            value={title}
            onChange={e => setTitle(e.target.value)}
            disabled={locked}
            required
            className="w-full bg-[#1a1a1a] border border-[#2e2e2e] text-[#f0f0f0] rounded-lg px-3 py-2 text-sm focus-visible:outline-none focus-visible:border-[#E85D24] disabled:opacity-50 disabled:cursor-not-allowed"
          />
          {locked && <p className="text-xs text-[#888888] mt-1">Bloqueado: existen entregas</p>}
        </div>

        <div>
          <label htmlFor="tipo" className="block text-sm text-[#888888] mb-1">Tipo</label>
          <ActividadTypeSelect id="tipo" value={tipo} onChange={setTipo} disabled={locked} />
          {locked && <p className="text-xs text-[#888888] mt-1">Bloqueado: existen entregas</p>}
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
            required
            rows={4}
            className="w-full bg-[#1a1a1a] border border-[#2e2e2e] text-[#f0f0f0] rounded-lg px-3 py-2 text-sm focus-visible:outline-none focus-visible:border-[#E85D24] resize-none"
          />
        </div>

        <div className="flex items-center gap-3">
          <input
            type="checkbox"
            id="late-allowed"
            checked={lateAllowed}
            onChange={e => setLateAllowed(e.target.checked)}
            className="w-4 h-4 accent-[#E85D24]"
          />
          <label htmlFor="late-allowed" className="text-sm text-[#f0f0f0]">
            Permitir entregas tardías
          </label>
        </div>

        <div className="flex gap-3 pt-2">
          <button
            type="submit"
            disabled={submitting || !title.trim() || !tipo || !deadline || !instructions.trim()}
            className="px-4 py-2 text-sm bg-[#E85D24] text-white rounded-lg hover:bg-[#d4521f] disabled:opacity-50 transition-colors"
          >
            {submitting ? 'Guardando...' : 'Guardar Cambios'}
          </button>
          <button
            type="button"
            onClick={() => navigate(`/docente/cursos/${cursoId}/actividades/${aid}`)}
            className="px-4 py-2 text-sm border border-[#2e2e2e] text-[#f0f0f0] rounded-lg hover:border-[#888888] transition-colors"
          >
            Cancelar
          </button>
        </div>
      </form>
    </div>
  )
}
