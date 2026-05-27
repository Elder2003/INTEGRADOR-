import { useState, useEffect } from 'react'
import { getStudentProfiles, createStudentProfile, updateStudentProfile, deactivateStudentProfile } from '../../api/admin'
import { getCarreras } from '../../api/admin'
import type { StudentProfile, Carrera } from '../../types/academic'
import { ApiError } from '../../types'
import { LoadingSpinner } from '../../components/common/LoadingSpinner'
import { ErrorMessage } from '../../components/common/ErrorMessage'
import { ConfirmModal } from '../../components/common/ConfirmModal'

const SEMESTERS = Array.from({ length: 10 }, (_, i) => i + 1)

export function StudentProfilesPage() {
  const [items, setItems] = useState<StudentProfile[]>([])
  const [carreras, setCarreras] = useState<Carrera[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [actionError, setActionError] = useState<string | null>(null)

  const [showCreateForm, setShowCreateForm] = useState(false)
  const [createEmail, setCreateEmail] = useState('')
  const [createCarreraId, setCreateCarreraId] = useState('')
  const [createSemester, setCreateSemester] = useState('1')
  const [createStudentCode, setCreateStudentCode] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)

  const [editingId, setEditingId] = useState<string | null>(null)
  const [editSemester, setEditSemester] = useState('1')
  const [editStudentCode, setEditStudentCode] = useState('')

  const [showConfirm, setShowConfirm] = useState(false)
  const [selected, setSelected] = useState<StudentProfile | null>(null)

  useEffect(() => {
    Promise.all([getStudentProfiles(), getCarreras()])
      .then(([profiles, cars]) => {
        setItems(profiles)
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
    if (!createEmail.trim() || !createCarreraId || isSubmitting) return
    setIsSubmitting(true)
    setActionError(null)
    try {
      const created = await createStudentProfile({
        email: createEmail.trim(),
        carrera_id: createCarreraId,
        semester: Number(createSemester),
        student_code: createStudentCode.trim() || undefined,
      })
      setItems(prev => [created, ...prev])
      setCreateEmail('')
      setCreateSemester('1')
      setCreateStudentCode('')
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
      const updated = await updateStudentProfile(id, {
        semester: Number(editSemester),
        student_code: editStudentCode.trim() || null,
      })
      setItems(prev => prev.map(item => item.id === id ? updated : item))
      setEditingId(null)
    } catch (err) {
      setActionError((err as ApiError).message)
    } finally {
      setIsSubmitting(false)
    }
  }

  function openConfirm(profile: StudentProfile) {
    setSelected(profile)
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
      await deactivateStudentProfile(selected.id)
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
        <h1 className="text-2xl font-semibold text-[#f0f0f0]">Perfiles de Estudiantes</h1>
        {!showCreateForm && (
          <button
            type="button"
            onClick={() => setShowCreateForm(true)}
            className="px-4 py-2 text-sm bg-[#E85D24] text-white rounded-lg hover:bg-[#d4521f] transition-colors"
          >
            Nuevo Perfil
          </button>
        )}
      </div>

      {error && <ErrorMessage message={error} />}
      {actionError && <ErrorMessage message={actionError} onDismiss={() => setActionError(null)} />}

      {showCreateForm && (
        <div className="mb-6 p-4 bg-[#1a1a1a] border border-[#2e2e2e] rounded-lg space-y-3">
          <h2 className="text-sm font-semibold text-[#f0f0f0]">Nuevo Perfil de Estudiante</h2>
          <div className="grid grid-cols-2 gap-3">
            <input
              type="email"
              placeholder="Email (@unifranz.edu.bo)"
              value={createEmail}
              onChange={e => setCreateEmail(e.target.value)}
              className="bg-[#111111] border border-[#2e2e2e] text-[#f0f0f0] rounded-lg px-3 py-2 text-sm focus-visible:outline-none focus-visible:border-[#E85D24]"
            />
            <input
              type="text"
              placeholder="Código de estudiante (opcional)"
              value={createStudentCode}
              onChange={e => setCreateStudentCode(e.target.value)}
              className="bg-[#111111] border border-[#2e2e2e] text-[#f0f0f0] rounded-lg px-3 py-2 text-sm focus-visible:outline-none focus-visible:border-[#E85D24]"
            />
            <select
              value={createCarreraId}
              onChange={e => setCreateCarreraId(e.target.value)}
              className="bg-[#111111] border border-[#2e2e2e] text-[#f0f0f0] rounded-lg px-3 py-2 text-sm focus-visible:outline-none focus-visible:border-[#E85D24]"
            >
              {carreras.length === 0 ? (
                <option value="">Sin carreras disponibles</option>
              ) : (
                carreras.map(c => (
                  <option key={c.id} value={c.id}>{c.name}</option>
                ))
              )}
            </select>
            <select
              value={createSemester}
              onChange={e => setCreateSemester(e.target.value)}
              className="bg-[#111111] border border-[#2e2e2e] text-[#f0f0f0] rounded-lg px-3 py-2 text-sm focus-visible:outline-none focus-visible:border-[#E85D24]"
            >
              {SEMESTERS.map(s => (
                <option key={s} value={s}>{s}° Semestre</option>
              ))}
            </select>
          </div>
          <div className="flex gap-2">
            <button
              type="button"
              onClick={handleCreate}
              disabled={!createEmail.trim() || !createCarreraId || isSubmitting}
              className="px-4 py-2 text-sm bg-[#E85D24] text-white rounded-lg hover:bg-[#d4521f] disabled:opacity-50 transition-colors"
            >
              {isSubmitting ? 'Guardando...' : 'Crear'}
            </button>
            <button
              type="button"
              onClick={() => { setShowCreateForm(false); setCreateEmail(''); setCreateStudentCode('') }}
              className="px-4 py-2 text-sm border border-[#2e2e2e] text-[#f0f0f0] rounded-lg hover:border-[#888888] transition-colors"
            >
              Cancelar
            </button>
          </div>
        </div>
      )}

      {items.length === 0 ? (
        <div className="text-center py-16 text-[#888888]">
          <p className="mb-4">No hay perfiles de estudiantes. ¡Crea el primero!</p>
          {!showCreateForm && (
            <button
              type="button"
              onClick={() => setShowCreateForm(true)}
              className="px-4 py-2 text-sm bg-[#E85D24] text-white rounded-lg hover:bg-[#d4521f] transition-colors"
            >
              Nuevo Perfil
            </button>
          )}
        </div>
      ) : (
        <table className="w-full text-sm bg-[#1a1a1a] border border-[#2e2e2e] rounded-lg overflow-hidden">
          <thead>
            <tr className="border-b border-[#2e2e2e] text-[#888888]">
              <th className="text-left px-4 py-3">Email</th>
              <th className="text-left px-4 py-3">Carrera</th>
              <th className="text-left px-4 py-3">Semestre</th>
              <th className="text-left px-4 py-3">Código</th>
              <th className="px-4 py-3" />
            </tr>
          </thead>
          <tbody>
            {items.map(profile => (
              <tr key={profile.id} className="border-b border-[#2e2e2e] last:border-0">
                {editingId === profile.id ? (
                  <>
                    <td className="px-4 py-3 text-[#888888]">{profile.email}</td>
                    <td className="px-4 py-3 text-[#888888]">{carreraName(profile.carrera_id)}</td>
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
                    <td className="px-4 py-3">
                      <input
                        type="text"
                        value={editStudentCode}
                        onChange={e => setEditStudentCode(e.target.value)}
                        placeholder="Código (opcional)"
                        className="w-full bg-[#111111] border border-[#2e2e2e] text-[#f0f0f0] rounded px-2 py-1 text-sm focus-visible:outline-none focus-visible:border-[#E85D24]"
                      />
                    </td>
                    <td className="px-4 py-3">
                      <div className="flex gap-2 justify-end">
                        <button
                          type="button"
                          onClick={() => handleSaveEdit(profile.id)}
                          disabled={isSubmitting}
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
                    <td className="px-4 py-3 text-[#f0f0f0]">{profile.email}</td>
                    <td className="px-4 py-3 text-[#f0f0f0]">{carreraName(profile.carrera_id)}</td>
                    <td className="px-4 py-3 text-[#f0f0f0]">{profile.semester}°</td>
                    <td className="px-4 py-3 text-[#888888]">{profile.student_code ?? '—'}</td>
                    <td className="px-4 py-3">
                      <div className="flex gap-2 justify-end">
                        <button
                          type="button"
                          onClick={() => {
                            setEditingId(profile.id)
                            setEditSemester(String(profile.semester))
                            setEditStudentCode(profile.student_code ?? '')
                          }}
                          className="px-3 py-1 text-xs border border-[#2e2e2e] text-[#f0f0f0] rounded hover:border-[#888888] transition-colors"
                        >
                          Editar
                        </button>
                        <button
                          type="button"
                          onClick={() => openConfirm(profile)}
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
        title="Desactivar perfil"
        message={`¿Desactivar el perfil de ${selected?.email}? El estudiante no podrá acceder hasta que sea reactivado.`}
        confirmLabel="Desactivar"
        isLoading={isSubmitting}
        onConfirm={handleDeactivate}
        onCancel={closeConfirm}
      />
    </div>
  )
}
