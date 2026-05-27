import { Link } from 'react-router-dom'

const NAV_CARDS = [
  { to: '/admin/carreras', title: 'Carreras', description: 'Gestionar carreras académicas' },
  { to: '/admin/materias', title: 'Materias', description: 'Gestionar materias por carrera' },
  { to: '/admin/student-profiles', title: 'Perfiles de Estudiantes', description: 'Gestionar perfiles y accesos de estudiantes' },
]

export function DashboardPage() {
  return (
    <div className="p-8">
      <h1 className="text-2xl font-semibold text-[#f0f0f0] mb-6">Panel de Administración</h1>
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        {NAV_CARDS.map(card => (
          <Link
            key={card.to}
            to={card.to}
            className="bg-[#1a1a1a] border border-[#2e2e2e] rounded-lg p-4 hover:border-[#E85D24] transition-colors"
          >
            <h2 className="text-[#f0f0f0] font-semibold mb-1">{card.title}</h2>
            <p className="text-sm text-[#888888]">{card.description}</p>
          </Link>
        ))}
      </div>
    </div>
  )
}
