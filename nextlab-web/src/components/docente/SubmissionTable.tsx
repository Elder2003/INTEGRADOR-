import { useNavigate } from 'react-router-dom'
import type { SubmissionListItem } from '../../types/academic'

interface Props {
  submissions: SubmissionListItem[]
  cursoId: string
  actividadId: string
  userRole: 'docente' | 'co_docente'
}

export function SubmissionTable({ submissions, cursoId, actividadId, userRole }: Props) {
  const navigate = useNavigate()
  const base = userRole === 'docente' ? 'docente' : 'codocente'

  if (submissions.length === 0) {
    return (
      <p className="text-[#888888] py-4">Ningún estudiante ha enviado archivos aún.</p>
    )
  }

  return (
    <div className="overflow-x-auto">
      <table className="w-full text-sm">
        <thead>
          <tr className="text-left border-b border-[#2e2e2e]">
            <th className="pb-2 pr-4 text-[#888888] font-medium">Estudiante</th>
            <th className="pb-2 pr-4 text-[#888888] font-medium">Email</th>
            <th className="pb-2 pr-4 text-[#888888] font-medium">Archivo</th>
            <th className="pb-2 pr-4 text-[#888888] font-medium">Tamaño</th>
            <th className="pb-2 pr-4 text-[#888888] font-medium">Enviado</th>
            <th className="pb-2 text-[#888888] font-medium">Estado</th>
          </tr>
        </thead>
        <tbody>
          {submissions.map((item) => (
            <tr
              key={item.id}
              onClick={() =>
                navigate(
                  `/${base}/cursos/${cursoId}/actividades/${actividadId}/submissions/${item.id}`,
                  { state: item },
                )
              }
              className="border-b border-[#2e2e2e] cursor-pointer hover:bg-[#1a1a1a] transition-colors"
            >
              <td className="py-3 pr-4 text-[#f0f0f0]">{item.student.name}</td>
              <td className="py-3 pr-4 text-[#888888]">{item.student.email}</td>
              <td className="py-3 pr-4 text-[#f0f0f0] max-w-[180px] truncate">
                {item.original_filename}
              </td>
              <td className="py-3 pr-4 text-[#888888] whitespace-nowrap">
                {(item.file_size_bytes / 1024).toFixed(0)} KB
              </td>
              <td className="py-3 pr-4 text-[#888888] whitespace-nowrap">
                {new Date(item.submitted_at).toLocaleString('es-BO')}
              </td>
              <td className="py-3">
                <div className="flex flex-wrap gap-1">
                  {item.is_late && (
                    <span className="bg-yellow-400/10 text-yellow-400 text-xs px-2 py-0.5 rounded">
                      Tarde
                    </span>
                  )}
                  {item.is_locked && (
                    <span className="bg-[#2e2e2e] text-[#888888] text-xs px-2 py-0.5 rounded">
                      Bloqueada
                    </span>
                  )}
                </div>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
