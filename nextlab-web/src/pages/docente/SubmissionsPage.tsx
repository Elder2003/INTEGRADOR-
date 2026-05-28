import { useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { listSubmissions, listConfigChanges } from '../../api/submissions'
import { getActividad } from '../../api/academic'
import { listActivityAppeals } from '../../api/appeals'
import type { SubmissionListResponse, ConfigChangesResponse, Actividad, ActividadFlags } from '../../types/academic'
import { LoadingSpinner } from '../../components/common/LoadingSpinner'
import { ErrorMessage } from '../../components/common/ErrorMessage'
import { SubmissionTable } from '../../components/docente/SubmissionTable'

export function SubmissionsPage() {
  const { cid, aid } = useParams<{ cid: string; aid: string }>()
  const [data, setData] = useState<SubmissionListResponse | null>(null)
  const [changes, setChanges] = useState<ConfigChangesResponse | null>(null)
  const [actividad, setActividad] = useState<Actividad | null>(null)
  const [pendingAppealSessionIds, setPendingAppealSessionIds] = useState<Set<string> | undefined>(undefined)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!cid || !aid) return
    Promise.all([
      listSubmissions(cid, aid),
      listConfigChanges(cid, aid),
      getActividad(aid),
      listActivityAppeals(aid).catch(() => []),
    ])
      .then(([submData, changesData, actData, appeals]) => {
        setData(submData)
        setChanges(changesData)
        setActividad(actData)
        const pending = new Set(
          appeals.filter((a) => a.status === 'pending').map((a) => a.session_id),
        )
        setPendingAppealSessionIds(pending)
      })
      .catch((err: Error) => setError(err.message))
      .finally(() => setLoading(false))
  }, [cid, aid])

  if (loading) return <LoadingSpinner />

  return (
    <div className="p-8 max-w-5xl space-y-8">
      <div>
        <Link
          to={`/docente/cursos/${cid}/actividades/${aid}`}
          className="text-sm text-[#888888] hover:text-[#E85D24] transition-colors"
        >
          ← Actividad
        </Link>
        <h1 className="text-2xl font-semibold text-[#f0f0f0] mt-4">Entregas</h1>
      </div>

      {error && <ErrorMessage message={error} />}

      {!error && (
        <div className="bg-[#1a1a1a] border border-[#2e2e2e] rounded-xl p-5">
          <SubmissionTable
            submissions={data?.submissions ?? []}
            cursoId={cid!}
            actividadId={aid!}
            userRole="docente"
            actividadTipo={actividad?.tipo}
            actividadFlags={actividad ? ({
              git_enabled: actividad.git_enabled,
              ai_enabled: actividad.ai_enabled,
              detection_enabled: actividad.detection_enabled,
            } satisfies ActividadFlags) : undefined}
            pendingAppealSessionIds={pendingAppealSessionIds}
          />
        </div>
      )}

      {!error && (
        <div>
          <h2 className="text-lg font-semibold text-[#f0f0f0] mb-4">Cambios de Configuración</h2>
          {changes?.changes.length ? (
            <div className="bg-[#1a1a1a] border border-[#2e2e2e] rounded-xl overflow-hidden">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-[#2e2e2e]">
                    <th className="text-left px-4 py-2 text-[#888888] font-medium">Quién</th>
                    <th className="text-left px-4 py-2 text-[#888888] font-medium">Campo</th>
                    <th className="text-left px-4 py-2 text-[#888888] font-medium">Anterior</th>
                    <th className="text-left px-4 py-2 text-[#888888] font-medium">Nuevo</th>
                    <th className="text-left px-4 py-2 text-[#888888] font-medium">Cuándo</th>
                  </tr>
                </thead>
                <tbody>
                  {changes.changes.map((ch) => (
                    <tr key={ch.id} className="border-b border-[#2e2e2e] last:border-0">
                      <td className="px-4 py-3 text-[#f0f0f0]">{ch.changed_by.name}</td>
                      <td className="px-4 py-3 text-[#888888]">{ch.field_name}</td>
                      <td className="px-4 py-3 text-[#888888] max-w-[160px] truncate">
                        {ch.previous_value}
                      </td>
                      <td className="px-4 py-3 text-[#f0f0f0] max-w-[160px] truncate">
                        {ch.new_value}
                      </td>
                      <td className="px-4 py-3 text-[#888888] whitespace-nowrap">
                        {new Date(ch.changed_at).toLocaleString('es-BO')}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : (
            <p className="text-[#888888]">Sin cambios de configuración registrados.</p>
          )}
        </div>
      )}
    </div>
  )
}
