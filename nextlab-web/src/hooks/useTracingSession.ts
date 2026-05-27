import { useCallback, useEffect, useRef, useState } from 'react'
import { createSession, endSession } from '../api/tracing'
import { useAuth } from '../context/AuthContext'

export type SessionStatus =
  | 'idle'
  | 'starting'
  | 'connecting'
  | 'active'
  | 'reconnecting'
  | 'ended'
  | 'error'

interface TracingSessionState {
  status: SessionStatus
  sessionId: string | null
  startTime: string | null
  startSession: () => Promise<void>
  terminateSession: () => Promise<void>
}

export function useTracingSession(aid: string, traceEnabled: boolean): TracingSessionState {
  const { accessToken } = useAuth()
  const [status, setStatus] = useState<SessionStatus>('idle')
  const [sessionId, setSessionId] = useState<string | null>(null)
  const [startTime, setStartTime] = useState<string | null>(null)

  const wsRef = useRef<WebSocket | null>(null)
  const reconnectAttemptsRef = useRef(0)
  const listenersCleanupRef = useRef<(() => void) | null>(null)

  const removeListeners = useCallback(() => {
    if (listenersCleanupRef.current) {
      listenersCleanupRef.current()
      listenersCleanupRef.current = null
    }
  }, [])

  const sendEvent = useCallback(
    (type: string, content?: string) => {
      if (wsRef.current?.readyState === WebSocket.OPEN) {
        const msg: Record<string, string> = { type, timestamp: new Date().toISOString() }
        if (content !== undefined) msg.content = content.slice(0, 500)
        wsRef.current.send(JSON.stringify(msg))
      }
    },
    [],
  )

  const attachListeners = useCallback(() => {
    const onCopy = (e: ClipboardEvent) => {
      sendEvent('copy', e.clipboardData?.getData('text') ?? '')
    }
    const onPaste = (e: ClipboardEvent) => {
      sendEvent('paste', e.clipboardData?.getData('text') ?? '')
    }
    const onKeydown = () => sendEvent('keystroke')
    const onBlur = () => sendEvent('focus_out')
    const onFocus = () => sendEvent('focus_in')

    document.addEventListener('copy', onCopy)
    document.addEventListener('paste', onPaste)
    document.addEventListener('keydown', onKeydown)
    window.addEventListener('blur', onBlur)
    window.addEventListener('focus', onFocus)

    listenersCleanupRef.current = () => {
      document.removeEventListener('copy', onCopy)
      document.removeEventListener('paste', onPaste)
      document.removeEventListener('keydown', onKeydown)
      window.removeEventListener('blur', onBlur)
      window.removeEventListener('focus', onFocus)
    }
  }, [sendEvent])

  const openWebSocket = useCallback(
    (sid: string, token: string) => {
      const wsUrl = `${import.meta.env.VITE_API_URL.replace('http', 'ws')}/sessions/${sid}/ws?token=${token}`
      const ws = new WebSocket(wsUrl)
      wsRef.current = ws

      ws.onopen = () => {
        reconnectAttemptsRef.current = 0
        setStatus('active')
        attachListeners()
      }

      ws.onmessage = (event) => {
        try {
          const msg = JSON.parse(event.data as string)
          if (msg.type === 'ping') {
            ws.send(JSON.stringify({ type: 'pong' }))
          }
        } catch {
          // ignore non-JSON messages
        }
      }

      ws.onclose = () => {
        removeListeners()
        if (reconnectAttemptsRef.current < 3) {
          const attempt = reconnectAttemptsRef.current
          reconnectAttemptsRef.current += 1
          setStatus('reconnecting')
          setTimeout(() => {
            if (token) openWebSocket(sid, token)
          }, 1000 * Math.pow(2, attempt))
        } else {
          setStatus('error')
        }
      }

      ws.onerror = () => {
        ws.close()
      }
    },
    [attachListeners, removeListeners],
  )

  const startSession = useCallback(async () => {
    if (!traceEnabled || !accessToken) return
    setStatus('starting')
    try {
      const session = await createSession(aid)
      setSessionId(session.id)
      setStartTime(session.started_at)
      setStatus('connecting')
      openWebSocket(session.id, accessToken)
    } catch {
      setStatus('error')
    }
  }, [aid, traceEnabled, accessToken, openWebSocket])

  const terminateSession = useCallback(async () => {
    removeListeners()
    if (wsRef.current) {
      wsRef.current.onclose = null
      wsRef.current.close()
      wsRef.current = null
    }
    if (sessionId) {
      try {
        await endSession(sessionId)
      } catch {
        // session may already be ended server-side
      }
    }
    setStatus('ended')
  }, [sessionId, removeListeners])

  useEffect(() => {
    return () => {
      removeListeners()
      if (wsRef.current) {
        wsRef.current.onclose = null
        wsRef.current.close()
        wsRef.current = null
      }
    }
  }, [removeListeners])

  if (!traceEnabled) {
    return {
      status: 'idle',
      sessionId: null,
      startTime: null,
      startSession: async () => {},
      terminateSession: async () => {},
    }
  }

  return { status, sessionId, startTime, startSession, terminateSession }
}
