import { useState, useEffect, type FormEvent } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { getActividad, updateActividad, deleteActividad, grantAccess, revokeAccess } from '../../api/academic'
import type { Actividad, CodocenteGrant } from '../../types/academic'
import { ApiError } from '../../types'
import { LoadingSpinner } from '../../components/common/LoadingSpinner'
import { ErrorMessage } from '../../components/common/ErrorMessage'
import { ConfirmModal } from '../../components/common/ConfirmModal'

const TIPO_LABELS: Record<string, string> = {
  practica: 'Práctica',
  tarea: 'Tarea',
  informe: 'Informe',
  examen: 'Examen',
  defensa_oral: 'Defensa Oral',
}

export function ActividadDetailPage() {
  const { id: cursoId, aid } = useParams<{ id: string; aid: string }>()
  const navigate = useNavigate()

  const [actividad, setActividad] = useState<Actividad | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  // Delete actividad
  const [showDelete, setShowDelete] = useState(false)
  const [deleting, setDeleting] = useState(false)
  const [deleteError, setDeleteError] = useState<string | null>(null)

  // Late submissions toggle
  const [togglingLate, setTogglingLate] = useState(false)
  const [toggleError, setToggleError] = useState<string | null>(null)

  // Co-docente grants (client-state only — no GET endpoint)
  const [grants, setGrants] = useState<CodocenteGrant[]>([])
  const [grantUserId, setGrantUserId] = useState('')
  const [grantLoading, setGrantLoading] = useState(false)
  const [grantError, setGrantError] = useState<string | null>(null)

  useEffect(() => {
    if (!aid) return
    getActividad(aid)
      .then(setActividad)
      .catch(err => setError((err as ApiError).message))
      .finally(() => setLoading(false))
  }, [aid])

  async function handleDelete() {
    if (!aid) return
    setDeleting(true)
    try {
      await deleteActividad(aid)
      navigate(`/docente/cursos/${cursoId}?tab=actividades`)
    } catch (err) {
      setDeleteError((err as ApiError).message)
    } finally {
      setDeleting(false)
      setShowDelete(false)
    }
  }

  async function handleToggleLate() {
    if (!aid || !actividad) return
    setTogglingLate(true)
    setToggleError(null)
    try {
      const updated = await updateActividad(aid, {
        late_submission_allowed: !actividad.late_submission_allowed,
      })
      setActividad(updated)
    } catch (err) {
      setToggleError((err as ApiError).message)
    } finally {
      setTogglingLate(false)
    }
  }

  async function handleGrant(e: FormEvent) {
    e.preventDefault()
    if (!aid || !grantUserId.trim()) return
    setGrantLoading(true)
    setGrantError(null)
    try {
      const grant = await grantAccess(aid, grantUserId.trim())
      setGrants(prev => [...prev, grant])
      setGrantUserId('')
    } catch (err) {
      setGrantError((err as ApiError).message)
    } finally {
      setGrantLoading(false)
    }
  }

  async function handleRevoke(userId: string) {
    if (!aid) return
    setGrantError(null)
    try {
      await revokeAccess(aid, userId)
      setGrants(prev => prev.filter(g => g.user_id !== userId))
    } catch (err) {
      setGrantError((err as ApiError).message)
    }
  }

  if (loading) return <LoadingSpinner />
  if (error) return <ErrorMessage message={error} />
  if (!actividad) return null

  return (
    <div className="p-8 max-w-2xl">
      {/* Header */}
      <div className="flex items-start justify-between mb-6">
        <div>
          <h1 className="text-2xl font-semibold text-[#f0f0f0]">{actividad.title}</h1>
          <span className="mt-1 inline-block text-xs bg-[#E85D24]/20 text-[#E85D24] px-2 py-0.5 rounded">
            {TIPO_LABELS[actividad.tipo] ?? actividad.tipo}
          </span>
        </div>
        <div className="flex gap-2">
          <button
            type="button"
            onClick={() => navigate(`/docente/cursos/${cursoId}/actividades/${aid}/edit`)}
            className="px-3 py-1.5 text-sm border border-[#2e2e2e] text-[#f0f0f0] rounded-lg hover:border-[#E85D24] transition-colors"
          >
            Editar
          </button>
          <button
            type="button"
            onClick={() => setShowDelete(true)}
            className="px-3 py-1.5 text-sm border border-[#2e2e2e] text-red-400 rounded-lg hover:border-red-400 transition-colors"
          >
            Eliminar
          </button>
        </div>
      </div>

      {deleteError && <ErrorMessage message={deleteError} onDismiss={() => setDeleteError(null)} />}

      {/* Details */}
      <div className="bg-[#1a1a1a] border border-[#2e2e2e] rounded-xl p-5 space-y-4 mb-6">
        <div>
          <p className="text-xs text-[#888888] uppercase tracking-wide mb-1">Fecha límite</p>
          <p className="text-[#f0f0f0]">{new Date(actividad.deadline).toLocaleString()}</p>
        </div>
        <div>
          <p className="text-xs text-[#888888] uppercase tracking-wide mb-1">Instrucciones</p>
          <p className="text-[#f0f0f0] text-sm whitespace-pre-wrap">{actividad.instructions}</p>
        </div>
        <div className="flex items-center justify-between">
          <div>
            <p className="text-xs text-[#888888] uppercase tracking-wide mb-1">Entregas tardías</p>
            <p className="text-[#f0f0f0] text-sm">
              {actividad.late_submission_allowed ? 'Permitidas' : 'No permitidas'}
            </p>
          </div>
          <button
            type="button"
            onClick={handleToggleLate}
            disabled={togglingLate}
            className="px-3 py-1.5 text-xs border border-[#2e2e2e] text-[#f0f0f0] rounded-lg hover:border-[#E85D24] transition-colors disabled:opacity-50"
          >
            {togglingLate ? 'Guardando...' : (actividad.late_submission_allowed ? 'Deshabilitar' : 'Habilitar')}
          </button>
        </div>
        {toggleError && <ErrorMessage message={toggleError} onDismiss={() => setToggleError(null)} />}
      </div>

      {/* Co-docente grants */}
      <div className="bg-[#1a1a1a] border border-[#2e2e2e] rounded-xl p-5">
        <h2 className="text-[#f0f0f0] font-medium mb-4">Acceso co-docente</h2>

        {grantError && <ErrorMessage message={grantError} onDismiss={() => setGrantError(null)} />}

        <form onSubmit={handleGrant} className="flex gap-2 mb-4">
          <input
            type="text"
            value={grantUserId}
            onChange={e => setGrantUserId(e.target.value)}
            placeholder="ID del co-docente (UUID)"
            className="flex-1 bg-[#242424] border border-[#2e2e2e] text-[#f0f0f0] rounded-lg px-3 py-1.5 text-sm focus-visible:outline-none focus-visible:border-[#E85D24]"
          />
          <button
            type="submit"
            disabled={grantLoading || !grantUserId.trim()}
            className="px-3 py-1.5 text-sm bg-[#E85D24] text-white rounded-lg hover:bg-[#d4521f] disabled:opacity-50 transition-colors"
          >
            {grantLoading ? 'Otorgando...' : 'Otorgar acceso'}
          </button>
        </form>

        {grants.length === 0 ? (
          <p className="text-[#888888] text-sm">No hay co-docentes con acceso.</p>
        ) : (
          <ul className="space-y-2">
            {grants.map(grant => (
              <li
                key={grant.id}
                className="flex items-center justify-between text-sm"
              >
                <span className="text-[#f0f0f0] font-mono text-xs">{grant.user_id}</span>
                <button
                  type="button"
                  onClick={() => handleRevoke(grant.user_id)}
                  className="text-red-400 hover:underline text-xs"
                >
                  Revocar
                </button>
              </li>
            ))}
          </ul>
        )}

        <p className="text-[#888888] text-xs mt-4">
          La lista de co-docentes se restablece al recargar la página.
        </p>
      </div>

      <ConfirmModal
        isOpen={showDelete}
        title="Eliminar actividad"
        message="¿Eliminar esta actividad? Esta acción no se puede deshacer."
        isLoading={deleting}
        onConfirm={handleDelete}
        onCancel={() => setShowDelete(false)}
      />
    </div>
  )
}
