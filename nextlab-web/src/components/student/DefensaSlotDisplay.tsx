import type { DefensaSlot } from '../../types/academic'

interface DefensaSlotDisplayProps {
  slot: DefensaSlot | null
}

export function DefensaSlotDisplay({ slot }: DefensaSlotDisplayProps) {
  if (slot === null) {
    return (
      <div className="bg-[#1a1a1a] border border-[#2e2e2e] rounded-lg p-4">
        <p className="text-[#888888] text-sm">Defensa aún no programada.</p>
      </div>
    )
  }

  return (
    <div className="bg-[#1a1a1a] border border-[#2e2e2e] rounded-lg p-4 space-y-1">
      <p className="text-[#f0f0f0] text-sm font-medium">Tu defensa está programada</p>
      <p className="text-[#f0f0f0]">
        {new Date(slot.scheduled_at).toLocaleString('es-BO', {
          dateStyle: 'full',
          timeStyle: 'short',
        })}
      </p>
      {slot.location && <p className="text-[#888888] text-sm">{slot.location}</p>}
    </div>
  )
}
