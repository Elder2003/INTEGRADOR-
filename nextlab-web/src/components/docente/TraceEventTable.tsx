import type { TraceLogEntry } from '../../types/academic'

interface Props {
  events: TraceLogEntry[]
}

const typeBadge: Record<TraceLogEntry['type'], string> = {
  keystroke: 'bg-[#2e2e2e] text-[#888888]',
  copy: 'bg-yellow-400/10 text-yellow-400',
  paste: 'bg-yellow-400/10 text-yellow-400',
  focus_in: 'bg-blue-400/10 text-blue-400',
  focus_out: 'bg-blue-400/10 text-blue-400',
}

export function TraceEventTable({ events }: Props) {
  if (events.length === 0) {
    return (
      <p className="text-[#888888] py-4">No hay eventos de trazas registrados.</p>
    )
  }

  return (
    <div className="overflow-x-auto">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-[#2e2e2e] text-left">
            <th className="py-2 pr-4 text-[#888888] font-medium">Tipo</th>
            <th className="py-2 pr-4 text-[#888888] font-medium">Timestamp</th>
            <th className="py-2 text-[#888888] font-medium">Contenido</th>
          </tr>
        </thead>
        <tbody>
          {events.map((event) => (
            <tr key={event.id} className="border-b border-[#2e2e2e]">
              <td className="py-2 pr-4">
                <span
                  className={`inline-block text-xs px-2 py-0.5 rounded ${typeBadge[event.type]}`}
                >
                  {event.type}
                </span>
              </td>
              <td className="py-2 pr-4 text-[#888888] text-xs whitespace-nowrap">
                {new Date(event.timestamp).toLocaleString('es-BO')}
              </td>
              <td className="py-2 text-[#f0f0f0] text-xs">
                {event.content ?? (
                  <span className="text-[#888888] italic">—</span>
                )}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
