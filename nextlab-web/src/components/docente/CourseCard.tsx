import type { Curso } from '../../types/academic'

interface CourseCardProps {
  curso: Curso
  onClick: () => void
}

export function CourseCard({ curso, onClick }: CourseCardProps) {
  return (
    <button
      type="button"
      onClick={onClick}
      className="w-full text-left bg-[#1a1a1a] border border-[#2e2e2e] rounded-xl p-5 hover:border-[#E85D24] transition-colors"
    >
      <h3 className="text-[#f0f0f0] font-semibold text-base">{curso.materia_nombre}</h3>
      <p className="text-[#888888] text-sm mt-1">
        {curso.semester} · Sección {curso.section}
      </p>
      <p className="text-[#888888] text-xs mt-2">
        {curso.student_count} {curso.student_count === 1 ? 'estudiante' : 'estudiantes'}
      </p>
    </button>
  )
}
