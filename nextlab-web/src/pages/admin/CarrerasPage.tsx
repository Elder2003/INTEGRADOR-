import { useState, useEffect } from 'react'
import { getCarreras, createCarrera, updateCarrera, deleteCarrera } from '../../api/admin'
import type { Carrera } from '../../types/academic'
import { ApiError } from '../../types'
import { LoadingSpinner } from '../../components/common/LoadingSpinner'
import { ErrorMessage } from '../../components/common/ErrorMessage'
import { ConfirmModal } from '../../components/common/ConfirmModal'

export function CarrerasPage() {
  const [items, setItems] = useState<Carrera[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [actionError, setActionError] = useState<string | null>(null)

  const [showCreateForm, setShowCreateForm] = useState(false)
  const [createCode, setCreateCode] = useState('')
  const [createName, setCreateName] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)

  const [editingId, setEditingId] = useState<string | null>(null)
  const [editName, setEditName] = useState('')

  const [showConfirm, setShowConfirm] = useState(false)
  const [selected, setSelected] = useState<Carrera | null>(null)

  useEffect(() => {
    getCarreras()
      .then(setItems)
      .catch((err: ApiError) => setError(err.message))
      .finally(() => setLoading(false))
  }, [])

  async function handleCreate() {
    if (!createCode.trim() || !createName.trim() || isSubmitting) return
    setIsSubmitting(true)
    setActionError(null)
    try {
      const created = await createCarrera({ code: createCode.trim(), name: createName.trim() })
      setItems(prev => [created, ...prev])
      setCreateCode('')
      setCreateName('')
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
      const updated = await updateCarrera(id, { name: editName.trim() })
      setItems(prev => prev.map(item => item.id === id ? updated : item))
      setEditingId(null)
    } catch (err) {
      setActionError((err as ApiError).message)
    } finally {
      setIsSubmitting(false)
    }
  }

  function openConfirm(carrera: Carrera) {
    setSelected(carrera)
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
      await deleteCarrera(selected.id)
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
        <h1 className="text-2xl font-semibold text-[#f0f0f0]">Carreras</h1>
        {!showCreateForm && (
          <button
            type="button"
            onClick={() => setShowCreateForm(true)}
            className="px-4 py-2 text-sm bg-[#E85D24] text-white rounded-lg hover:bg-[#d4521f] transition-colors"
          >
            Nueva Carrera
          </button>
        )}
      </div>

      {error && <ErrorMessage message={error} />}
      {actionError && <ErrorMessage message={actionError} onDismiss={() => setActionError(null)} />}

      {showCreateForm && (
        <div className="mb-6 p-4 bg-[#1a1a1a] border border-[#2e2e2e] rounded-lg space-y-3">
          <h2 className="text-sm font-semibold text-[#f0f0f0]">Nueva Carrera</h2>
          <div className="flex gap-3">
            <input
              type="text"
              placeholder="Código (ej: IS)"
              value={createCode}
              onChange={e => setCreateCode(e.target.value)}
              className="flex-1 bg-[#111111] border border-[#2e2e2e] text-[#f0f0f0] rounded-lg px-3 py-2 text-sm focus-visible:outline-none focus-visible:border-[#E85D24]"
            />
            <input
              type="text"
              placeholder="Nombre"
              value={createName}
              onChange={e => setCreateName(e.target.value)}
              className="flex-1 bg-[#111111] border border-[#2e2e2e] text-[#f0f0f0] rounded-lg px-3 py-2 text-sm focus-visible:outline-none focus-visible:border-[#E85D24]"
            />
          </div>
          <div className="flex gap-2">
            <button
              type="button"
              onClick={handleCreate}
              disabled={!createCode.trim() || !createName.trim() || isSubmitting}
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
        </div>
      )}

      {items.length === 0 ? (
        <div className="text-center py-16 text-[#888888]">
          <p className="mb-4">No hay carreras. ¡Crea la primera!</p>
          {!showCreateForm && (
            <button
              type="button"
              onClick={() => setShowCreateForm(true)}
              className="px-4 py-2 text-sm bg-[#E85D24] text-white rounded-lg hover:bg-[#d4521f] transition-colors"
            >
              Nueva Carrera
            </button>
          )}
        </div>
      ) : (
        <table className="w-full text-sm bg-[#1a1a1a] border border-[#2e2e2e] rounded-lg overflow-hidden">
          <thead>
            <tr className="border-b border-[#2e2e2e] text-[#888888]">
              <th className="text-left px-4 py-3">Código</th>
              <th className="text-left px-4 py-3">Nombre</th>
              <th className="px-4 py-3" />
            </tr>
          </thead>
          <tbody>
            {items.map(carrera => (
              <tr key={carrera.id} className="border-b border-[#2e2e2e] last:border-0">
                {editingId === carrera.id ? (
                  <>
                    <td className="px-4 py-3 text-[#888888]">{carrera.code}</td>
                    <td className="px-4 py-3">
                      <input
                        type="text"
                        value={editName}
                        onChange={e => setEditName(e.target.value)}
                        className="w-full bg-[#111111] border border-[#2e2e2e] text-[#f0f0f0] rounded px-2 py-1 text-sm focus-visible:outline-none focus-visible:border-[#E85D24]"
                      />
                    </td>
                    <td className="px-4 py-3">
                      <div className="flex gap-2 justify-end">
                        <button
                          type="button"
                          onClick={() => handleSaveEdit(carrera.id)}
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
                    <td className="px-4 py-3 text-[#f0f0f0] font-mono">{carrera.code}</td>
                    <td className="px-4 py-3 text-[#f0f0f0]">{carrera.name}</td>
                    <td className="px-4 py-3">
                      <div className="flex gap-2 justify-end">
                        <button
                          type="button"
                          onClick={() => { setEditingId(carrera.id); setEditName(carrera.name) }}
                          className="px-3 py-1 text-xs border border-[#2e2e2e] text-[#f0f0f0] rounded hover:border-[#888888] transition-colors"
                        >
                          Editar
                        </button>
                        <button
                          type="button"
                          onClick={() => openConfirm(carrera)}
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
        title="Desactivar carrera"
        message={`¿Desactivar ${selected?.name}? Solo es posible si no tiene materias activas.`}
        confirmLabel="Desactivar"
        isLoading={isSubmitting}
        onConfirm={handleDeactivate}
        onCancel={closeConfirm}
      />
    </div>
  )
}
