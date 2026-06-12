import { useState, useEffect } from 'react'
import { Link, useLocation, useParams } from 'react-router-dom'
import { downloadSubmission } from '../../api/submissions'
import type { SubmissionListItem, ActividadTipo, ActividadFlags } from '../../types/academic'
import { ErrorMessage } from '../../components/common/ErrorMessage'
import { useAuth } from '../../context/AuthContext'
import { GradePanel } from '../../components/docente/GradePanel'
import { StageSelector } from '../../components/docente/StageSelector'
import { getAppeal, reviewAppeal } from '../../api/appeals'
import type { Appeal, AppealDecision } from '../../types/academic'

function formatDateTime(iso: string): string {
  return new Date(iso).toLocaleString('es-BO', {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

const APPEAL_STATUS_LABEL: Record<string, string> = {
  pending: 'Pendiente',
  escalated: 'Derivada al administrador',
  resolved: 'Resuelta',
  admin_resolved: 'Resuelta por administrador',
}

const APPEAL_STATUS_CLASS: Record<string, string> = {
  pending: 'bg-yellow-400/10 text-yellow-400',
  escalated: 'bg-blue-400/10 text-blue-400',
  resolved: 'bg-green-400/10 text-green-400',
  admin_resolved: 'bg-green-400/10 text-green-400',
}

export function SubmissionDetailPage() {
  const { cid, aid, sid } = useParams<{ cid: string; aid: string; sid: string }>()
  const location = useLocation()
  const rawState = location.state as (SubmissionListItem & { actividadTipo?: ActividadTipo; actividadFlags?: ActividadFlags }) | { item: SubmissionListItem; actividadTipo?: ActividadTipo; actividadFlags?: ActividadFlags } | undefined
  const item: SubmissionListItem | undefined = rawState
    ? ('item' in rawState && rawState.item ? rawState.item : rawState as SubmissionListItem)
    : undefined
  const actividadTipo = rawState && 'actividadTipo' in rawState ? rawState.actividadTipo : undefined
  const actividadFlags = rawState && 'actividadFlags' in rawState ? rawState.actividadFlags : undefined
  const { user } = useAuth()

  const routePrefix = location.pathname.startsWith('/docente') ? '/docente' : '/codocente'
  const backPath = `${routePrefix}/cursos/${cid}/actividades/${aid}/submissions`

  const [downloading, setDownloading] = useState(false)
  const [downloadError, setDownloadError] = useState<string | null>(null)

  // Appeal state
  const [appeal, setAppeal] = useState<Appeal | null>(null)
  const [appealDecision, setAppealDecision] = useState<AppealDecision | null>(null)
  const [appealLoading, setAppealLoading] = useState(false)
  const [appealError, setAppealError] = useState<string | null>(null)
  const [showResolveForm, setShowResolveForm] = useState(false)
  const [resolveDecision, setResolveDecision] = useState<'upheld' | 'rejected'>('rejected')
  const [justification, setJustification] = useState('')
  const [resolveError, setResolveError] = useState<string | null>(null)
  const [resolveSubmitting, setResolveSubmitting] = useState(false)

  useEffect(() => {
    if (!item?.session_id) return
    setAppealLoading(true)
    getAppeal(item.session_id)
      .then(setAppeal)
      .catch((err: Error) => setAppealError(err.message))
      .finally(() => setAppealLoading(false))
  }, [item?.session_id])

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

      <div className="flex flex-wrap gap-3">
        <button
          type="button"
          onClick={handleDownload}
          disabled={downloading}
          className="bg-[#E85D24] hover:bg-[#d4521f] text-white px-4 py-2 rounded-lg text-sm transition-colors disabled:opacity-50"
        >
          {downloading ? 'Descargando...' : 'Descargar'}
        </button>

        {item.session_id && (
          <Link
            to={`${location.pathname}/trace`}
            className="inline-flex items-center gap-2 bg-[#1a1a1a] border border-[#2e2e2e] rounded-lg px-4 py-2 text-sm text-[#f0f0f0] hover:border-[#E85D24] transition-colors"
          >
            Ver trazas
          </Link>
        )}
      </div>

      <div className="border-t border-[#2e2e2e] pt-4 mt-4">
        <h2 className="text-base font-semibold text-[#f0f0f0] mb-3">Calificación</h2>
        <GradePanel
          submissionId={sid!}
          existingGrade={item.grade}
          readOnly={user?.role === 'co_docente'}
        />
      </div>

      {item.session_id && (
        <div className="border-t border-[#2e2e2e] pt-4 mt-4">
          <h2 className="text-base font-semibold text-[#f0f0f0] mb-3">Apelación</h2>
          {appealLoading && <p className="text-[#888888] text-sm">Cargando apelación...</p>}
          {appealError && <ErrorMessage message={appealError} />}
          {!appealLoading && !appealError && appeal === null && (
            <p className="text-[#888888] text-sm">Sin apelación.</p>
          )}
          {!appealLoading && !appealError && appeal !== null && (
            <div className="space-y-3">
              <div className="bg-[#1a1a1a] border border-[#2e2e2e] rounded-lg p-4 space-y-3">
                <div className="flex items-center gap-2">
                  <span className={`text-xs px-2 py-0.5 rounded ${APPEAL_STATUS_CLASS[appeal.status] ?? 'bg-[#2e2e2e] text-[#888888]'}`}>
                    {APPEAL_STATUS_LABEL[appeal.status] ?? appeal.status}
                  </span>
                  <span className="text-xs text-[#888888]">{formatDateTime(appeal.created_at)}</span>
                </div>
                <p className="text-sm text-[#f0f0f0]">{appeal.reason}</p>
                {appeal.status === 'escalated' && (
                  <p className="text-xs text-[#888888]">
                    La apelación ha sido derivada al administrador para revisión final.
                  </p>
                )}
                {appealDecision && (
                  <div className="border-t border-[#2e2e2e] pt-3">
                    <p className="text-xs text-[#888888] mb-1">Tu evaluación registrada:</p>
                    <p className="text-sm text-[#f0f0f0]">{appealDecision.justification}</p>
                  </div>
                )}
              </div>
              {appeal.status === 'pending' && user?.role === 'docente' && !showResolveForm && (
                <button
                  type="button"
                  onClick={() => setShowResolveForm(true)}
                  className="text-sm border border-[#2e2e2e] text-[#f0f0f0] px-4 py-2 rounded-lg hover:border-[#E85D24] transition-colors"
                >
                  Evaluar apelación
                </button>
              )}
              {showResolveForm && (
                <div className="bg-[#1a1a1a] border border-[#2e2e2e] rounded-lg p-4 space-y-3">
                  <p className="text-sm font-medium text-[#f0f0f0]">Tu evaluación</p>
                  <div className="flex gap-4">
                    <label className="flex items-center gap-2 cursor-pointer">
                      <input
                        type="radio"
                        name="resolve-decision"
                        value="upheld"
                        checked={resolveDecision === 'upheld'}
                        onChange={() => setResolveDecision('upheld')}
                        className="accent-[#E85D24]"
                      />
                      <span className="text-sm text-[#f0f0f0]">La apelación tiene mérito</span>
                    </label>
                    <label className="flex items-center gap-2 cursor-pointer">
                      <input
                        type="radio"
                        name="resolve-decision"
                        value="rejected"
                        checked={resolveDecision === 'rejected'}
                        onChange={() => setResolveDecision('rejected')}
                        className="accent-[#E85D24]"
                      />
                      <span className="text-sm text-[#f0f0f0]">Mantener calificación</span>
                    </label>
                  </div>
                  <textarea
                    value={justification}
                    onChange={(e) => setJustification(e.target.value)}
                    placeholder="Justifica tu evaluación..."
                    rows={3}
                    className="w-full bg-[#111111] border border-[#2e2e2e] rounded-lg p-3 text-[#f0f0f0] text-sm placeholder-[#888888] focus:outline-none focus:border-[#E85D24]"
                  />
                  {resolveError && <p className="text-red-400 text-sm">{resolveError}</p>}
                  <div className="flex gap-2">
                    <button
                      type="button"
                      disabled={resolveSubmitting}
                      onClick={async () => {
                        if (!justification.trim()) {
                          setResolveError('La justificación no puede estar vacía.')
                          return
                        }
                        setResolveError(null)
                        setResolveSubmitting(true)
                        try {
                          const result = await reviewAppeal(appeal.id, resolveDecision, justification.trim())
                          setAppeal(result.appeal)
                          setAppealDecision(result.decision ?? null)
                          setShowResolveForm(false)
                          setJustification('')
                        } catch (err: unknown) {
                          setResolveError((err as Error).message)
                        } finally {
                          setResolveSubmitting(false)
                        }
                      }}
                      className="px-4 py-2 bg-[#E85D24] text-white text-sm rounded-lg hover:bg-[#d4521f] disabled:opacity-50 transition-colors"
                    >
                      {resolveSubmitting ? 'Enviando...' : 'Confirmar'}
                    </button>
                    <button
                      type="button"
                      onClick={() => { setShowResolveForm(false); setJustification(''); setResolveError(null) }}
                      className="px-4 py-2 text-sm border border-[#2e2e2e] text-[#f0f0f0] rounded-lg hover:border-[#888888] transition-colors"
                    >
                      Cancelar
                    </button>
                  </div>
                </div>
              )}
            </div>
          )}
        </div>
      )}

      {actividadTipo === 'informe' && (
        <div className="border-t border-[#2e2e2e] pt-4 mt-4">
          <h2 className="text-base font-semibold text-[#f0f0f0] mb-3">Etapa de investigación</h2>
          <StageSelector submissionId={sid!} currentStage={item.stage} />
        </div>
      )}

      {(item.repo_url || item.session_id) && (
        <div className="border-t border-[#2e2e2e] pt-4 mt-4 space-y-2">
          <h2 className="text-base font-semibold text-[#f0f0f0] mb-3">Análisis y Reportes</h2>
          {item.repo_url && (
            <Link
              to={`${routePrefix}/cursos/${cid}/actividades/${aid}/submissions/${sid}/git-analysis`}
              className="flex items-center gap-2 text-sm text-[#f0f0f0] border border-[#2e2e2e] rounded-lg px-4 py-2 hover:border-[#E85D24] transition-colors w-fit"
            >
              Ver análisis Git
            </Link>
          )}
          {item.session_id && (
            <Link
              to={`${routePrefix}/cursos/${cid}/actividades/${aid}/submissions/${sid}/detection-report`}
              state={{ sessionId: item.session_id }}
              className="flex items-center gap-2 text-sm text-[#f0f0f0] border border-[#2e2e2e] rounded-lg px-4 py-2 hover:border-[#E85D24] transition-colors w-fit"
            >
              Ver informe de detección e IA
            </Link>
          )}
        </div>
      )}
    </div>
  )
}
