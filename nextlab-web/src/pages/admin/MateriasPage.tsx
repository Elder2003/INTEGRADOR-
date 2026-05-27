import { useState, useEffect } from 'react'
import { Link } from 'react-router-dom'
import { getAdminMaterias, createMateria, updateMateria, deleteMateria } from '../../api/admin'
import { getCarreras } from '../../api/admin'
import type { Materia, Carrera } from '../../types/academic'
import { ApiError } from '../../types'
import { LoadingSpinner } from '../../components/common/LoadingSpinner'
import { ErrorMessage } from '../../components/common/ErrorMessage'
import { ConfirmModal } from '../../components/common/ConfirmModal'

const SEMESTERS = Array.from({ length: 10 }, (_, i) => i + 1)

export function MateriasPage() {
  const [items, setItems] = useState<Materia[]>([])
  const [carreras, setCarreras] = useState<Carrera[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [actionError, setActionError] = useState<string | null>(null)

  const [showCreateForm, setShowCreateForm] = useState(false)
  const [createCode, setCreateCode] = useState('')
  const [createName, setCreateName] = useState('')
  const [createSemester, setCreateSemester] = useState('1')
  const [createCarreraId, setCreateCarreraId] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)

  const [editingId, setEditingId] = useState<string | null>(null)
  const [editName, setEditName] = useState('')
  const [editSemester, setEditSemester] = useState('1')

  const [showConfirm, setShowConfirm] = useState(false)
  const [selected, setSelected] = useState<Materia | null>(null)

  useEffect(() => {
    Promise.all([getAdminMaterias(), getCarreras()])
      .then(([mats, cars]) => {
        setItems(mats)
        setCarreras(cars)
        if (cars.length > 0) setCreateCarreraId(cars[0].id)
      })
      .catch((err: ApiError) => setError(err.message))
      .finally(() => setLoading(false))
  }, [])

  function carreraName(carreraId: string): string {
    return carreras.find(c => c.id === carreraId)?.name ?? carreraId
  }

  async function handleCreate() {
    if (!createCode.trim() || !createName.trim() || !createCarreraId || isSubmitting) return
    setIsSubmitting(true)
    setActionError(null)
    try {
      const created = await createMateria({
        code: createCode.trim(),
        name: createName.trim(),
        semester: Number(createSemester),
        carrera_id: createCarreraId,
      })
      setItems(prev => [created, ...prev])
      setCreateCode('')
      setCreateName('')
      setCreateSemester('1')
      setShowCreateForm(false)
    } catch (err) {
      setActionError((err as ApiError).message)
    } finally {
      setIsSubmitting(false)
    }
  }

  async function handleSaveEdit(id: string) {
    setIsSubmitting(true)
    setActionError(null)
    try {
      const updated = await updateMateria(id, { name: editName.trim(), semester: Number(editSemester) })
      setItems(prev => prev.map(item => item.id === id ? updated : item))
      setEditingId(null)
    } catch (err) {
      setActionError((err as ApiError).message)
    } finally {
      setIsSubmitting(false)
    }
  }

  function openConfirm(materia: Materia) {
    setSelected(materia)
    setShowConfirm(true)
  }

  function closeConfirm() {
    setShowConfirm(false)
    setSelected(null)
  }

  async function handleDeactivate() {
    if (!selected) return
    setIsSubmitting(true)
    try {
      await deleteMateria(selected.id)
      setItems(prev => prev.filter(item => item.id !== selected.id))
      closeConfirm()
    } catch (err) {
      closeConfirm()
      setActionError((err as ApiError).message)
    } finally {
      setIsSubmitting(false)
    }
  }

  if (loading) return <LoadingSpinner />

  return (
    <div className="p-8">
      <div className="flex items-center justify-between mb-4">
        <h1 className="text-2xl font-semibold text-[#f0f0f0]">Materias</h1>
        {!showCreateForm && (
          <button
            type="button"
            onClick={() => setShowCreateForm(true)}
            className="px-4 py-2 text-sm bg-[#E85D24] text-white rounded-lg hover:bg-[#d4521f] transition-colors"
          >
            Nueva Materia
          </button>
        )}
      </div>

      {error && <ErrorMessage message={error} />}
      {actionError && <ErrorMessage message={actionError} onDismiss={() => setActionError(null)} />}

      {showCreateForm && (
        <div className="mb-6 p-4 bg-[#1a1a1a] border border-[#2e2e2e] rounded-lg space-y-3">
          <h2 className="text-sm font-semibold text-[#f0f0f0]">Nueva Materia</h2>
          <div className="grid grid-cols-2 gap-3">
            <input
              type="text"
              placeholder="Código (ej: MAT101)"
              value={createCode}
              onChange={e => setCreateCode(e.target.value)}
              className="bg-[#111111] border border-[#2e2e2e] text-[#f0f0f0] rounded-lg px-3 py-2 text-sm focus-visible:outline-none focus-visible:border-[#E85D24]"
            />
            <input
              type="text"
              placeholder="Nombre"
              value={createName}
              onChange={e => setCreateName(e.target.value)}
              className="bg-[#111111] border border-[#2e2e2e] text-[#f0f0f0] rounded-lg px-3 py-2 text-sm focus-visible:outline-none focus-visible:border-[#E85D24]"
            />
            <select
              value={createSemester}
              onChange={e => setCreateSemester(e.target.value)}
              className="bg-[#111111] border border-[#2e2e2e] text-[#f0f0f0] rounded-lg px-3 py-2 text-sm focus-visible:outline-none focus-visible:border-[#E85D24]"
            >
              {SEMESTERS.map(s => (
                <option key={s} value={s}>{s}° Semestre</option>
              ))}
            </select>
            {carreras.length > 0 ? (
              <select
                value={createCarreraId}
                onChange={e => setCreateCarreraId(e.target.value)}
                className="bg-[#111111] border border-[#2e2e2e] text-[#f0f0f0] rounded-lg px-3 py-2 text-sm focus-visible:outline-none focus-visible:border-[#E85D24]"
              >
                {carreras.map(c => (
                  <option key={c.id} value={c.id}>{c.name}</option>
                ))}
              </select>
            ) : (
              <div className="flex items-center gap-2 text-sm text-[#888888]">
                <span>No hay carreras.</span>
                <Link to="/admin/carreras" className="text-[#E85D24] hover:underline">Crea una primero.</Link>
              </div>
            )}
          </div>
          <div className="flex gap-2">
            <button
              type="button"
              onClick={handleCreate}
              disabled={!createCode.trim() || !createName.trim() || !createCarreraId || carreras.length === 0 || isSubmitting}
              className="px-4 py-2 text-sm bg-[#E85D24] text-white rounded-lg hover:bg-[#d4521f] disabled:opacity-50 transition-colors"
            >
              {isSubmitting ? 'Guardando...' : 'Crear'}
            </button>
            <button
              type="button"
              onClick={() => { setShowCreateForm(false); setCreateCode(''); setCreateName('') }}
              className="px-4 py-2 text-sm border border-[#2e2e2e] text-[#f0f0f0] rounded-lg hover:border-[#888888] transition-colors"
            >
              Cancelar
            </button>
          </div>
          {carreras.length === 0 && (
            <p className="text-xs text-[#888888]">No hay carreras disponibles. Crea una primero.</p>
          )}
        </div>
      )}

      {items.length === 0 ? (
        <div className="text-center py-16 text-[#888888]">
          <p className="mb-4">No hay materias. ¡Crea la primera!</p>
          {!showCreateForm && (
            <button
              type="button"
              onClick={() => setShowCreateForm(true)}
              className="px-4 py-2 text-sm bg-[#E85D24] text-white rounded-lg hover:bg-[#d4521f] transition-colors"
            >
              Nueva Materia
            </button>
          )}
        </div>
      ) : (
        <table className="w-full text-sm bg-[#1a1a1a] border border-[#2e2e2e] rounded-lg overflow-hidden">
          <thead>
            <tr className="border-b border-[#2e2e2e] text-[#888888]">
              <th className="text-left px-4 py-3">Código</th>
              <th className="text-left px-4 py-3">Nombre</th>
              <th className="text-left px-4 py-3">Semestre</th>
              <th className="text-left px-4 py-3">Carrera</th>
              <th className="px-4 py-3" />
            </tr>
          </thead>
          <tbody>
            {items.map(materia => (
              <tr key={materia.id} className="border-b border-[#2e2e2e] last:border-0">
                {editingId === materia.id ? (
                  <>
                    <td className="px-4 py-3 text-[#888888] font-mono">{materia.code}</td>
                    <td className="px-4 py-3">
                      <input
                        type="text"
                        value={editName}
                        onChange={e => setEditName(e.target.value)}
                        className="w-full bg-[#111111] border border-[#2e2e2e] text-[#f0f0f0] rounded px-2 py-1 text-sm focus-visible:outline-none focus-visible:border-[#E85D24]"
                      />
                    </td>
                    <td className="px-4 py-3">
                      <select
                        value={editSemester}
                        onChange={e => setEditSemester(e.target.value)}
                        className="bg-[#111111] border border-[#2e2e2e] text-[#f0f0f0] rounded px-2 py-1 text-sm focus-visible:outline-none focus-visible:border-[#E85D24]"
                      >
                        {SEMESTERS.map(s => (
                          <option key={s} value={s}>{s}°</option>
                        ))}
                      </select>
                    </td>
                    <td className="px-4 py-3 text-[#888888]">{carreraName(materia.carrera_id)}</td>
                    <td className="px-4 py-3">
                      <div className="flex gap-2 justify-end">
                        <button
                          type="button"
                          onClick={() => handleSaveEdit(materia.id)}
                          disabled={isSubmitting || !editName.trim()}
                          className="px-3 py-1 text-xs bg-[#E85D24] text-white rounded hover:bg-[#d4521f] disabled:opacity-50 transition-colors"
                        >
                          Guardar
                        </button>
                        <button
                          type="button"
                          onClick={() => setEditingId(null)}
                          className="px-3 py-1 text-xs border border-[#2e2e2e] text-[#f0f0f0] rounded hover:border-[#888888] transition-colors"
                        >
                          Cancelar
                        </button>
                      </div>
                    </td>
                  </>
                ) : (
                  <>
                    <td className="px-4 py-3 text-[#f0f0f0] font-mono">{materia.code}</td>
                    <td className="px-4 py-3 text-[#f0f0f0]">{materia.name}</td>
                    <td className="px-4 py-3 text-[#f0f0f0]">{materia.semester}°</td>
                    <td className="px-4 py-3 text-[#f0f0f0]">{carreraName(materia.carrera_id)}</td>
                    <td className="px-4 py-3">
                      <div className="flex gap-2 justify-end">
                        <button
                          type="button"
                          onClick={() => { setEditingId(materia.id); setEditName(materia.name); setEditSemester(String(materia.semester)) }}
                          className="px-3 py-1 text-xs border border-[#2e2e2e] text-[#f0f0f0] rounded hover:border-[#888888] transition-colors"
                        >
                          Editar
                        </button>
                        <button
                          type="button"
                          onClick={() => openConfirm(materia)}
                          className="px-3 py-1 text-xs border border-red-800 text-red-400 rounded hover:border-red-600 transition-colors"
                        >
                          Desactivar
                        </button>
                      </div>
                    </td>
                  </>
                )}
              </tr>
            ))}
          </tbody>
        </table>
      )}

      <ConfirmModal
        isOpen={showConfirm}
        title="Desactivar materia"
        message={`¿Desactivar ${selected?.name}? Solo es posible si no tiene cursos activos.`}
        confirmLabel="Desactivar"
        isLoading={isSubmitting}
        onConfirm={handleDeactivate}
        onCancel={closeConfirm}
      />
    </div>
  )
}
