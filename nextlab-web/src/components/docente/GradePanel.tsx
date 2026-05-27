import { useState } from 'react'
import { createGrade, updateGrade } from '../../api/grading'
import type { GradeDecision } from '../../types/academic'

type GradePanelMode = 'view' | 'editing' | 'submitting' | 'success' | 'error'

interface GradePanelProps {
  submissionId: string
  existingGrade: GradeDecision | null
  readOnly?: boolean
}

export function GradePanel({ submissionId, existingGrade, readOnly }: GradePanelProps) {
  const [mode, setMode] = useState<GradePanelMode>(existingGrade ? 'view' : 'editing')
  const [score, setScore] = useState(existingGrade ? String(existingGrade.score) : '')
  const [feedback, setFeedback] = useState(existingGrade?.feedback ?? '')
  const [errorMsg, setErrorMsg] = useState<string | null>(null)
  const [grade, setGrade] = useState<GradeDecision | null>(existingGrade)

  async function handleSubmit() {
    const n = parseInt(score)
    if (isNaN(n) || n < 0 || n > 100) {
      setErrorMsg('El puntaje debe ser un número entre 0 y 100')
      return
    }
    setMode('submitting')
    setErrorMsg(null)
    try {
      const saved = await (grade ? updateGrade : createGrade)(submissionId, n, feedback)
      setGrade(saved)
      setMode('success')
      setTimeout(() => setMode('view'), 3000)
    } catch (err: unknown) {
      setMode('error')
      setErrorMsg((err as Error).message)
    }
  }

  if (readOnly) {
    if (!grade) {
      return <p className="text-[#888888] text-sm">Sin calificación.</p>
    }
    return (
      <div className="bg-[#1a1a1a] border border-[#2e2e2e] rounded-lg p-4 space-y-2">
        <p className="text-xs text-[#888888] uppercase tracking-wide">Calificación</p>
        <span className="text-2xl font-bold text-[#E85D24]">{grade.score} / 100</span>
        <p className="text-[#f0f0f0] text-sm whitespace-pre-wrap">{grade.feedback}</p>
      </div>
    )
  }

  if (mode === 'view' && grade) {
    return (
      <div className="bg-[#1a1a1a] border border-[#2e2e2e] rounded-lg p-4 space-y-2">
        <p className="text-xs text-[#888888] uppercase tracking-wide">Calificación</p>
        <span className="text-2xl font-bold text-[#E85D24]">{grade.score} / 100</span>
        <p className="text-[#f0f0f0] text-sm whitespace-pre-wrap">{grade.feedback}</p>
        <button
          type="button"
          onClick={() => { setScore(String(grade.score)); setFeedback(grade.feedback); setMode('editing') }}
          className="text-xs text-[#888888] hover:text-[#E85D24] transition-colors mt-2"
        >
          Editar calificación
        </button>
      </div>
    )
  }

  if (mode === 'success') {
    return (
      <div className="bg-[#1a1a1a] border border-[#2e2e2e] rounded-lg p-4 space-y-2">
        <p className="text-xs text-[#888888] uppercase tracking-wide">Calificación</p>
        <span className="text-2xl font-bold text-[#E85D24]">{grade?.score} / 100</span>
        <p className="text-[#f0f0f0] text-sm whitespace-pre-wrap">{grade?.feedback}</p>
        <p className="text-green-400 text-xs">Calificación guardada ✓</p>
      </div>
    )
  }

  return (
    <div className="bg-[#1a1a1a] border border-[#2e2e2e] rounded-lg p-4 space-y-3">
      <p className="text-xs text-[#888888] uppercase tracking-wide">Calificar entrega</p>
      <div>
        <label htmlFor={`grade-score-${submissionId}`} className="text-xs text-[#888888] mb-1 block">Puntaje (0–100)</label>
        <input
          id={`grade-score-${submissionId}`}
          type="number"
          min="0"
          max="100"
          value={score}
          onChange={e => setScore(e.target.value)}
          disabled={mode === 'submitting'}
          className="w-24 bg-[#111111] border border-[#2e2e2e] rounded px-3 py-1.5 text-[#f0f0f0] text-sm focus:border-[#E85D24] focus:outline-none disabled:opacity-50"
          placeholder="0–100"
        />
      </div>
      <div>
        <label htmlFor={`grade-feedback-${submissionId}`} className="text-xs text-[#888888] mb-1 block">Retroalimentación</label>
        <textarea
          id={`grade-feedback-${submissionId}`}
          value={feedback}
          onChange={e => setFeedback(e.target.value)}
          rows={4}
          disabled={mode === 'submitting'}
          className="w-full bg-[#111111] border border-[#2e2e2e] rounded px-3 py-2 text-[#f0f0f0] text-sm focus:border-[#E85D24] focus:outline-none resize-none disabled:opacity-50"
          placeholder="Comentarios para el estudiante..."
        />
      </div>
      {errorMsg && <p className="text-red-400 text-sm">{errorMsg}</p>}
      <button
        type="button"
        onClick={handleSubmit}
        disabled={mode === 'submitting'}
        className="px-4 py-2 bg-[#E85D24] text-white text-sm rounded-lg hover:bg-[#d4521f] disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
      >
        {mode === 'submitting' ? 'Calificando...' : 'Calificar'}
      </button>
    </div>
  )
}
