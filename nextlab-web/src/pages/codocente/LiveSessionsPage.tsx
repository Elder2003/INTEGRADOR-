import { useEffect, useRef, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { listActiveSessions } from '../../api/tracing'
import type { ActiveSessionsResponse } from '../../types/academic'
import { LoadingSpinner } from '../../components/common/LoadingSpinner'
import { ErrorMessage } from '../../components/common/ErrorMessage'

export function CodocenteLiveSessionsPage() {
  const { cid, aid } = useParams<{ cid: string; aid: string }>()
  const navigate = useNavigate()

  const [data, setData] = useState<ActiveSessionsResponse | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null)

  const backPath = `/codocente/cursos/${cid}/actividades/${aid}`

  function fetchSessions() {
    return listActiveSessions(cid!, aid!)
      .then(setData)
      .catch(() => {})
  }

  useEffect(() => {
    if (!cid || !aid) return
    listActiveSessions(cid, aid)
      .then(setData)
      .catch((e: Error) => setError(e.message))
      .finally(() => setLoading(false))

    intervalRef.current = setInterval(fetchSessions, 5000)
    return () => {
      if (intervalRef.current) clearInterval(intervalRef.current)
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [cid, aid])

  if (loading) return <LoadingSpinner />

  return (
    <div className="p-8 max-w-3xl space-y-6">
      <Link
        to={backPath}
        className="text-sm text-[#888888] hover:text-[#E85D24] transition-colors"
      >
        ← Actividad
      </Link>

      <h1 className="text-2xl font-semibold text-[#f0f0f0]">Sesiones en Vivo</h1>

      {error && <ErrorMessage message={error} />}

      {data?.sessions.length === 0 ? (
        <p className="text-[#888888]">No hay sesiones activas en este momento.</p>
      ) : (
        <div className="bg-[#1a1a1a] border border-[#2e2e2e] rounded-lg overflow-hidden">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-[#2e2e2e] text-left">
                <th className="py-3 px-4 text-[#888888] font-medium">Estudiante</th>
                <th className="py-3 px-4 text-[#888888] font-medium">Email</th>
                <th className="py-3 px-4 text-[#888888] font-medium">Inicio</th>
                <th className="py-3 px-4 text-[#888888] font-medium">Eventos</th>
              </tr>
            </thead>
            <tbody>
              {data?.sessions.map((entry) => (
                <tr
                  key={entry.session_id}
                  onClick={() =>
                    navigate(
                      `/codocente/cursos/${cid}/actividades/${aid}/sessions/${entry.session_id}/live`,
                    )
                  }
                  className="border-b border-[#2e2e2e] cursor-pointer hover:bg-[#242424] transition-colors"
                >
                  <td className="py-3 px-4 text-[#f0f0f0]">{entry.student.name}</td>
                  <td className="py-3 px-4 text-[#888888]">{entry.student.email}</td>
                  <td className="py-3 px-4 text-[#888888] text-xs">
                    {new Date(entry.started_at).toLocaleString('es-BO')}
                  </td>
                  <td className="py-3 px-4 text-[#f0f0f0]">{entry.event_count}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}
