import type { SessionStatus } from '../../hooks/useTracingSession'

interface Props {
  status: SessionStatus
  startTime: string | null
  onStart: () => void
  onEnd: () => void
}

function formatTime(iso: string): string {
  return new Date(iso).toLocaleString('es-BO', {
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  })
}

export function SessionPanel({ status, startTime, onStart, onEnd }: Props) {
  if (status === 'idle') {
    return (
      <div className="bg-[#1a1a1a] border border-[#2e2e2e] rounded-lg p-4">
        <p className="text-[#888888] text-sm">
          Esta actividad requiere monitoreo de trazabilidad.
        </p>
        <button
          type="button"
          onClick={onStart}
          className="mt-3 px-4 py-2 bg-[#E85D24] text-white text-sm rounded-lg hover:bg-[#d4521f] transition-colors"
        >
          Iniciar sesión de monitoreo
        </button>
      </div>
    )
  }

  if (status === 'starting' || status === 'connecting') {
    return (
      <div className="bg-[#1a1a1a] border border-[#2e2e2e] rounded-lg p-4 flex items-center gap-3">
        <div className="w-4 h-4 rounded-full border-2 border-[#E85D24] border-t-transparent animate-spin" />
        <span className="text-[#888888] text-sm">Conectando...</span>
      </div>
    )
  }

  if (status === 'active') {
    return (
      <div className="bg-[#1a1a1a] border border-[#2e2e2e] rounded-lg p-4 flex items-center justify-between">
        <div className="flex items-center gap-2">
          <span className="w-2 h-2 rounded-full bg-green-500 animate-pulse" />
          <span className="text-green-400 text-sm font-medium">Sesión activa</span>
          {startTime && (
            <span className="text-[#888888] text-xs">desde {formatTime(startTime)}</span>
          )}
        </div>
        <button
          type="button"
          onClick={onEnd}
          className="text-xs border border-[#2e2e2e] text-[#888888] px-3 py-1 rounded hover:border-red-400 hover:text-red-400 transition-colors"
        >
          Terminar sesión
        </button>
      </div>
    )
  }

  if (status === 'reconnecting') {
    return (
      <div className="bg-[#1a1a1a] border border-[#2e2e2e] rounded-lg p-4 flex items-center gap-2">
        <span className="w-2 h-2 rounded-full bg-yellow-400" />
        <span className="text-yellow-400 text-sm">Conexión perdida — reconectando…</span>
      </div>
    )
  }

  if (status === 'ended') {
    return (
      <div className="bg-[#1a1a1a] border border-[#2e2e2e] rounded-lg p-4 space-y-2">
        <div className="flex items-center gap-2">
          <span className="w-2 h-2 rounded-full bg-[#888888]" />
          <span className="text-[#888888] text-sm">Sesión terminada</span>
          <span className="text-[#888888] text-xs">a las {formatTime(new Date().toISOString())}</span>
        </div>
        <button
          type="button"
          onClick={onStart}
          className="text-xs border border-[#2e2e2e] text-[#888888] px-3 py-1 rounded hover:border-[#E85D24] hover:text-[#E85D24] transition-colors"
        >
          Iniciar nueva sesión
        </button>
      </div>
    )
  }

  // status === 'error'
  return (
    <div className="bg-[#1a1a1a] border border-red-400/30 rounded-lg p-4 space-y-2">
      <span className="inline-block text-xs bg-red-400/10 text-red-400 px-2 py-0.5 rounded">
        Error de conexión — sesión interrumpida
      </span>
      <div>
        <button
          type="button"
          onClick={onStart}
          className="text-xs border border-[#2e2e2e] text-[#888888] px-3 py-1 rounded hover:border-[#E85D24] hover:text-[#E85D24] transition-colors"
        >
          Reintentar
        </button>
      </div>
    </div>
  )
}
