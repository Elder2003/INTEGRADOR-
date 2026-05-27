import { useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { getStudentActividades } from '../../api/student'
import {
  getSubmission,
  uploadSubmission,
  replaceSubmission,
  deleteSubmission,
  studentDownloadSubmission,
} from '../../api/submissions'
import type { StudentActividadEntry, SubmissionResponse } from '../../types/academic'
import { LoadingSpinner } from '../../components/common/LoadingSpinner'
import { ErrorMessage } from '../../components/common/ErrorMessage'
import { ConfirmModal } from '../../components/common/ConfirmModal'
import { FileUpload } from '../../components/student/FileUpload'
import { IntegrityDeclaration } from '../../components/student/IntegrityDeclaration'
import { SessionPanel } from '../../components/student/SessionPanel'
import { useTracingSession } from '../../hooks/useTracingSession'
import { GradeDisplay } from '../../components/student/GradeDisplay'
import { ExamForm } from '../../components/student/ExamForm'
import { DefensaSlotDisplay } from '../../components/student/DefensaSlotDisplay'
import { GitSubmissionForm } from '../../components/student/GitSubmissionForm'
import { getMyDefensaSlot } from '../../api/grading'
import type { DefensaSlot } from '../../types/academic'
import { getAppeal, createAppeal } from '../../api/appeals'
import type { Appeal } from '../../types/academic'

const APPEAL_STATUS_LABEL: Record<string, string> = {
  pending: 'Pendiente',
  accepted: 'Aceptada',
  rejected: 'Rechazada',
}

const APPEAL_STATUS_CLASS: Record<string, string> = {
  pending: 'bg-yellow-400/10 text-yellow-400',
  accepted: 'bg-green-400/10 text-green-400',
  rejected: 'bg-red-400/10 text-red-400',
}

const tipoLabel: Record<string, string> = {
  practica: 'Práctica',
  tarea: 'Tarea',
  informe: 'Informe',
  examen: 'Examen',
  defensa_oral: 'Defensa Oral',
}

function formatDeadline(iso: string): { label: string; isPast: boolean } {
  const d = new Date(iso)
  const isPast = d < new Date()
  const label = d.toLocaleString('es-BO', {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
  return { label, isPast }
}

function formatDateTime(iso: string): string {
  return new Date(iso).toLocaleString('es-BO', {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

export function ActividadDetailPage() {
  const { id: cursoId, aid } = useParams<{ id: string; aid: string }>()

  const [actividad, setActividad] = useState<StudentActividadEntry | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  // Submission state
  const [submission, setSubmission] = useState<SubmissionResponse | null>(null)
  const [submLoading, setSubmLoading] = useState(true)
  const [submError, setSubmError] = useState<string | null>(null)

  // Upload form state
  const [selectedFile, setSelectedFile] = useState<File | null>(null)
  const [integrityChecked, setIntegrityChecked] = useState(false)
  const [uploading, setUploading] = useState(false)
  const [uploadProgress, setUploadProgress] = useState(0)
  const [uploadError, setUploadError] = useState<string | null>(null)

  // Replace form state
  const [showReplaceForm, setShowReplaceForm] = useState(false)
  const [replaceFile, setReplaceFile] = useState<File | null>(null)
  const [replacing, setReplacing] = useState(false)
  const [replaceProgress, setReplaceProgress] = useState(0)
  const [replaceError, setReplaceError] = useState<string | null>(null)

  // Delete state
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false)
  const [deleting, setDeleting] = useState(false)

  // Session state
  const [showEndConfirm, setShowEndConfirm] = useState(false)
  const { status: sessionStatus, startTime: sessionStartTime, startSession, terminateSession } =
    useTracingSession(aid ?? '', actividad?.trace_enabled ?? false)

  // Defensa slot state (only fetched for defensa_oral)
  const [defensaSlot, setDefensaSlot] = useState<DefensaSlot | null>(null)

  // Appeal state
  const [appeal, setAppeal] = useState<Appeal | null>(null)
  const [appealLoading, setAppealLoading] = useState(false)
  const [showAppealForm, setShowAppealForm] = useState(false)
  const [appealMessage, setAppealMessage] = useState('')
  const [appealSubmitError, setAppealSubmitError] = useState<string | null>(null)
  const [appealSubmitting, setAppealSubmitting] = useState(false)

  useEffect(() => {
    if (!cursoId || !aid) return

    getStudentActividades(cursoId)
      .then((list) => {
        const found = list.find((a) => a.id === aid)
        if (!found) {
          setError('Actividad no encontrada.')
        } else {
          setActividad(found)
        }
      })
      .catch((err: Error) => {
        if (
          err.message.toLowerCase().includes('not enrolled') ||
          err.message.toLowerCase().includes('forbidden')
        ) {
          setError('No tienes acceso a este curso.')
        } else {
          setError(err.message)
        }
      })
      .finally(() => setLoading(false))

    getSubmission(aid)
      .then((sub) => setSubmission(sub))
      .catch((err: Error) => setSubmError(err.message))
      .finally(() => setSubmLoading(false))
  }, [cursoId, aid])

  useEffect(() => {
    if (!aid || !actividad || actividad.tipo !== 'defensa_oral') return
    getMyDefensaSlot(aid).then(setDefensaSlot).catch(() => {})
  }, [aid, actividad])

  useEffect(() => {
    if (!submission || !submission.grade) return
    setAppealLoading(true)
    getAppeal(submission.id)
      .then(setAppeal)
      .catch(() => {})
      .finally(() => setAppealLoading(false))
  }, [submission])

  async function handleUpload() {
    if (!aid || !selectedFile) return
    setUploading(true)
    setUploadError(null)
    try {
      const result = await uploadSubmission(aid, selectedFile, setUploadProgress)
      setSubmission(result)
      setSelectedFile(null)
      setIntegrityChecked(false)
      setUploadProgress(0)
    } catch (err: unknown) {
      setUploadError((err as Error).message)
    } finally {
      setUploading(false)
    }
  }

  async function handleReplace() {
    if (!aid || !replaceFile) return
    setReplacing(true)
    setReplaceError(null)
    try {
      const result = await replaceSubmission(aid, replaceFile, setReplaceProgress)
      setSubmission(result)
      setReplaceFile(null)
      setShowReplaceForm(false)
      setReplaceProgress(0)
    } catch (err: unknown) {
      setReplaceError((err as Error).message)
    } finally {
      setReplacing(false)
    }
  }

  async function handleDelete() {
    if (!aid) return
    setDeleting(true)
    try {
      await deleteSubmission(aid)
      setSubmission(null)
    } catch {
      // deletion error handled by modal
    } finally {
      setDeleting(false)
      setShowDeleteConfirm(false)
    }
  }

  if (loading) return <LoadingSpinner />

  if (error) {
    return (
      <div className="p-8 space-y-4">
        <ErrorMessage message={error} />
        <Link
          to={`/student/cursos/${cursoId}/actividades`}
          className="text-sm text-[#E85D24] hover:underline"
        >
          ← Volver al curso
        </Link>
      </div>
    )
  }

  if (!actividad) return null

  const { label, isPast } = formatDeadline(actividad.deadline)

  return (
    <div className="p-8 max-w-3xl space-y-6">
      <Link
        to={`/student/cursos/${cursoId}/actividades`}
        className="text-sm text-[#888888] hover:text-[#E85D24] transition-colors"
      >
        ← Volver al curso
      </Link>

      <h1 className="text-2xl font-semibold text-[#f0f0f0] mt-4">{actividad.title}</h1>

      <div className="flex flex-wrap items-center gap-2">
        <span className="bg-[#2e2e2e] text-[#888888] text-xs px-2 py-0.5 rounded">
          {tipoLabel[actividad.tipo] ?? actividad.tipo}
        </span>
        <span className="text-sm text-[#888888]">{label}</span>
        {isPast && <span className="text-xs text-[#888888] font-medium">Vencido</span>}
        {actividad.trace_enabled && (
          <span className="bg-[#E85D24]/20 text-[#E85D24] text-xs px-2 py-0.5 rounded">
            Requiere monitoreo
          </span>
        )}
      </div>

      <div className="bg-[#1a1a1a] border border-[#2e2e2e] rounded-lg p-4">
        <h2 className="text-sm font-medium text-[#888888] mb-2">Instrucciones</h2>
        <pre className="whitespace-pre-wrap text-[#f0f0f0] text-sm">{actividad.instructions}</pre>
      </div>

      {actividad.trace_enabled && (
        <div className="space-y-1">
          <h2 className="text-lg font-semibold text-[#f0f0f0]">Sesión de monitoreo</h2>
          <SessionPanel
            status={sessionStatus}
            startTime={sessionStartTime}
            onStart={startSession}
            onEnd={() => setShowEndConfirm(true)}
          />
        </div>
      )}

      {/* Submission section */}
      <div>
        <h2 className="text-lg font-semibold text-[#f0f0f0] mb-4">Mi Entrega</h2>

        {submLoading && <LoadingSpinner />}

        {!submLoading && submError && <ErrorMessage message={submError} />}

        {/* Git-enabled activities use GitSubmissionForm instead of file upload */}
        {!submLoading && !submError && actividad.git_enabled && (
          <GitSubmissionForm
            actividadId={aid!}
            existingSubmission={submission}
            isPastDeadline={isPast}
            onSuccess={setSubmission}
          />
        )}

        {/* Exam activities use ExamForm instead of file upload */}
        {!submLoading && !submError && !actividad.git_enabled && actividad.tipo === 'examen' && (
          <ExamForm
            actividadId={aid!}
            existingAnswer={submission?.exam_answer ?? null}
            submittedAt={submission?.submitted_at ?? null}
            deadline={actividad.deadline}
            sessionGate={actividad.trace_enabled && sessionStatus !== 'active'}
          />
        )}

        {!submLoading && !submError && !actividad.git_enabled && actividad.tipo !== 'examen' && submission === null && (
          actividad.trace_enabled && sessionStatus !== 'active' ? (
            <p className="text-[#888888] text-sm">
              Inicia la sesión de monitoreo para poder subir tu entrega.
            </p>
          ) : (
            <div className="space-y-4">
              <IntegrityDeclaration checked={integrityChecked} onChange={setIntegrityChecked} />
              <FileUpload
                acceptedTypes={[]}
                acceptedLabels={['Cualquier formato']}
                onFileSelect={(f) => {
                  if (f.size > 52_428_800) {
                    setUploadError('El archivo supera el límite de 50 MB.')
                    return
                  }
                  setSelectedFile(f)
                  setUploadError(null)
                }}
                selectedFile={selectedFile}
                onClear={() => setSelectedFile(null)}
              />
              {uploadError && <p className="text-red-400 text-sm">{uploadError}</p>}
              {uploading && (
                <div className="w-full bg-[#2e2e2e] rounded-full h-1.5">
                  <div
                    className="bg-[#E85D24] h-1.5 rounded-full transition-all"
                    style={{ width: `${uploadProgress}%` }}
                  />
                </div>
              )}
              <button
                type="button"
                onClick={handleUpload}
                disabled={!integrityChecked || !selectedFile || uploading}
                className="px-4 py-2 bg-[#E85D24] text-white text-sm rounded-lg hover:bg-[#d4521f] disabled:opacity-50 transition-colors"
              >
                {uploading ? 'Subiendo...' : 'Subir entrega'}
              </button>
            </div>
          )
        )}

        {!submLoading && !submError && !actividad.git_enabled && actividad.tipo !== 'examen' && submission !== null && (
          <div className="bg-[#1a1a1a] border border-[#2e2e2e] rounded-lg p-4 space-y-3">
            <div className="flex flex-wrap items-start justify-between gap-3">
              <div className="space-y-1">
                <p className="text-[#f0f0f0] font-medium text-sm">{submission.original_filename}</p>
                <p className="text-[#888888] text-xs">
                  {(submission.file_size_bytes / 1024).toFixed(0)} KB
                </p>
                <p className="text-[#888888] text-xs">
                  Enviado el {formatDateTime(submission.submitted_at)}
                </p>
                {submission.is_late && (
                  <span className="inline-block text-xs bg-yellow-400/10 text-yellow-400 px-2 py-0.5 rounded">
                    Entrega tardía
                  </span>
                )}
                {submission.is_locked ? (
                  <span className="inline-block text-xs bg-[#2e2e2e] text-[#888888] px-2 py-0.5 rounded">
                    Entrega bloqueada
                  </span>
                ) : (
                  submission.edit_window_expires_at && (
                    <p className="text-[#888888] text-xs">
                      Ventana de edición hasta: {formatDateTime(submission.edit_window_expires_at)}
                    </p>
                  )
                )}
              </div>

              <div className="flex flex-col gap-2">
                <button
                  type="button"
                  onClick={() => studentDownloadSubmission(aid!, submission.original_filename)}
                  className="text-sm text-[#E85D24] hover:underline"
                >
                  Descargar mi entrega
                </button>
                {!submission.is_locked && (
                  <>
                    <button
                      type="button"
                      onClick={() => setShowReplaceForm(true)}
                      className="text-sm text-[#f0f0f0] border border-[#2e2e2e] px-3 py-1 rounded-lg hover:border-[#E85D24] transition-colors"
                    >
                      Reemplazar
                    </button>
                    <button
                      type="button"
                      onClick={() => setShowDeleteConfirm(true)}
                      className="text-sm text-red-400 border border-[#2e2e2e] px-3 py-1 rounded-lg hover:border-red-400 transition-colors"
                    >
                      Eliminar
                    </button>
                  </>
                )}
              </div>
            </div>

            {showReplaceForm && !submission.is_locked && (
              <div className="border-t border-[#2e2e2e] pt-4 space-y-3">
                <p className="text-sm text-[#888888]">Selecciona el archivo de reemplazo:</p>
                <FileUpload
                  acceptedTypes={[]}
                  acceptedLabels={['Cualquier formato']}
                  onFileSelect={(f) => {
                    if (f.size > 52_428_800) {
                      setReplaceError('El archivo supera el límite de 50 MB.')
                      return
                    }
                    setReplaceFile(f)
                    setReplaceError(null)
                  }}
                  selectedFile={replaceFile}
                  onClear={() => setReplaceFile(null)}
                />
                {replaceError && <p className="text-red-400 text-sm">{replaceError}</p>}
                {replacing && (
                  <div className="w-full bg-[#2e2e2e] rounded-full h-1.5">
                    <div
                      className="bg-[#E85D24] h-1.5 rounded-full transition-all"
                      style={{ width: `${replaceProgress}%` }}
                    />
                  </div>
                )}
                <div className="flex gap-2">
                  <button
                    type="button"
                    onClick={handleReplace}
                    disabled={!replaceFile || replacing}
                    className="px-4 py-2 bg-[#E85D24] text-white text-sm rounded-lg hover:bg-[#d4521f] disabled:opacity-50 transition-colors"
                  >
                    {replacing ? 'Reemplazando...' : 'Confirmar reemplazo'}
                  </button>
                  <button
                    type="button"
                    onClick={() => {
                      setShowReplaceForm(false)
                      setReplaceFile(null)
                      setReplaceError(null)
                    }}
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

      {actividad.tipo === 'defensa_oral' && (
        <div className="border-t border-[#2e2e2e] pt-4 mt-4">
          <h2 className="text-base font-semibold text-[#f0f0f0] mb-3">Mi Defensa</h2>
          <DefensaSlotDisplay slot={defensaSlot} />
        </div>
      )}

      <div className="border-t border-[#2e2e2e] pt-4 mt-4">
        <h2 className="text-base font-semibold text-[#f0f0f0] mb-3">Mi Calificación</h2>
        <GradeDisplay grade={submission?.grade ?? null} hasSubmission={submission !== null} />
      </div>

      {submission !== null && submission.grade !== null && !appealLoading && (
        <div className="border-t border-[#2e2e2e] pt-4 mt-4">
          <h2 className="text-base font-semibold text-[#f0f0f0] mb-3">Apelación</h2>
          {appeal === null && !showAppealForm && (
            <button
              type="button"
              onClick={() => setShowAppealForm(true)}
              className="text-sm border border-[#2e2e2e] text-[#f0f0f0] px-4 py-2 rounded-lg hover:border-[#E85D24] transition-colors"
            >
              Presentar apelación
            </button>
          )}
          {showAppealForm && (
            <div className="space-y-3">
              <textarea
                value={appealMessage}
                onChange={(e) => setAppealMessage(e.target.value)}
                placeholder="Explica el motivo de tu apelación..."
                rows={4}
                className="w-full bg-[#1a1a1a] border border-[#2e2e2e] rounded-lg p-3 text-[#f0f0f0] text-sm placeholder-[#888888] focus:outline-none focus:border-[#E85D24]"
              />
              {appealSubmitError && <p className="text-red-400 text-sm">{appealSubmitError}</p>}
              <div className="flex gap-2">
                <button
                  type="button"
                  disabled={appealSubmitting}
                  onClick={async () => {
                    if (!appealMessage.trim()) {
                      setAppealSubmitError('La explicación no puede estar vacía.')
                      return
                    }
                    setAppealSubmitError(null)
                    setAppealSubmitting(true)
                    try {
                      const result = await createAppeal(submission.id, appealMessage.trim())
                      setAppeal(result)
                      setShowAppealForm(false)
                      setAppealMessage('')
                    } catch (err: unknown) {
                      setAppealSubmitError((err as Error).message)
                    } finally {
                      setAppealSubmitting(false)
                    }
                  }}
                  className="px-4 py-2 bg-[#E85D24] text-white text-sm rounded-lg hover:bg-[#d4521f] disabled:opacity-50 transition-colors"
                >
                  {appealSubmitting ? 'Enviando...' : 'Enviar apelación'}
                </button>
                <button
                  type="button"
                  onClick={() => { setShowAppealForm(false); setAppealMessage(''); setAppealSubmitError(null) }}
                  className="px-4 py-2 text-sm border border-[#2e2e2e] text-[#f0f0f0] rounded-lg hover:border-[#888888] transition-colors"
                >
                  Cancelar
                </button>
              </div>
            </div>
          )}
          {appeal !== null && (
              <div className="bg-[#1a1a1a] border border-[#2e2e2e] rounded-lg p-4 space-y-3">
                <div className="flex items-center gap-2">
                  <span className={`text-xs px-2 py-0.5 rounded ${APPEAL_STATUS_CLASS[appeal.status] ?? 'bg-[#2e2e2e] text-[#888888]'}`}>
                    {APPEAL_STATUS_LABEL[appeal.status] ?? appeal.status}
                  </span>
                  <span className="text-xs text-[#888888]">{formatDateTime(appeal.created_at)}</span>
                </div>
                <p className="text-sm text-[#f0f0f0]">{appeal.student_message}</p>
                {appeal.docente_response && (
                  <div className="border-t border-[#2e2e2e] pt-3">
                    <p className="text-xs text-[#888888] mb-1">Respuesta del docente:</p>
                    <p className="text-sm text-[#f0f0f0]">{appeal.docente_response}</p>
                  </div>
                )}
              </div>
          )}
        </div>
      )}

      <ConfirmModal
        isOpen={showDeleteConfirm}
        title="Eliminar entrega"
        message="¿Seguro que deseas eliminar tu entrega? Esta acción no se puede deshacer."
        confirmLabel="Eliminar"
        isLoading={deleting}
        onConfirm={handleDelete}
        onCancel={() => setShowDeleteConfirm(false)}
      />

      <ConfirmModal
        isOpen={showEndConfirm}
        title="Terminar sesión"
        message="¿Seguro que deseas terminar la sesión de monitoreo?"
        confirmLabel="Terminar"
        isLoading={false}
        onConfirm={() => { terminateSession(); setShowEndConfirm(false) }}
        onCancel={() => setShowEndConfirm(false)}
      />
    </div>
  )
}
