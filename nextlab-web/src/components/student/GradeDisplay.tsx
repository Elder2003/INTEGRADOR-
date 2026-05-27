import type { GradeDecision } from '../../types/academic'

interface GradeDisplayProps {
  grade: GradeDecision | null
  hasSubmission: boolean
}

export function GradeDisplay({ grade, hasSubmission }: GradeDisplayProps) {
  if (!hasSubmission) return null

  if (grade === null) {
    return (
      <div className="bg-[#1a1a1a] border border-[#2e2e2e] rounded-lg p-4">
        <p className="text-[#888888] text-sm">Pendiente de calificación</p>
      </div>
    )
  }

  return (
    <div className="bg-[#1a1a1a] border border-[#2e2e2e] rounded-lg p-4 space-y-2">
      <div className="flex items-center gap-2">
        <span className="text-2xl font-bold text-[#E85D24]">{grade.score} / 100</span>
      </div>
      <p className="text-[#f0f0f0] text-sm whitespace-pre-wrap">{grade.feedback}</p>
      <p className="text-[#888888] text-xs">
        Calificado el {new Date(grade.graded_at).toLocaleDateString('es-BO')}
      </p>
    </div>
  )
}
