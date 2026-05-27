import { useState, useEffect, type FormEvent } from 'react'
import { Link, useNavigate, useParams, useSearchParams } from 'react-router-dom'
import {
  getCurso, deleteCurso,
  getActividades, deleteActividad,
  getRoster,
  getGrupos, createGrupo, updateGrupo, deleteGrupo,
  addStudentToGrupo, removeStudentFromGrupo,
} from '../../api/academic'
import type { Curso, Actividad, RosterEntry, Grupo } from '../../types/academic'
import { ApiError } from '../../types'
import { LoadingSpinner } from '../../components/common/LoadingSpinner'
import { ErrorMessage } from '../../components/common/ErrorMessage'
import { ConfirmModal } from '../../components/common/ConfirmModal'

type Tab = 'actividades' | 'estudiantes' | 'grupos'
const VALID_TABS: Tab[] = ['actividades', 'estudiantes', 'grupos']

export function CursoDetailPage() {
  const { id: cursoId } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [searchParams, setSearchParams] = useSearchParams()

  const rawTab = searchParams.get('tab')
  const activeTab: Tab = VALID_TABS.includes(rawTab as Tab) ? (rawTab as Tab) : 'actividades'

  // Course
  const [curso, setCurso] = useState<Curso | null>(null)
  const [loading, setLoading] = useState(true)
  const [pageError, setPageError] = useState<string | null>(null)

  // Delete curso
  const [showDeleteCurso, setShowDeleteCurso] = useState(false)
  const [deletingCurso, setDeletingCurso] = useState(false)
  const [deleteCursoError, setDeleteCursoError] = useState<string | null>(null)

  // Actividades
  const [actividades, setActividades] = useState<Actividad[]>([])
  const [actLoading, setActLoading] = useState(false)
  const [actError, setActError] = useState<string | null>(null)
  const [deleteActividadId, setDeleteActividadId] = useState<string | null>(null)
  const [deletingActividad, setDeletingActividad] = useState(false)
  const [deleteActError, setDeleteActError] = useState<string | null>(null)

  // Roster
  const [roster, setRoster] = useState<RosterEntry[]>([])
  const [rosterLoading, setRosterLoading] = useState(false)
  const [rosterError, setRosterError] = useState<string | null>(null)
  const [rosterFetched, setRosterFetched] = useState(false)
  const [assignLoading, setAssignLoading] = useState<Record<string, boolean>>({})
  const [assignError, setAssignError] = useState<string | null>(null)

  // Grupos
  const [grupos, setGrupos] = useState<Grupo[]>([])
  const [gruposLoading, setGruposLoading] = useState(false)
  const [gruposError, setGruposError] = useState<string | null>(null)
  // Create grupo form
  const [showCreateGrupo, setShowCreateGrupo] = useState(false)
  const [newGrupoNombre, setNewGrupoNombre] = useState('')
  const [newGrupoCapacity, setNewGrupoCapacity] = useState('')
  const [creatingGrupo, setCreatingGrupo] = useState(false)
  const [createGrupoError, setCreateGrupoError] = useState<string | null>(null)
  // Edit grupo
  const [editingGrupoId, setEditingGrupoId] = useState<string | null>(null)
  const [editGrupoNombre, setEditGrupoNombre] = useState('')
  const [editGrupoCapacity, setEditGrupoCapacity] = useState('')
  const [savingGrupo, setSavingGrupo] = useState(false)
  const [editGrupoError, setEditGrupoError] = useState<string | null>(null)
  // Delete grupo
  const [deleteGrupoId, setDeleteGrupoId] = useState<string | null>(null)
  const [deletingGrupo, setDeletingGrupo] = useState(false)
  const [deleteGrupoError, setDeleteGrupoError] = useState<string | null>(null)

  // Fetch curso on mount
  useEffect(() => {
    if (!cursoId) return
    getCurso(cursoId)
      .then(setCurso)
      .catch(err => setPageError((err as ApiError).message))
      .finally(() => setLoading(false))
  }, [cursoId])

  // Fetch actividades on mount (default tab)
  useEffect(() => {
    if (!cursoId) return
    setActLoading(true)
    getActividades(cursoId)
      .then(setActividades)
      .catch(err => setActError((err as ApiError).message))
      .finally(() => setActLoading(false))
  }, [cursoId])

  // Fetch grupos on mount (needed for both Grupos tab and student assignment dropdown)
  useEffect(() => {
    if (!cursoId) return
    setGruposLoading(true)
    getGrupos(cursoId)
      .then(setGrupos)
      .catch(err => setGruposError((err as ApiError).message))
      .finally(() => setGruposLoading(false))
  }, [cursoId])

  // Fetch roster lazily on first estudiantes tab visit
  useEffect(() => {
    if (!cursoId || activeTab !== 'estudiantes' || rosterFetched) return
    setRosterLoading(true)
    getRoster(cursoId)
      .then(data => { setRoster(data); setRosterFetched(true) })
      .catch(err => setRosterError((err as ApiError).message))
      .finally(() => setRosterLoading(false))
  }, [cursoId, activeTab, rosterFetched])

  function setTab(tab: Tab) {
    setSearchParams({ tab }, { replace: false })
  }

  // --- Curso handlers ---
  async function handleDeleteCurso() {
    if (!cursoId) return
    setDeletingCurso(true)
    try {
      await deleteCurso(cursoId)
      navigate('/docente/dashboard')
    } catch (err) {
      setDeleteCursoError((err as ApiError).message)
    } finally {
      setDeletingCurso(false)
      setShowDeleteCurso(false)
    }
  }

  // --- Actividad handlers ---
  async function handleDeleteActividad() {
    if (!deleteActividadId) return
    setDeletingActividad(true)
    try {
      await deleteActividad(deleteActividadId)
      setActividades(prev => prev.filter(a => a.id !== deleteActividadId))
    } catch (err) {
      setDeleteActError((err as ApiError).message)
    } finally {
      setDeletingActividad(false)
      setDeleteActividadId(null)
    }
  }

  // --- Grupo handlers ---
  async function handleCreateGrupo(e: FormEvent) {
    e.preventDefault()
    if (!cursoId || !newGrupoNombre.trim()) return
    setCreatingGrupo(true)
    setCreateGrupoError(null)
    try {
      const grupo = await createGrupo(cursoId, {
        nombre: newGrupoNombre.trim(),
        capacity: newGrupoCapacity ? parseInt(newGrupoCapacity) : undefined,
      })
      setGrupos(prev => [...prev, grupo])
      setNewGrupoNombre('')
      setNewGrupoCapacity('')
      setShowCreateGrupo(false)
    } catch (err) {
      setCreateGrupoError((err as ApiError).message)
    } finally {
      setCreatingGrupo(false)
    }
  }

  function handleStartEditGrupo(grupo: Grupo) {
    setEditingGrupoId(grupo.id)
    setEditGrupoNombre(grupo.nombre)
    setEditGrupoCapacity(grupo.capacity?.toString() ?? '')
    setEditGrupoError(null)
  }

  async function handleSaveGrupo(e: FormEvent) {
    e.preventDefault()
    if (!cursoId || !editingGrupoId || !editGrupoNombre.trim()) return
    setSavingGrupo(true)
    setEditGrupoError(null)
    try {
      const updated = await updateGrupo(cursoId, editingGrupoId, {
        nombre: editGrupoNombre.trim(),
        capacity: editGrupoCapacity ? parseInt(editGrupoCapacity) : undefined,
      })
      setGrupos(prev => prev.map(g => g.id === editingGrupoId ? updated : g))
      setEditingGrupoId(null)
    } catch (err) {
      setEditGrupoError((err as ApiError).message)
    } finally {
      setSavingGrupo(false)
    }
  }

  async function handleDeleteGrupo() {
    if (!cursoId || !deleteGrupoId) return
    setDeletingGrupo(true)
    try {
      await deleteGrupo(cursoId, deleteGrupoId)
      setGrupos(prev => prev.filter(g => g.id !== deleteGrupoId))
    } catch (err) {
      setDeleteGrupoError((err as ApiError).message)
    } finally {
      setDeletingGrupo(false)
      setDeleteGrupoId(null)
    }
  }

  // --- Student assignment handler ---
  async function handleAssignGrupo(userId: string, currentGrupoId: string | null, newGrupoId: string) {
    if (!cursoId) return
    setAssignLoading(prev => ({ ...prev, [userId]: true }))
    setAssignError(null)
    try {
      if (currentGrupoId && newGrupoId === '') {
        await removeStudentFromGrupo(cursoId, currentGrupoId, userId)
        setRoster(prev => prev.map(r => r.user_id === userId ? { ...r, grupo_id: null, grupo_nombre: null } : r))
      } else if (newGrupoId) {
        if (currentGrupoId) {
          await removeStudentFromGrupo(cursoId, currentGrupoId, userId)
        }
        await addStudentToGrupo(cursoId, newGrupoId, userId)
        const grupo = grupos.find(g => g.id === newGrupoId)
        setRoster(prev => prev.map(r =>
          r.user_id === userId ? { ...r, grupo_id: newGrupoId, grupo_nombre: grupo?.nombre ?? null } : r
        ))
      }
    } catch (err) {
      setAssignError((err as ApiError).message)
    } finally {
      setAssignLoading(prev => ({ ...prev, [userId]: false }))
    }
  }

  if (loading) return <LoadingSpinner />
  if (pageError) return <ErrorMessage message={pageError} />
  if (!curso) return null

  return (
    <div className="p-8">
      {/* Page header */}
      <div className="flex items-start justify-between mb-6">
        <div>
          <h1 className="text-2xl font-semibold text-[#f0f0f0]">{curso.materia_nombre}</h1>
          <p className="text-[#888888] text-sm mt-1">
            {curso.semester} · Sección {curso.section} · {curso.student_count} estudiante{curso.student_count !== 1 ? 's' : ''}
          </p>
        </div>
        <div className="flex gap-2">
          <Link
            to={`/docente/cursos/${cursoId}/analytics`}
            className="px-3 py-1.5 text-sm border border-[#2e2e2e] text-[#f0f0f0] rounded-lg hover:border-[#E85D24] transition-colors"
          >
            Ver Analíticas del Curso
          </Link>
          <button
            type="button"
            onClick={() => navigate(`/docente/cursos/${cursoId}/edit`)}
            className="px-3 py-1.5 text-sm border border-[#2e2e2e] text-[#f0f0f0] rounded-lg hover:border-[#E85D24] transition-colors"
          >
            Editar
          </button>
          <button
            type="button"
            onClick={() => setShowDeleteCurso(true)}
            className="px-3 py-1.5 text-sm border border-[#2e2e2e] text-red-400 rounded-lg hover:border-red-400 transition-colors"
          >
            Eliminar
          </button>
        </div>
      </div>

      {deleteCursoError && (
        <ErrorMessage message={deleteCursoError} onDismiss={() => setDeleteCursoError(null)} />
      )}

      {/* Tab bar */}
      <div className="flex border-b border-[#2e2e2e] mb-6 gap-1">
        {VALID_TABS.map(tab => (
          <button
            key={tab}
            type="button"
            onClick={() => setTab(tab)}
            className={`px-4 py-2 text-sm font-medium transition-colors ${
              activeTab === tab
                ? 'border-b-2 border-[#E85D24] text-[#E85D24]'
                : 'text-[#888888] hover:text-[#f0f0f0]'
            }`}
          >
            {tab.charAt(0).toUpperCase() + tab.slice(1)}
          </button>
        ))}
      </div>

      {/* ===== ACTIVIDADES TAB ===== */}
      {activeTab === 'actividades' && (
        <div>
          {deleteActError && (
            <ErrorMessage message={deleteActError} onDismiss={() => setDeleteActError(null)} />
          )}
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-lg font-medium text-[#f0f0f0]">Actividades</h2>
            <button
              type="button"
              onClick={() => navigate(`/docente/cursos/${cursoId}/actividades/new`)}
              className="px-3 py-1.5 text-sm bg-[#E85D24] text-white rounded-lg hover:bg-[#d4521f] transition-colors"
            >
              Nueva Actividad
            </button>
          </div>

          {actLoading && <LoadingSpinner />}
          {actError && <ErrorMessage message={actError} />}

          {!actLoading && !actError && actividades.length === 0 && (
            <div className="text-center py-16">
              <p className="text-[#888888] mb-3">No hay actividades. Crea la primera.</p>
              <button
                type="button"
                onClick={() => navigate(`/docente/cursos/${cursoId}/actividades/new`)}
                className="text-[#E85D24] hover:underline text-sm"
              >
                Nueva Actividad
              </button>
            </div>
          )}

          <div className="space-y-2">
            {actividades.map(act => (
              <div
                key={act.id}
                className="flex items-center justify-between bg-[#1a1a1a] border border-[#2e2e2e] rounded-xl p-4"
              >
                <div>
                  <span className="text-[#f0f0f0] font-medium">{act.title}</span>
                  <span className="ml-2 text-xs bg-[#E85D24]/20 text-[#E85D24] px-2 py-0.5 rounded capitalize">
                    {act.tipo.replace('_', ' ')}
                  </span>
                  <p className="text-xs text-[#888888] mt-1">
                    Límite: {new Date(act.deadline).toLocaleString()}
                  </p>
                </div>
                <div className="flex gap-3">
                  <button
                    type="button"
                    onClick={() => navigate(`/docente/cursos/${cursoId}/actividades/${act.id}`)}
                    className="text-sm text-[#E85D24] hover:underline"
                  >
                    Ver detalle
                  </button>
                  <button
                    type="button"
                    onClick={() => setDeleteActividadId(act.id)}
                    className="text-sm text-red-400 hover:underline"
                  >
                    Eliminar
                  </button>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* ===== ESTUDIANTES TAB ===== */}
      {activeTab === 'estudiantes' && (
        <div>
          {assignError && (
            <ErrorMessage message={assignError} onDismiss={() => setAssignError(null)} />
          )}

          {rosterLoading && <LoadingSpinner />}
          {rosterError && <ErrorMessage message={rosterError} />}

          {!rosterLoading && !rosterError && roster.length === 0 && (
            <p className="text-[#888888] text-center py-16">No hay estudiantes inscritos aún.</p>
          )}

          {!rosterLoading && roster.length > 0 && (
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="text-[#888888] border-b border-[#2e2e2e] text-left">
                    <th className="py-2 pr-4 font-medium">Email</th>
                    <th className="py-2 pr-4 font-medium">Inscripción</th>
                    <th className="py-2 font-medium">Grupo</th>
                  </tr>
                </thead>
                <tbody>
                  {roster.map(entry => (
                    <tr key={entry.user_id} className="border-b border-[#2e2e2e]">
                      <td className="py-2.5 pr-4 text-[#f0f0f0]">{entry.email}</td>
                      <td className="py-2.5 pr-4 text-[#888888]">
                        {new Date(entry.enrolled_at).toLocaleDateString()}
                      </td>
                      <td className="py-2.5">
                        <select
                          disabled={!!assignLoading[entry.user_id]}
                          value={entry.grupo_id ?? ''}
                          onChange={e => handleAssignGrupo(entry.user_id, entry.grupo_id, e.target.value)}
                          className="bg-[#1a1a1a] border border-[#2e2e2e] text-[#f0f0f0] text-sm rounded px-2 py-1 focus-visible:outline-none focus-visible:border-[#E85D24] disabled:opacity-50"
                        >
                          <option value="">Sin grupo</option>
                          {grupos.map(g => (
                            <option key={g.id} value={g.id}>{g.nombre}</option>
                          ))}
                        </select>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      )}

      {/* ===== GRUPOS TAB ===== */}
      {activeTab === 'grupos' && (
        <div>
          {deleteGrupoError && (
            <ErrorMessage message={deleteGrupoError} onDismiss={() => setDeleteGrupoError(null)} />
          )}
          {editGrupoError && (
            <ErrorMessage message={editGrupoError} onDismiss={() => setEditGrupoError(null)} />
          )}
          {createGrupoError && (
            <ErrorMessage message={createGrupoError} onDismiss={() => setCreateGrupoError(null)} />
          )}

          <div className="flex items-center justify-between mb-4">
            <h2 className="text-lg font-medium text-[#f0f0f0]">Grupos</h2>
            <button
              type="button"
              onClick={() => setShowCreateGrupo(!showCreateGrupo)}
              className="px-3 py-1.5 text-sm bg-[#E85D24] text-white rounded-lg hover:bg-[#d4521f] transition-colors"
            >
              Nuevo Grupo
            </button>
          </div>

          {showCreateGrupo && (
            <form
              onSubmit={handleCreateGrupo}
              className="bg-[#1a1a1a] border border-[#2e2e2e] rounded-xl p-4 mb-4"
            >
              <h3 className="text-[#f0f0f0] font-medium mb-3">Crear grupo</h3>
              <div className="flex flex-wrap gap-3 items-end">
                <div>
                  <label htmlFor="new-grupo-nombre" className="block text-xs text-[#888888] mb-1">Nombre *</label>
                  <input
                    id="new-grupo-nombre"
                    value={newGrupoNombre}
                    onChange={e => setNewGrupoNombre(e.target.value)}
                    placeholder="Ej: Grupo A"
                    required
                    className="bg-[#242424] border border-[#2e2e2e] text-[#f0f0f0] text-sm rounded-lg px-3 py-1.5 focus-visible:outline-none focus-visible:border-[#E85D24]"
                  />
                </div>
                <div>
                  <label htmlFor="new-grupo-capacity" className="block text-xs text-[#888888] mb-1">Capacidad (opcional)</label>
                  <input
                    id="new-grupo-capacity"
                    type="number"
                    min="1"
                    value={newGrupoCapacity}
                    onChange={e => setNewGrupoCapacity(e.target.value)}
                    placeholder="Sin límite"
                    className="bg-[#242424] border border-[#2e2e2e] text-[#f0f0f0] text-sm rounded-lg px-3 py-1.5 w-32 focus-visible:outline-none focus-visible:border-[#E85D24]"
                  />
                </div>
                <button
                  type="submit"
                  disabled={creatingGrupo || !newGrupoNombre.trim()}
                  className="px-3 py-1.5 text-sm bg-[#E85D24] text-white rounded-lg disabled:opacity-50 transition-colors"
                >
                  {creatingGrupo ? 'Creando...' : 'Crear'}
                </button>
                <button
                  type="button"
                  onClick={() => { setShowCreateGrupo(false); setNewGrupoNombre(''); setNewGrupoCapacity('') }}
                  className="px-3 py-1.5 text-sm border border-[#2e2e2e] text-[#888888] rounded-lg hover:text-[#f0f0f0] transition-colors"
                >
                  Cancelar
                </button>
              </div>
            </form>
          )}

          {gruposLoading && <LoadingSpinner />}
          {gruposError && <ErrorMessage message={gruposError} />}

          {!gruposLoading && !gruposError && grupos.length === 0 && (
            <div className="text-center py-16">
              <p className="text-[#888888] mb-3">No hay grupos. Crea el primero.</p>
              <button
                type="button"
                onClick={() => setShowCreateGrupo(true)}
                className="text-[#E85D24] hover:underline text-sm"
              >
                Nuevo Grupo
              </button>
            </div>
          )}

          <div className="space-y-2">
            {grupos.map(grupo => (
              <div
                key={grupo.id}
                className="bg-[#1a1a1a] border border-[#2e2e2e] rounded-xl p-4"
              >
                {editingGrupoId === grupo.id ? (
                  <form onSubmit={handleSaveGrupo} className="flex flex-wrap gap-3 items-end">
                    <div>
                      <label htmlFor={`edit-grupo-nombre-${grupo.id}`} className="block text-xs text-[#888888] mb-1">Nombre</label>
                      <input
                        id={`edit-grupo-nombre-${grupo.id}`}
                        value={editGrupoNombre}
                        onChange={e => setEditGrupoNombre(e.target.value)}
                        required
                        className="bg-[#242424] border border-[#2e2e2e] text-[#f0f0f0] text-sm rounded-lg px-3 py-1.5 focus-visible:outline-none focus-visible:border-[#E85D24]"
                      />
                    </div>
                    <div>
                      <label htmlFor={`edit-grupo-capacity-${grupo.id}`} className="block text-xs text-[#888888] mb-1">Capacidad</label>
                      <input
                        id={`edit-grupo-capacity-${grupo.id}`}
                        type="number"
                        min="1"
                        value={editGrupoCapacity}
                        onChange={e => setEditGrupoCapacity(e.target.value)}
                        placeholder="Sin límite"
                        className="bg-[#242424] border border-[#2e2e2e] text-[#f0f0f0] text-sm rounded-lg px-3 py-1.5 w-32 focus-visible:outline-none focus-visible:border-[#E85D24]"
                      />
                    </div>
                    <button
                      type="submit"
                      disabled={savingGrupo || !editGrupoNombre.trim()}
                      className="px-3 py-1.5 text-sm bg-[#E85D24] text-white rounded-lg disabled:opacity-50"
                    >
                      {savingGrupo ? 'Guardando...' : 'Guardar'}
                    </button>
                    <button
                      type="button"
                      onClick={() => setEditingGrupoId(null)}
                      className="px-3 py-1.5 text-sm border border-[#2e2e2e] text-[#888888] rounded-lg"
                    >
                      Cancelar
                    </button>
                  </form>
                ) : (
                  <div className="flex items-center justify-between">
                    <div>
                      <span className="text-[#f0f0f0] font-medium">{grupo.nombre}</span>
                      <span className="ml-2 text-xs text-[#888888]">
                        {grupo.member_count}/{grupo.capacity ?? '∞'} miembros
                      </span>
                    </div>
                    <div className="flex gap-3">
                      <button
                        type="button"
                        onClick={() => handleStartEditGrupo(grupo)}
                        className="text-sm text-[#f0f0f0] hover:text-[#E85D24] transition-colors"
                      >
                        Editar
                      </button>
                      <button
                        type="button"
                        onClick={() => setDeleteGrupoId(grupo.id)}
                        className="text-sm text-red-400 hover:underline"
                      >
                        Eliminar
                      </button>
                    </div>
                  </div>
                )}
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Confirm modals */}
      <ConfirmModal
        isOpen={showDeleteCurso}
        title="Eliminar curso"
        message="¿Eliminar este curso y todas sus actividades? Esta acción no se puede deshacer."
        isLoading={deletingCurso}
        onConfirm={handleDeleteCurso}
        onCancel={() => setShowDeleteCurso(false)}
      />
      <ConfirmModal
        isOpen={!!deleteActividadId}
        title="Eliminar actividad"
        message="¿Eliminar esta actividad? Esta acción no se puede deshacer."
        isLoading={deletingActividad}
        onConfirm={handleDeleteActividad}
        onCancel={() => setDeleteActividadId(null)}
      />
      <ConfirmModal
        isOpen={!!deleteGrupoId}
        title="Eliminar grupo"
        message="¿Eliminar este grupo? Esta acción no se puede deshacer."
        isLoading={deletingGrupo}
        onConfirm={handleDeleteGrupo}
        onCancel={() => setDeleteGrupoId(null)}
      />
    </div>
  )
}
