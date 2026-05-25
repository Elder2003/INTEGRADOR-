import { Navigate, Outlet, useLocation } from 'react-router-dom'
import { useAuth } from '../../context/AuthContext'
import { LoadingSpinner } from './LoadingSpinner'
import { roleHomeFor } from '../../router'
import type { Role } from '../../types'

interface Props {
  allowedRoles: Role[]
}

export function ProtectedRoute({ allowedRoles }: Props) {
  const { user, isLoading } = useAuth()
  const location = useLocation()

  if (isLoading) return <LoadingSpinner />
  if (!user) return <Navigate to="/login" state={{ from: location }} replace />
  if (!allowedRoles.includes(user.role)) return <Navigate to={roleHomeFor(user.role)} replace />

  return <Outlet />
}
