import { useState } from 'react'
import { submitExamAnswer } from '../../api/grading'

interface ExamFormProps {
  actividadId: string
  existingAnswer: string | null
  submittedAt: string | null
  deadline: string
  sessionGate: boolean
}

export function ExamForm({
  actividadId,
  existingAnswer,
  submittedAt,
  deadline,
  sessionGate,
}: ExamFormProps) {
  const [answer, setAnswer] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [localAnswer, setLocalAnswer] = useState<string | null>(existingAnswer)
  const [localSubmittedAt, setLocalSubmittedAt] = useState<string | null>(submittedAt)
  const [errorMsg, setErrorMsg] = useState<string | null>(null)

  const mode: 'submitted' | 'closed' | 'gated' | 'draft' = (() => {
    if (localAnswer !== null) return 'submitted'
    if (new Date() > new Date(deadline)) return 'closed'
    if (sessionGate) return 'gated'
    return 'draft'
  })()

  async function handleSubmit() {
    if (!answer.trim()) return
    setSubmitting(true)
    setErrorMsg(null)
    try {
      const res = await submitExamAnswer(actividadId, answer)
      setLocalAnswer(res.exam_answer)
      setLocalSubmittedAt(res.submitted_at)
    } catch (err: unknown) {
      setErrorMsg((err as Error).message)
    } finally {
      setSubmitting(false)
    }
  }

  if (mode === 'gated') {
    return (
      <p className="text-[#888888] text-sm">
        Inicia la sesión de monitoreo para poder responder el examen.
      </p>
    )
  }

  if (mode === 'closed') {
    return (
      <div className="bg-[#1a1a1a] border border-[#2e2e2e] rounded-lg p-4">
        <p className="text-sm font-medium text-[#888888] mb-2">Examen cerrado</p>
        {localAnswer ? (
          <p className="text-[#f0f0f0] text-sm whitespace-pre-wrap">{localAnswer}</p>
        ) : (
          <p className="text-[#888888] text-xs">Sin respuesta registrada.</p>
        )}
      </div>
    )
  }

  if (mode === 'submitted') {
    return (
      <div className="bg-[#1a1a1a] border border-[#2e2e2e] rounded-lg p-4 space-y-2">
        <p className="text-green-400 text-xs">
          Enviado el {new Date(localSubmittedAt!).toLocaleString('es-BO')}
        </p>
        <p className="text-[#f0f0f0] text-sm whitespace-pre-wrap">{localAnswer}</p>
      </div>
    )
  }

  return (
    <div className="space-y-3">
      <textarea
        aria-label="Respuesta del examen"
        value={answer}
        onChange={e => setAnswer(e.target.value)}
        rows={12}
        disabled={submitting}
        className="w-full bg-[#111111] border border-[#2e2e2e] rounded px-3 py-2 text-[#f0f0f0] text-sm focus:border-[#E85D24] focus:outline-none resize-none disabled:opacity-50"
        placeholder="Escribe tus respuestas aquí..."
      />
      {errorMsg && <p className="text-red-400 text-sm">{errorMsg}</p>}
      <button
        type="button"
        onClick={handleSubmit}
        disabled={submitting || !answer.trim()}
        className="px-4 py-2 bg-[#E85D24] text-white text-sm rounded-lg hover:bg-[#d4521f] disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
      >
        {submitting ? 'Enviando...' : 'Enviar respuesta'}
      </button>
    </div>
  )
}
