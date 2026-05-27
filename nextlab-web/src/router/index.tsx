import { createBrowserRouter, Navigate, Outlet, useLocation } from 'react-router-dom'
import type { Role } from '../types'
import { useAuth } from '../context/AuthContext'
import { LoadingSpinner } from '../components/common/LoadingSpinner'
import { ProtectedRoute } from '../components/common/ProtectedRoute'
import { Layout } from '../components/common/Layout'
import { LoginPage } from '../pages/auth/LoginPage'
import { DashboardPage as StudentDashboard } from '../pages/student/DashboardPage'
import { ActividadesPage } from '../pages/student/ActividadesPage'
import { ActividadDetailPage as StudentActividadDetailPage } from '../pages/student/ActividadDetailPage'
import { DashboardPage as DocenteDashboard } from '../pages/docente/DashboardPage'
import { CreateCursoPage } from '../pages/docente/CreateCursoPage'
import { EditCursoPage } from '../pages/docente/EditCursoPage'
import { CursoDetailPage } from '../pages/docente/CursoDetailPage'
import { CreateActividadPage } from '../pages/docente/CreateActividadPage'
import { EditActividadPage } from '../pages/docente/EditActividadPage'
import { ActividadDetailPage } from '../pages/docente/ActividadDetailPage'
import { DashboardPage as CodocenteDashboard } from '../pages/codocente/DashboardPage'
import { DashboardPage as AdminDashboard } from '../pages/admin/DashboardPage'
import { CarrerasPage } from '../pages/admin/CarrerasPage'
import { MateriasPage } from '../pages/admin/MateriasPage'
import { StudentProfilesPage } from '../pages/admin/StudentProfilesPage'
import { NotFoundPage } from '../pages/NotFoundPage'

export function roleHomeFor(role: Role): string {
  switch (role) {
    case 'docente': return '/docente/dashboard'
    case 'co_docente': return '/codocente/dashboard'
    case 'admin': return '/admin/dashboard'
    case 'estudiante': return '/student/dashboard'
  }
}

function RootRedirect() {
  const { user, isLoading } = useAuth()
  if (isLoading) return <LoadingSpinner />
  if (user) return <Navigate to={roleHomeFor(user.role)} replace />
  return <Navigate to="/login" replace />
}

function PublicOnlyRoute() {
  const { user, isLoading } = useAuth()
  if (isLoading) return <LoadingSpinner />
  if (user) return <Navigate to={roleHomeFor(user.role)} replace />
  return <Outlet />
}

const router = createBrowserRouter([
  {
    path: '/',
    element: <RootRedirect />,
  },
  {
    element: <PublicOnlyRoute />,
    children: [
      { path: '/login', element: <LoginPage /> },
    ],
  },
  {
    element: <ProtectedRoute allowedRoles={['estudiante']} />,
    children: [
      {
        element: <Layout />,
        children: [
          { path: '/student/dashboard', element: <StudentDashboard /> },
          { path: '/student/cursos/:id/actividades', element: <ActividadesPage /> },
          { path: '/student/cursos/:id/actividades/:aid', element: <StudentActividadDetailPage /> },
        ],
      },
    ],
  },
  {
    element: <ProtectedRoute allowedRoles={['docente']} />,
    children: [
      {
        element: <Layout />,
        children: [
          { path: '/docente/dashboard', element: <DocenteDashboard /> },
          // MW2 routes — cursos/new MUST precede cursos/:id to prevent "new" matching as UUID
          { path: '/docente/cursos/new', element: <CreateCursoPage /> },
          { path: '/docente/cursos/:id', element: <CursoDetailPage /> },
          { path: '/docente/cursos/:id/edit', element: <EditCursoPage /> },
          { path: '/docente/cursos/:id/actividades/new', element: <CreateActividadPage /> },
          { path: '/docente/cursos/:id/actividades/:aid', element: <ActividadDetailPage /> },
          { path: '/docente/cursos/:id/actividades/:aid/edit', element: <EditActividadPage /> },
        ],
      },
    ],
  },
  {
    element: <ProtectedRoute allowedRoles={['co_docente']} />,
    children: [
      {
        element: <Layout />,
        children: [
          { path: '/codocente/dashboard', element: <CodocenteDashboard /> },
        ],
      },
    ],
  },
  {
    element: <ProtectedRoute allowedRoles={['admin']} />,
    children: [
      {
        element: <Layout />,
        children: [
          { path: '/admin/dashboard', element: <AdminDashboard /> },
          { path: '/admin/carreras', element: <CarrerasPage /> },
          { path: '/admin/materias', element: <MateriasPage /> },
          { path: '/admin/student-profiles', element: <StudentProfilesPage /> },
        ],
      },
    ],
  },
  {
    path: '*',
    element: <NotFoundPage />,
  },
])

export default router
