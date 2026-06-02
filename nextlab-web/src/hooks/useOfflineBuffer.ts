import { useCallback, useEffect, useRef } from 'react'
import api from '../lib/api'

interface BufferedEvent {
  type: string
  client_timestamp: number
  metadata: Record<string, unknown>
}

const MAX_BUFFER_SIZE = 1000

/**
 * Buffers trace events when the browser is offline and flushes them
 * to the batch endpoint when connectivity is restored.
 */
export function useOfflineBuffer(sessionId: string | null) {
  const bufferRef = useRef<BufferedEvent[]>([])
  const flushingRef = useRef(false)

  const flushBuffer = useCallback(async () => {
    if (!sessionId || bufferRef.current.length === 0 || flushingRef.current) return

    flushingRef.current = true
    const events = [...bufferRef.current]
    bufferRef.current = []

    try {
      await api.post(`/student/sessions/${sessionId}/trace/batch`, { events })
    } catch {
      // Re-queue events on failure so they are not lost
      bufferRef.current = [...events, ...bufferRef.current].slice(-MAX_BUFFER_SIZE)
    } finally {
      flushingRef.current = false
    }
  }, [sessionId])

  useEffect(() => {
    const handleOnline = () => {
      flushBuffer()
    }
    window.addEventListener('online', handleOnline)
    return () => window.removeEventListener('online', handleOnline)
  }, [flushBuffer])

  /**
   * Call this instead of sending directly over WebSocket when offline.
   * Returns true if the event was buffered (offline), false if it should be sent normally.
   */
  const bufferEventIfOffline = useCallback(
    (type: string, metadata: Record<string, unknown> = {}): boolean => {
      if (navigator.onLine) return false

      if (bufferRef.current.length < MAX_BUFFER_SIZE) {
        bufferRef.current.push({
          type,
          client_timestamp: Date.now(),
          metadata,
        })
      }
      return true
    },
    [],
  )

  return { bufferEventIfOffline, flushBuffer }
}
