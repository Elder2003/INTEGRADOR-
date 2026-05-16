import { BookOpen, ClipboardList, FileText, Code, FolderGit2 } from 'lucide-react'
import type { ActividadTipo } from '../../types'

interface Props {
  tipo: ActividadTipo
}

const config: Record<
  ActividadTipo,
  { label: string; icon: React.ReactNode; color: string }
> = {
  examen:   { label: 'Examen',   icon: <ClipboardList size={11} />, color: '#e05252' },
  tarea:    { label: 'Tarea',    icon: <BookOpen size={11} />,      color: '#4a9eda' },
  informe:  { label: 'Informe',  icon: <FileText size={11} />,      color: '#f0a500' },
  practica: { label: 'Práctica', icon: <Code size={11} />,          color: '#3fa66e' },
  proyecto: { label: 'Proyecto', icon: <FolderGit2 size={11} />,    color: '#E85D24' },
}

export default function ActivityTypeBadge({ tipo }: Props) {
  const { label, icon, color } = config[tipo]
  return (
    <span
      style={{
        display: 'inline-flex',
        alignItems: 'center',
        gap: 4,
        padding: '2px 8px',
        borderRadius: 4,
        fontSize: 'var(--font-size-xs)',
        fontWeight: 500,
        color,
        backgroundColor: `${color}1a`,
        border: `1px solid ${color}33`,
      }}
    >
      {icon}
      {label}
    </span>
  )
}
