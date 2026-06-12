import type { TraceLogEntry } from '../../types/academic'

interface Props {
  events: TraceLogEntry[]
}

const typeBadge: Record<string, string> = {
  keystroke: 'bg-[#2e2e2e] text-[#888888]',
  copy: 'bg-yellow-400/10 text-yellow-400',
  paste: 'bg-yellow-400/10 text-yellow-400',
  clipboard_copy: 'bg-yellow-400/10 text-yellow-400',
  clipboard_paste: 'bg-yellow-400/10 text-yellow-400',
  focus_in: 'bg-blue-400/10 text-blue-400',
  focus_out: 'bg-blue-400/10 text-blue-400',
  tab_blur: 'bg-red-400/10 text-red-400',
  tab_focus: 'bg-blue-400/10 text-blue-400',
  heartbeat: 'bg-[#2e2e2e] text-[#555555]',
  session_close: 'bg-green-400/10 text-green-400',
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
                  className={`inline-block text-xs px-2 py-0.5 rounded ${typeBadge[event.type] ?? 'bg-[#2e2e2e] text-[#888888]'}`}
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
