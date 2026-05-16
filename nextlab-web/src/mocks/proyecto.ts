import type { Proyecto } from '../types'

export const proyecto: Proyecto = {
  id: 'act-005',
  tipo: 'proyecto',
  titulo: 'Proyecto final — Sistema contable',
  materia: 'Ingeniería de Software',
  docente: 'Ing. Carlos Mendoza',
  fechaLimite: '2026-06-30T23:59:00',
  estado: 'activa',
  auditoria: true,
  descripcion:
    'Sistema contable completo con módulos de autenticación, gestión de cuentas y reportes financieros. Desarrollado en equipo.',
  etapas: [
    { id: 'eta-001', nombre: 'Diagrama de BD', estado: 'entregada', fechaLimite: '2026-05-10T23:59:00' },
    { id: 'eta-002', nombre: 'Módulo de autenticación', estado: 'activa', fechaLimite: '2026-05-20T23:59:00' },
    { id: 'eta-003', nombre: 'Módulo contable', estado: 'pendiente', fechaLimite: '2026-06-05T23:59:00' },
    { id: 'eta-004', nombre: 'Sistema completo + informe', estado: 'pendiente', fechaLimite: '2026-06-30T23:59:00' },
  ],
  defensa: {
    fecha: '2026-07-05T10:00:00',
    id: 'def-001',
  },
}
