import { NavLink } from 'react-router-dom'
import { useAuth } from '../../context/AuthContext'
import type { Role } from '../../types'

interface NavItem {
  label: string
  to: string
}

function navItemsFor(role: Role): NavItem[] {
  switch (role) {
    case 'estudiante':
      return [{ label: 'My Courses', to: '/student/dashboard' }]
    case 'docente':
      return [
        { label: 'My Courses', to: '/docente/dashboard' },
        { label: 'Analytics', to: '/docente/analytics' },
      ]
    case 'co_docente':
      return [{ label: 'Granted Activities', to: '/codocente/dashboard' }]
    case 'admin':
      return [
        { label: 'Carreras', to: '/admin/carreras' },
        { label: 'Materias', to: '/admin/materias' },
        { label: 'Student Profiles', to: '/admin/dashboard' },
      ]
  }
}

export function Sidebar() {
  const { user } = useAuth()
  if (!user) return null

  const items = navItemsFor(user.role)

  return (
    <nav className="w-56 bg-[#1a1a1a] border-r border-[#2e2e2e] flex flex-col py-4">
      {items.map((item) => (
        <NavLink
          key={item.to}
          to={item.to}
          className={({ isActive }) =>
            `flex items-center gap-3 px-4 py-2 text-sm transition-colors ${
              isActive
                ? 'border-l-4 border-[#E85D24] text-[#E85D24] bg-[#242424] pl-3'
                : 'border-l-4 border-transparent text-[#888888] hover:text-[#f0f0f0] pl-3'
            }`
          }
        >
          {item.label}
        </NavLink>
      ))}
    </nav>
  )
}
