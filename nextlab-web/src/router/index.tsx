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
import { SubmissionsPage } from '../pages/docente/SubmissionsPage'
import { SubmissionDetailPage } from '../pages/docente/SubmissionDetailPage'
import { CodocenteSubmissionsPage } from '../pages/codocente/SubmissionsPage'
import { DashboardPage as AdminDashboard } from '../pages/admin/DashboardPage'
import { CarrerasPage } from '../pages/admin/CarrerasPage'
import { MateriasPage } from '../pages/admin/MateriasPage'
import { StudentProfilesPage } from '../pages/admin/StudentProfilesPage'
import { NotFoundPage } from '../pages/NotFoundPage'
import { TraceViewerPage } from '../pages/docente/TraceViewerPage'
import { LiveSessionsPage } from '../pages/docente/LiveSessionsPage'
import { LiveSessionFeedPage } from '../pages/docente/LiveSessionFeedPage'
import { CodocenteTraceViewerPage } from '../pages/codocente/TraceViewerPage'
import { CodocenteLiveSessionsPage } from '../pages/codocente/LiveSessionsPage'
import { CodocenteLiveSessionFeedPage } from '../pages/codocente/LiveSessionFeedPage'
import { DefensaSchedulingPage } from '../pages/docente/DefensaSchedulingPage'
import { GitAnalysisPage } from '../pages/docente/GitAnalysisPage'
import { AIReportPage } from '../pages/docente/AIReportPage'
import { DetectionReportPage } from '../pages/docente/DetectionReportPage'

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
          { path: '/docente/cursos/:cid/actividades/:aid/submissions', element: <SubmissionsPage /> },
          { path: '/docente/cursos/:cid/actividades/:aid/submissions/:sid', element: <SubmissionDetailPage /> },
          { path: '/docente/cursos/:cid/actividades/:aid/sessions', element: <LiveSessionsPage /> },
          { path: '/docente/cursos/:cid/actividades/:aid/sessions/:sid/live', element: <LiveSessionFeedPage /> },
          { path: '/docente/cursos/:cid/actividades/:aid/submissions/:subid/trace', element: <TraceViewerPage /> },
          { path: '/docente/cursos/:cid/actividades/:aid/defensa-slots', element: <DefensaSchedulingPage /> },
          { path: '/docente/cursos/:cid/actividades/:aid/submissions/:subid/git-analysis', element: <GitAnalysisPage /> },
          { path: '/docente/cursos/:cid/actividades/:aid/submissions/:subid/ai-report', element: <AIReportPage /> },
          { path: '/docente/cursos/:cid/actividades/:aid/submissions/:subid/detection-report', element: <DetectionReportPage /> },
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
          { path: '/codocente/cursos/:cid/actividades/:aid/submissions', element: <CodocenteSubmissionsPage /> },
          { path: '/codocente/cursos/:cid/actividades/:aid/submissions/:sid', element: <SubmissionDetailPage /> },
          { path: '/codocente/cursos/:cid/actividades/:aid/sessions', element: <CodocenteLiveSessionsPage /> },
          { path: '/codocente/cursos/:cid/actividades/:aid/sessions/:sid/live', element: <CodocenteLiveSessionFeedPage /> },
          { path: '/codocente/cursos/:cid/actividades/:aid/submissions/:subid/trace', element: <CodocenteTraceViewerPage /> },
          { path: '/codocente/cursos/:cid/actividades/:aid/submissions/:subid/git-analysis', element: <GitAnalysisPage /> },
          { path: '/codocente/cursos/:cid/actividades/:aid/submissions/:subid/ai-report', element: <AIReportPage /> },
          { path: '/codocente/cursos/:cid/actividades/:aid/submissions/:subid/detection-report', element: <DetectionReportPage /> },
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
