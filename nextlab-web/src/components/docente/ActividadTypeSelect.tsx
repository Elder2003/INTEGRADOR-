import type { ActividadTipo } from '../../types/academic'

const TIPO_LABELS: Record<ActividadTipo, string> = {
  practica: 'Práctica',
  tarea: 'Tarea',
  informe: 'Informe',
  examen: 'Examen',
  defensa_oral: 'Defensa Oral',
}

interface ActividadTypeSelectProps {
  value: ActividadTipo | ''
  onChange: (v: ActividadTipo) => void
  disabled?: boolean
  id?: string
}

export function ActividadTypeSelect({ value, onChange, disabled, id }: ActividadTypeSelectProps) {
  return (
    <select
      id={id}
      value={value}
      onChange={e => onChange(e.target.value as ActividadTipo)}
      disabled={disabled}
      className="bg-[#1a1a1a] border border-[#2e2e2e] text-[#f0f0f0] rounded-lg px-3 py-2 text-sm focus-visible:outline-none focus-visible:border-[#E85D24] disabled:opacity-50 disabled:cursor-not-allowed"
    >
      <option value="" disabled>Seleccionar tipo</option>
      {(Object.entries(TIPO_LABELS) as [ActividadTipo, string][]).map(([tipo, label]) => (
        <option key={tipo} value={tipo}>{label}</option>
      ))}
    </select>
  )
}
