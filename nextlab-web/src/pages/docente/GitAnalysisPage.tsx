import { useEffect, useState } from 'react'
import { Link, useLocation, useParams } from 'react-router-dom'
import { getGitAnalysis } from '../../api/gitAnalysis'
import type { GitAnalysisReport } from '../../types/academic'
import { LoadingSpinner } from '../../components/common/LoadingSpinner'
import { ErrorMessage } from '../../components/common/ErrorMessage'

function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString('es-BO', {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
  })
}

export function GitAnalysisPage() {
  const { cid, aid, subid } = useParams<{ cid: string; aid: string; subid: string }>()
  const location = useLocation()
  const routePrefix = location.pathname.startsWith('/docente') ? '/docente' : '/codocente'
  const backPath = `${routePrefix}/cursos/${cid}/actividades/${aid}/submissions/${subid}`

  const [data, setData] = useState<GitAnalysisReport | null>(null)
  const [loading, setLoading] = useState(true)
  const [fetchError, setFetchError] = useState<string | null>(null)

  useEffect(() => {
    if (!subid) return
    getGitAnalysis(subid)
      .then(setData)
      .catch((err: Error) => setFetchError(err.message))
      .finally(() => setLoading(false))
  }, [subid])

  if (loading) return <LoadingSpinner />

  return (
    <div className="p-8 max-w-4xl space-y-6">
      <div>
        <Link to={backPath} className="text-sm text-[#888888] hover:text-[#E85D24] transition-colors">
          ← Entrega
        </Link>
        <h1 className="text-2xl font-semibold text-[#f0f0f0] mt-4">Análisis Git</h1>
      </div>

      {fetchError && <ErrorMessage message={fetchError} />}

      {!fetchError && data && data.status === 'pending' && (
        <div className="bg-[#1a1a1a] border border-[#2e2e2e] rounded-lg p-4">
          <p className="text-[#888888] text-sm">Análisis en curso. Vuelve a revisar en unos minutos.</p>
        </div>
      )}

      {!fetchError && data && data.status === 'error' && (
        <div className="bg-red-900/20 border border-red-800 rounded-lg p-4">
          <p className="text-red-400 text-sm">{data.error_message ?? 'El análisis encontró un error.'}</p>
        </div>
      )}

      {!fetchError && data && data.status === 'done' && (
        <div className="space-y-6">
          <div className="grid grid-cols-2 sm:grid-cols-4 gap-4">
            <div className="bg-[#1a1a1a] border border-[#2e2e2e] rounded-lg p-4">
              <p className="text-xs text-[#888888] uppercase tracking-wide mb-1">Commits</p>
              <p className="text-2xl font-semibold text-[#f0f0f0]">{data.commit_count ?? 0}</p>
            </div>
            {data.date_from && data.date_to && (
              <div className="bg-[#1a1a1a] border border-[#2e2e2e] rounded-lg p-4 col-span-2">
                <p className="text-xs text-[#888888] uppercase tracking-wide mb-1">Rango de fechas</p>
                <p className="text-sm text-[#f0f0f0]">{formatDate(data.date_from)} — {formatDate(data.date_to)}</p>
              </div>
            )}
          </div>

          {data.authors && data.authors.length > 0 && (
            <div className="bg-[#1a1a1a] border border-[#2e2e2e] rounded-lg p-4">
              <p className="text-xs text-[#888888] uppercase tracking-wide mb-2">Autores</p>
              <ul className="space-y-1">
                {data.authors.map(author => (
                  <li key={author} className="text-sm text-[#f0f0f0] font-mono">{author}</li>
                ))}
              </ul>
            </div>
          )}

          {data.files && data.files.length > 0 ? (
            <div className="bg-[#1a1a1a] border border-[#2e2e2e] rounded-xl overflow-hidden">
              <p className="px-4 py-3 text-xs text-[#888888] uppercase tracking-wide border-b border-[#2e2e2e]">
                Archivos modificados ({data.files.length})
              </p>
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-[#2e2e2e] text-left">
                    <th className="px-4 py-2 text-[#888888] font-medium">Ruta</th>
                    <th className="px-4 py-2 text-[#888888] font-medium">Adiciones</th>
                    <th className="px-4 py-2 text-[#888888] font-medium">Eliminaciones</th>
                  </tr>
                </thead>
                <tbody>
                  {data.files.map(file => (
                    <tr key={file.path} className="border-b border-[#2e2e2e] last:border-0">
                      <td className="px-4 py-2 text-[#f0f0f0] font-mono text-xs break-all">{file.path}</td>
                      <td className="px-4 py-2 text-green-400">+{file.additions}</td>
                      <td className="px-4 py-2 text-red-400">-{file.deletions}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : (
            <p className="text-[#888888] text-sm">Sin archivos modificados registrados.</p>
          )}
        </div>
      )}
    </div>
  )
}
