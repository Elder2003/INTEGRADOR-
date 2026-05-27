import { useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { getTraceLog } from '../../api/tracing'
import type { TraceLogResponse } from '../../types/academic'
import { LoadingSpinner } from '../../components/common/LoadingSpinner'
import { ErrorMessage } from '../../components/common/ErrorMessage'
import { TraceEventTable } from '../../components/docente/TraceEventTable'

export function CodocenteTraceViewerPage() {
  const { cid, aid, subid } = useParams<{ cid: string; aid: string; subid: string }>()

  const backPath = `/codocente/cursos/${cid}/actividades/${aid}/submissions/${subid}`

  const [data, setData] = useState<TraceLogResponse | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!subid) return
    getTraceLog(subid)
      .then(setData)
      .catch((e: Error) => setError(e.message))
      .finally(() => setLoading(false))
  }, [subid])

  if (loading) return <LoadingSpinner />

  return (
    <div className="p-8 max-w-4xl space-y-6">
      <Link to={backPath} className="text-sm text-[#888888] hover:text-[#E85D24] transition-colors">
        ← Entrega
      </Link>

      <h1 className="text-2xl font-semibold text-[#f0f0f0]">Trazas de entrega</h1>

      {error ? (
        <ErrorMessage message={error} />
      ) : (
        <div className="bg-[#1a1a1a] border border-[#2e2e2e] rounded-lg p-4">
          <TraceEventTable events={data?.events ?? []} />
        </div>
      )}
    </div>
  )
}
