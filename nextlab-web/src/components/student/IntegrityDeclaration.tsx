interface Props {
  checked: boolean
  onChange: (checked: boolean) => void
}

export function IntegrityDeclaration({ checked, onChange }: Props) {
  return (
    <label
      className={`flex items-start gap-2.5 cursor-pointer px-3.5 py-3 rounded-md border transition-colors ${
        checked
          ? 'border-[#E85D24] bg-[#E85D24]/10'
          : 'border-[#2e2e2e] bg-transparent'
      }`}
    >
      <input
        type="checkbox"
        checked={checked}
        onChange={e => onChange(e.target.checked)}
        className="mt-0.5 w-4 h-4 shrink-0 cursor-pointer accent-[#E85D24]"
      />
      <span className="text-sm text-[#888888] leading-relaxed">
        Declaro que este trabajo es de mi autoría y que no he utilizado herramientas de inteligencia
        artificial generativa no declaradas en su elaboración. Entiendo que el sistema de auditoría
        ha registrado mi proceso de resolución y que cualquier anomalía detectada será revisada por
        mi docente.
      </span>
    </label>
  )
}
