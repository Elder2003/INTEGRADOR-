import { Link } from 'react-router-dom'

export function NotFoundPage() {
  return (
    <div className="fixed inset-0 flex flex-col items-center justify-center bg-[#111111]">
      <h1 className="text-4xl font-bold text-[#f0f0f0]">404</h1>
      <p className="mt-2 text-[#888888]">Page not found</p>
      <Link to="/" className="mt-6 text-[#E85D24] hover:underline text-sm">
        Back to home
      </Link>
    </div>
  )
}
