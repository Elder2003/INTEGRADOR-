import { useState, useEffect } from 'react'
import { Link } from 'react-router-dom'
import { listAlerts, dismissAlert } from '../../api/analytics'
import type { Alert } from '../../types/academic'
import { ApiError } from '../../types'
import { LoadingSpinner } from '../../components/common/LoadingSpinner'
import { ErrorMessage } from '../../components/common/ErrorMessage'

function severityClass(severity: string): string {
  switch (severity.toLowerCase()) {
    case 'high': return 'text-red-400 bg-red-400/10'
    case 'medium': return 'text-yellow-400 bg-yellow-400/10'
    case 'low': return 'text-green-400 bg-green-400/10'
    default: return 'text-[#888888] bg-[#242424]'
  }
}

export function AlertsPage() {
  const [alerts, setAlerts] = useState<Alert[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [dismissErrors, setDismissErrors] = useState<Record<string, string>>({})

  useEffect(() => {
    listAlerts()
      .then(setAlerts)
      .catch(err => setError((err as ApiError).message))
      .finally(() => setLoading(false))
  }, [])

  async function handleDismiss(alert: Alert) {
    setAlerts(prev => prev.filter(a => a.id !== alert.id))
    setDismissErrors(prev => { const next = { ...prev }; delete next[alert.id]; return next })
    try {
      await dismissAlert(alert.id)
    } catch (err) {
      setAlerts(prev => {
        const exists = prev.some(a => a.id === alert.id)
        return exists ? prev : [alert, ...prev]
      })
      setDismissErrors(prev => ({
        ...prev,
        [alert.id]: (err as ApiError).message ?? 'No se pudo descartar la alerta.',
      }))
    }
  }

  if (loading) return <LoadingSpinner />
  if (error) return <ErrorMessage message={error} />

  return (
    <div className="p-8 max-w-2xl">
      <h1 className="text-2xl font-semibold text-[#f0f0f0] mb-6">Alertas</h1>

      {alerts.length === 0 && (
        <p className="text-[#888888]">Sin alertas activas.</p>
      )}

      <div className="space-y-3">
        {alerts.map(alert => (
          <div
            key={alert.id}
            className="bg-[#1a1a1a] border border-[#2e2e2e] rounded-xl p-4"
          >
            <div className="flex items-start justify-between gap-4">
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2 mb-1 flex-wrap">
                  <span className="text-[#f0f0f0] font-medium text-sm capitalize">
                    {alert.type.replace(/_/g, ' ')}
                  </span>
                  <span className={`text-xs px-2 py-0.5 rounded ${severityClass(alert.severity)}`}>
                    {alert.severity}
                  </span>
                </div>
                <p className="text-[#888888] text-sm mb-2">{alert.description}</p>
                <div className="flex items-center gap-3 text-xs text-[#888888]">
                  <span>
                    {alert.actividad_id && alert.curso_id ? (
                      <Link
                        to={`/docente/cursos/${alert.curso_id}/actividades/${alert.actividad_id}`}
                        className="text-[#E85D24] hover:underline"
                      >
                        {alert.actividad_title}
                      </Link>
                    ) : (
                      <span>{alert.actividad_title}</span>
                    )}
                  </span>
                  <span>·</span>
                  <span>{new Date(alert.created_at).toLocaleDateString()}</span>
                </div>
                {dismissErrors[alert.id] && (
                  <p className="text-red-400 text-xs mt-2">{dismissErrors[alert.id]}</p>
                )}
              </div>
              <button
                type="button"
                onClick={() => handleDismiss(alert)}
                className="shrink-0 px-3 py-1.5 text-xs border border-[#2e2e2e] text-[#888888] rounded-lg hover:border-[#E85D24] hover:text-[#f0f0f0] transition-colors"
              >
                Descartar
              </button>
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}
