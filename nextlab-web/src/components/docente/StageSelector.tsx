import { useState } from 'react'
import { updateStage } from '../../api/grading'
import type { InvestigaciónStage } from '../../types/academic'

interface StageSelectorProps {
  submissionId: string
  currentStage: InvestigaciónStage | null
}

export function StageSelector({ submissionId, currentStage }: StageSelectorProps) {
  const [selected, setSelected] = useState<InvestigaciónStage>(currentStage ?? 'propuesta')
  const [saving, setSaving] = useState(false)
  const [saved, setSaved] = useState(false)
  const [errorMsg, setErrorMsg] = useState<string | null>(null)

  async function handleSave() {
    setSaving(true)
    setErrorMsg(null)
    setSaved(false)
    try {
      await updateStage(submissionId, selected)
      setSaved(true)
      setTimeout(() => setSaved(false), 3000)
    } catch (err: unknown) {
      setErrorMsg((err as Error).message)
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="bg-[#1a1a1a] border border-[#2e2e2e] rounded-lg p-4 space-y-3">
      <p className="text-xs text-[#888888] uppercase tracking-wide">Etapa de investigación</p>
      <select
        value={selected}
        onChange={e => setSelected(e.target.value as InvestigaciónStage)}
        disabled={saving}
        className="bg-[#111111] border border-[#2e2e2e] rounded px-3 py-1.5 text-[#f0f0f0] text-sm focus:border-[#E85D24] focus:outline-none disabled:opacity-50"
      >
        <option value="propuesta">Propuesta</option>
        <option value="desarrollo">Desarrollo</option>
        <option value="revisión">Revisión</option>
        <option value="aprobado">Aprobado</option>
      </select>
      <div className="flex items-center gap-3">
        <button
          type="button"
          onClick={handleSave}
          disabled={saving}
          className="px-4 py-2 bg-[#1a1a1a] border border-[#2e2e2e] text-[#f0f0f0] text-sm rounded-lg hover:border-[#E85D24] disabled:opacity-50 transition-colors"
        >
          {saving ? 'Guardando...' : 'Actualizar etapa'}
        </button>
        {saved && <span className="text-green-400 text-xs">Etapa actualizada ✓</span>}
      </div>
      {errorMsg && <p className="text-red-400 text-xs">{errorMsg}</p>}
    </div>
  )
}
