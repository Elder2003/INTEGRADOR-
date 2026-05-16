import { createContext, useContext, useState, type ReactNode } from 'react'
import type { Usuario } from '../types'

interface StudentContextValue {
  usuario: Usuario
}

const StudentContext = createContext<StudentContextValue | null>(null)

const usuarioMock: Usuario = {
  id: 'est-001',
  nombre: 'Elder Omar Gutierrez Rosales',
  carrera: 'Ingeniería de Sistemas',
  semestre: '6to Semestre',
}

export function StudentProvider({ children }: { children: ReactNode }) {
  const [usuario] = useState<Usuario>(usuarioMock)
  return (
    <StudentContext.Provider value={{ usuario }}>
      {children}
    </StudentContext.Provider>
  )
}

export function useStudent() {
  const ctx = useContext(StudentContext)
  if (!ctx) throw new Error('useStudent must be used within StudentProvider')
  return ctx
}
