import { useState } from 'react'
import { submitGitRepo, updateGitRepo } from '../../api/gitAnalysis'
import type { SubmissionResponse } from '../../types/academic'

interface Props {
  actividadId: string
  existingSubmission: SubmissionResponse | null
  isPastDeadline: boolean
  onSuccess: (submission: SubmissionResponse) => void
}

type Mode = 'form' | 'submitting' | 'confirmed' | 'editing' | 'updating' | 'closed'

function formatDateTime(iso: string): string {
  return new Date(iso).toLocaleString('es-BO', {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

export function GitSubmissionForm({ actividadId, existingSubmission, isPastDeadline, onSuccess }: Props) {
  const systemAccount = import.meta.env.VITE_GIT_SYSTEM_ACCOUNT ?? 'nextlab-bot'

  const [mode, setMode] = useState<Mode>(() => {
    if (existingSubmission) return 'confirmed'
    if (isPastDeadline) return 'closed'
    return 'form'
  })
  const [repoUrl, setRepoUrl] = useState(existingSubmission?.repo_url ?? '')
  const [validationError, setValidationError] = useState<string | null>(null)
  const [submitError, setSubmitError] = useState<string | null>(null)

  function validate(): boolean {
    if (!repoUrl.trim()) {
      setValidationError('El URL del repositorio es obligatorio.')
      return false
    }
    if (!repoUrl.includes('://')) {
      setValidationError('Ingresa un URL válido (debe contener "://").')
      return false
    }
    setValidationError(null)
    return true
  }

  async function handleSubmit() {
    if (!validate()) return
    setSubmitError(null)
    setMode('submitting')
    try {
      const result = await submitGitRepo(actividadId, repoUrl.trim())
      onSuccess(result)
      setRepoUrl(result.repo_url ?? repoUrl.trim())
      setMode('confirmed')
    } catch (err: unknown) {
      setSubmitError((err as Error).message)
      setMode('form')
    }
  }

  async function handleUpdate() {
    if (!validate()) return
    setSubmitError(null)
    setMode('updating')
    try {
      const result = await updateGitRepo(actividadId, repoUrl.trim())
      onSuccess(result)
      setRepoUrl(result.repo_url ?? repoUrl.trim())
      setMode('confirmed')
    } catch (err: unknown) {
      setSubmitError((err as Error).message)
      setMode('editing')
    }
  }

  const effectiveMode: Mode = (() => {
    if (existingSubmission && mode !== 'editing' && mode !== 'updating') return 'confirmed'
    if (isPastDeadline && !existingSubmission) return 'closed'
    return mode
  })()

  if (effectiveMode === 'closed') {
    return (
      <div className="bg-[#1a1a1a] border border-[#2e2e2e] rounded-lg p-4">
        <p className="text-[#888888] text-sm">El período de entrega ha cerrado. No se aceptan nuevas entregas.</p>
      </div>
    )
  }

  if (effectiveMode === 'confirmed') {
    const sub = existingSubmission!
    return (
      <div className="bg-[#1a1a1a] border border-[#2e2e2e] rounded-lg p-4 space-y-3">
        <div className="space-y-1">
          <p className="text-[#888888] text-xs font-medium uppercase tracking-wide">Repositorio enviado</p>
          <p className="text-[#f0f0f0] text-sm font-mono break-all">{sub.repo_url}</p>
          <p className="text-[#888888] text-xs">Enviado el {formatDateTime(sub.submitted_at)}</p>
          {sub.is_late && (
            <span className="inline-block text-xs bg-yellow-400/10 text-yellow-400 px-2 py-0.5 rounded">
              Entrega tardía
            </span>
          )}
          {sub.is_locked ? (
            <span className="inline-block text-xs bg-[#2e2e2e] text-[#888888] px-2 py-0.5 rounded">
              Entrega bloqueada
            </span>
          ) : (
            sub.edit_window_expires_at && (
              <p className="text-[#888888] text-xs">
                Ventana de edición hasta: {formatDateTime(sub.edit_window_expires_at)}
              </p>
            )
          )}
        </div>
        {!sub.is_locked && (
          <button
            type="button"
            onClick={() => {
              setRepoUrl(sub.repo_url ?? '')
              setMode('editing')
              setSubmitError(null)
              setValidationError(null)
            }}
            className="text-sm text-[#f0f0f0] border border-[#2e2e2e] px-3 py-1 rounded-lg hover:border-[#E85D24] transition-colors"
          >
            Editar
          </button>
        )}
      </div>
    )
  }

  const isEditing = effectiveMode === 'editing' || effectiveMode === 'updating'
  const isLoading = effectiveMode === 'submitting' || effectiveMode === 'updating'

  return (
    <div className="space-y-4">
      <div className="bg-[#1a1a1a] border border-[#2e2e2e] rounded-lg p-4 text-sm text-[#888888]">
        <p>
          Para repositorios privados, añade a{' '}
          <span className="font-mono text-[#f0f0f0]">{systemAccount}</span>{' '}
          como colaborador antes de enviar.
        </p>
      </div>

      <div className="space-y-2">
        <label htmlFor="git-repo-url" className="block text-sm font-medium text-[#888888]">
          URL del repositorio
        </label>
        <input
          id="git-repo-url"
          type="url"
          value={repoUrl}
          onChange={e => {
            setRepoUrl(e.target.value)
            setValidationError(null)
          }}
          placeholder="https://github.com/usuario/repositorio"
          disabled={isLoading}
          className="w-full bg-[#111111] border border-[#2e2e2e] rounded-lg px-3 py-2 text-[#f0f0f0] text-sm focus:border-[#E85D24] focus:outline-none disabled:opacity-50 font-mono"
        />
        {validationError && <p className="text-red-400 text-xs">{validationError}</p>}
        {submitError && <p className="text-red-400 text-sm">{submitError}</p>}
      </div>

      <div className="flex gap-2">
        <button
          type="button"
          onClick={isEditing ? handleUpdate : handleSubmit}
          disabled={isLoading}
          className="px-4 py-2 bg-[#E85D24] text-white text-sm rounded-lg hover:bg-[#d4521f] disabled:opacity-50 transition-colors"
        >
          {isLoading
            ? isEditing ? 'Actualizando...' : 'Enviando...'
            : isEditing ? 'Actualizar repositorio' : 'Enviar repositorio'
          }
        </button>
        {isEditing && (
          <button
            type="button"
            onClick={() => {
              setMode('confirmed')
              setRepoUrl(existingSubmission?.repo_url ?? '')
              setValidationError(null)
              setSubmitError(null)
            }}
            disabled={isLoading}
            className="px-4 py-2 text-sm border border-[#2e2e2e] text-[#f0f0f0] rounded-lg hover:border-[#888888] transition-colors disabled:opacity-50"
          >
            Cancelar
          </button>
        )}
      </div>
    </div>
  )
}
