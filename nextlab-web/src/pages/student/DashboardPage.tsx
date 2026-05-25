import { useAuth } from '../../context/AuthContext'

export function DashboardPage() {
  const { user } = useAuth()
  return (
    <div className="p-8">
      <h1 className="text-2xl font-semibold text-[#f0f0f0]">Student Dashboard</h1>
      <p className="mt-2 text-[#888888]">{user?.email}</p>
    </div>
  )
}
