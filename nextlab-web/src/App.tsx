import { StudentProvider } from './context/StudentContext'
import StudentRouter from './router/StudentRouter'

export default function App() {
  return (
    <StudentProvider>
      <StudentRouter />
    </StudentProvider>
  )
}
