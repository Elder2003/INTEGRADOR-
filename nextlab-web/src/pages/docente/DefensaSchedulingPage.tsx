import { useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { listDefensaSlots, assignDefensaSlot } from '../../api/grading'
import type { DefensaSlotsResponse } from '../../types/academic'
import { LoadingSpinner } from '../../components/common/LoadingSpinner'
import { ErrorMessage } from '../../components/common/ErrorMessage'

interface RowState {
  scheduledAt: string
  location: string
  saving: boolean
  saved: boolean
  errorMsg: string | null
}

export function DefensaSchedulingPage() {
  const { cid, aid } = useParams<{ cid: string; aid: string }>()
  const [data, setData] = useState<DefensaSlotsResponse | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [rowState, setRowState] = useState<Record<string, RowState>>({})

  useEffect(() => {
    if (!cid || !aid) return
    listDefensaSlots(cid, aid)
      .then(res => {
        setData(res)
        const initial: Record<string, RowState> = {}
        for (const entry of res.entries) {
          initial[entry.submission.id] = {
            scheduledAt: entry.slot?.scheduled_at ?? '',
            location: entry.slot?.location ?? '',
            saving: false,
            saved: false,
            errorMsg: null,
          }
        }
        setRowState(initial)
      })
      .catch((err: Error) => setError(err.message))
      .finally(() => setLoading(false))
  }, [cid, aid])

  function updateRow(id: string, patch: Partial<RowState>) {
    setRowState(prev => ({ ...prev, [id]: { ...prev[id], ...patch } }))
  }

  async function handleSave(submissionId: string) {
    const row = rowState[submissionId]
    if (!row || !row.scheduledAt) {
      updateRow(submissionId, { errorMsg: 'La fecha y hora son obligatorias.' })
      return
    }
    updateRow(submissionId, { saving: true, errorMsg: null, saved: false })
    try {
      await assignDefensaSlot(submissionId, row.scheduledAt, row.location || null)
      updateRow(submissionId, { saving: false, saved: true })
      setTimeout(() => updateRow(submissionId, { saved: false }), 3000)
    } catch (err: unknown) {
      updateRow(submissionId, { saving: false, errorMsg: (err as Error).message })
    }
  }

  if (loading) return <LoadingSpinner />

  return (
    <div className="p-8 max-w-5xl space-y-6">
      <div>
        <Link
          to={`/docente/cursos/${cid}/actividades/${aid}`}
          className="text-[#888888] hover:text-[#E85D24] text-sm transition-colors"
        >
          ← Actividad
        </Link>
        <h1 className="text-2xl font-semibold text-[#f0f0f0] mt-4">Programar Defensas</h1>
      </div>

      {error && <ErrorMessage message={error} />}

      {!error && data && (
        data.entries.length === 0 ? (
          <p className="text-[#888888]">No hay entregas registradas para esta actividad.</p>
        ) : (
          <div className="bg-[#1a1a1a] border border-[#2e2e2e] rounded-xl overflow-hidden">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-[#2e2e2e] text-left">
                  <th className="px-4 py-3 text-[#888888] font-medium">Estudiante</th>
                  <th className="px-4 py-3 text-[#888888] font-medium">Email</th>
                  <th className="px-4 py-3 text-[#888888] font-medium">Fecha y Hora</th>
                  <th className="px-4 py-3 text-[#888888] font-medium">Ubicación</th>
                  <th className="px-4 py-3 text-[#888888] font-medium">Acción</th>
                </tr>
              </thead>
              <tbody>
                {data.entries.map(entry => {
                  const id = entry.submission.id
                  const row = rowState[id] ?? { scheduledAt: '', location: '', saving: false, saved: false, errorMsg: null }
                  return (
                    <tr key={id} className="border-b border-[#2e2e2e] last:border-0">
                      <td className="px-4 py-3 text-[#f0f0f0]">{entry.submission.student.name}</td>
                      <td className="px-4 py-3 text-[#888888]">{entry.submission.student.email}</td>
                      <td className="px-4 py-3">
                        <input
                          type="datetime-local"
                          aria-label={`Fecha y hora de defensa para ${entry.submission.student.name}`}
                          value={row.scheduledAt}
                          onChange={e => updateRow(id, { scheduledAt: e.target.value })}
                          className="bg-[#111111] border border-[#2e2e2e] rounded px-2 py-1 text-[#f0f0f0] text-xs focus:border-[#E85D24] focus:outline-none"
                        />
                      </td>
                      <td className="px-4 py-3">
                        <input
                          type="text"
                          aria-label={`Ubicación de defensa para ${entry.submission.student.name}`}
                          value={row.location}
                          onChange={e => updateRow(id, { location: e.target.value })}
                          placeholder="Aula / Link (opcional)"
                          className="bg-[#111111] border border-[#2e2e2e] rounded px-2 py-1 text-[#f0f0f0] text-xs focus:border-[#E85D24] focus:outline-none w-full"
                        />
                      </td>
                      <td className="px-4 py-3">
                        <div className="flex items-center gap-2">
                          <button
                            type="button"
                            onClick={() => handleSave(id)}
                            disabled={row.saving}
                            className="px-3 py-1 bg-[#E85D24] text-white text-xs rounded hover:bg-[#d4521f] disabled:opacity-50 transition-colors"
                          >
                            {row.saving ? 'Guardando...' : 'Guardar'}
                          </button>
                          {row.saved && <span className="text-green-400 text-xs">✓</span>}
                        </div>
                        {row.errorMsg && (
                          <p className="text-red-400 text-xs mt-1">{row.errorMsg}</p>
                        )}
                      </td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          </div>
        )
      )}
    </div>
  )
}
