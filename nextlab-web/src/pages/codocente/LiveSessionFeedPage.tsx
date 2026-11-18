import { useEffect, useRef, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { getLiveEvents } from '../../api/tracing'
import type { TraceLogEntry } from '../../types/academic'
import { LoadingSpinner } from '../../components/common/LoadingSpinner'
import { ErrorMessage } from '../../components/common/ErrorMessage'
import { TraceEventTable } from '../../components/docente/TraceEventTable'

export function CodocenteLiveSessionFeedPage() {
  const { cid, aid, sid } = useParams<{ cid: string; aid: string; sid: string }>()

  const backPath = `/codocente/cursos/${cid}/actividades/${aid}/sessions`

  const [events, setEvents] = useState<TraceLogEntry[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [sessionEnded, setSessionEnded] = useState(false)
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null)
  const eventsRef = useRef<TraceLogEntry[]>([])

  useEffect(() => {
    eventsRef.current = events
  }, [events])

  useEffect(() => {
    if (!sid) return

    getLiveEvents(sid)
      .then((res) => setEvents(res.events))
      .catch((e: Error) => {
        if (e.message.includes('not_found') || e.message.includes('404')) {
          setSessionEnded(true)
        } else {
          setError(e.message)
        }
      })
      .finally(() => setLoading(false))

    intervalRef.current = setInterval(async () => {
      try {
        const current = eventsRef.current
        const last = current[current.length - 1]?.timestamp
        const res = await getLiveEvents(sid, last)
        if (res.events.length > 0) {
          setEvents((prev) => [...prev, ...res.events])
        }
      } catch (e: unknown) {
        const err = e as Error
        if (err.message.includes('not_found') || err.message.includes('404')) {
          setSessionEnded(true)
          if (intervalRef.current) clearInterval(intervalRef.current)
        }
      }
    }, 2000)

    return () => {
      if (intervalRef.current) clearInterval(intervalRef.current)
    }
  }, [sid])

  useEffect(() => {
    if (sessionEnded && intervalRef.current) {
      clearInterval(intervalRef.current)
    }
  }, [sessionEnded])

  if (loading) return <LoadingSpinner />

  return (
    <div className="p-8 max-w-4xl space-y-6">
      <Link to={backPath} className="text-sm text-[#888888] hover:text-[#E85D24] transition-colors">
        ← Sesiones en vivo
      </Link>

      <div className="flex items-center gap-3">
        <h1 className="text-2xl font-semibold text-[#f0f0f0]">Sesión en vivo</h1>
        {sessionEnded ? (
          <span className="text-xs bg-[#2e2e2e] text-[#888888] px-2 py-0.5 rounded">
            Sesión terminada
          </span>
        ) : (
          <span className="flex items-center gap-1.5 text-xs text-green-400">
            <span className="w-2 h-2 rounded-full bg-green-500 animate-pulse" />
            En vivo
          </span>
        )}
      </div>

      {error && <ErrorMessage message={error} />}

      <div className="bg-[#1a1a1a] border border-[#2e2e2e] rounded-lg p-4">
        {events.length === 0 && !sessionEnded ? (
          <p className="text-[#888888] py-4">Esperando eventos…</p>
        ) : (
          <TraceEventTable events={events} />
        )}
      </div>
    </div>
  )
}
