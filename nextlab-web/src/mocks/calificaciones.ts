import type { Calificacion } from '../types'

export const calificaciones: Calificacion[] = [
  {
    id: 'cal-001',
    actividad: 'Examen parcial — Algoritmos',
    tipo: 'examen',
    materia: 'Estructura de Datos',
    nota: 68,
    notaMaxima: 100,
    fecha: '2026-05-10',
    tieneAlerta: true,
    mensajeDocente:
      'Se detectaron patrones inusuales en tu proceso de resolución. La calificación refleja mi evaluación del trabajo presentado.',
    puedeApelar: true,
  },
  {
    id: 'cal-002',
    actividad: 'Tarea — Árbol binario',
    tipo: 'tarea',
    materia: 'Estructura de Datos',
    nota: 90,
    notaMaxima: 100,
    fecha: '2026-05-08',
    tieneAlerta: false,
    mensajeDocente: null,
    puedeApelar: true,
  },
  {
    id: 'cal-003',
    actividad: 'Informe — Análisis de complejidad',
    tipo: 'informe',
    materia: 'Análisis de Algoritmos',
    nota: 78,
    notaMaxima: 100,
    fecha: '2026-05-05',
    tieneAlerta: false,
    mensajeDocente:
      'Buen trabajo. El formato del informe tuvo observaciones menores — revisa el pie de figura en las imágenes.',
    puedeApelar: true,
  },
]
