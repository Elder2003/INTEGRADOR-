import { useState } from 'react'
import { Link, useLocation, useParams } from 'react-router-dom'
import { downloadSubmission } from '../../api/submissions'
import type { SubmissionListItem } from '../../types/academic'
import { ErrorMessage } from '../../components/common/ErrorMessage'

function formatDateTime(iso: string): string {
  return new Date(iso).toLocaleString('es-BO', {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

export function SubmissionDetailPage() {
  const { cid, aid, sid } = useParams<{ cid: string; aid: string; sid: string }>()
  const location = useLocation()
  const item = location.state as SubmissionListItem | undefined

  const backPath = location.pathname.startsWith('/docente')
    ? `/docente/cursos/${cid}/actividades/${aid}/submissions`
    : `/codocente/cursos/${cid}/actividades/${aid}/submissions`

  const [downloading, setDownloading] = useState(false)
  const [downloadError, setDownloadError] = useState<string | null>(null)

  async function handleDownload() {
    if (!sid || !item) return
    setDownloading(true)
    setDownloadError(null)
    try {
      await downloadSubmission(sid, item.original_filename)
    } catch (err: unknown) {
      setDownloadError((err as Error).message)
    } finally {
      setDownloading(false)
    }
  }

  if (!item) {
    return (
      <div className="p-8 space-y-4">
        <ErrorMessage message="Navega desde la lista de entregas para ver el detalle." />
        <Link to={backPath} className="text-sm text-[#E85D24] hover:underline">
          ← Volver a la lista
        </Link>
      </div>
    )
  }

  return (
    <div className="p-8 max-w-2xl space-y-6">
      <div>
        <Link
          to={backPath}
          className="text-sm text-[#888888] hover:text-[#E85D24] transition-colors"
        >
          ← Volver a la lista
        </Link>
        <h1 className="text-2xl font-semibold text-[#f0f0f0] mt-4 break-all">
          {item.original_filename}
        </h1>
      </div>

      <div className="bg-[#1a1a1a] border border-[#2e2e2e] rounded-xl p-5 space-y-4">
        <div>
          <p className="text-xs text-[#888888] uppercase tracking-wide mb-1">Estudiante</p>
          <p className="text-[#f0f0f0] font-medium">{item.student.name}</p>
          <p className="text-[#888888] text-sm">{item.student.email}</p>
        </div>

        <div>
          <p className="text-xs text-[#888888] uppercase tracking-wide mb-1">Archivo</p>
          <p className="text-[#f0f0f0] text-sm break-all">{item.original_filename}</p>
          <p className="text-[#888888] text-sm">{(item.file_size_bytes / 1024).toFixed(0)} KB</p>
        </div>

        <div>
          <p className="text-xs text-[#888888] uppercase tracking-wide mb-1">Enviado</p>
          <p className="text-[#f0f0f0] text-sm">{formatDateTime(item.submitted_at)}</p>
        </div>

        <div className="flex flex-wrap gap-2">
          {item.is_late && (
            <span className="text-xs bg-yellow-400/10 text-yellow-400 px-2 py-0.5 rounded">
              Entrega tardía
            </span>
          )}
          {item.is_locked ? (
            <span className="text-xs bg-[#2e2e2e] text-[#888888] px-2 py-0.5 rounded">
              Bloqueada
            </span>
          ) : (
            item.edit_window_expires_at && (
              <span className="text-xs text-[#888888]">
                Ventana hasta: {formatDateTime(item.edit_window_expires_at)}
              </span>
            )
          )}
        </div>
      </div>

      {downloadError && <ErrorMessage message={downloadError} />}

      <button
        type="button"
        onClick={handleDownload}
        disabled={downloading}
        className="bg-[#E85D24] hover:bg-[#d4521f] text-white px-4 py-2 rounded-lg text-sm transition-colors disabled:opacity-50"
      >
        {downloading ? 'Descargando...' : 'Descargar'}
      </button>
    </div>
  )
}
