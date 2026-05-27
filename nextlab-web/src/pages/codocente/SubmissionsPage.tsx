import { useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { listSubmissions, listConfigChanges } from '../../api/submissions'
import type { SubmissionListResponse, ConfigChangesResponse } from '../../types/academic'
import { LoadingSpinner } from '../../components/common/LoadingSpinner'
import { ErrorMessage } from '../../components/common/ErrorMessage'
import { SubmissionTable } from '../../components/docente/SubmissionTable'

export function CodocenteSubmissionsPage() {
  const { cid, aid } = useParams<{ cid: string; aid: string }>()
  const [data, setData] = useState<SubmissionListResponse | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!cid || !aid) return
    Promise.all([listSubmissions(cid, aid), listConfigChanges(cid, aid)])
      .then(([submData]) => {
        setData(submData)
      })
      .catch((err: Error) => setError(err.message))
      .finally(() => setLoading(false))
  }, [cid, aid])

  if (loading) return <LoadingSpinner />

  return (
    <div className="p-8 max-w-5xl space-y-6">
      <div>
        <Link
          to="/codocente/dashboard"
          className="text-sm text-[#888888] hover:text-[#E85D24] transition-colors"
        >
          ← Dashboard
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
            userRole="co_docente"
          />
        </div>
      )}
    </div>
  )
}
