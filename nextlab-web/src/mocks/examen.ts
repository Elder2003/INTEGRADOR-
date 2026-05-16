import type { Examen, FeedbackIA } from '../types'

export const examen: Examen = {
  id: 'act-001',
  titulo: 'Examen parcial — Algoritmos y Estructuras de Datos',
  materia: 'Estructura de Datos',
  docente: 'Ing. Fernando Hinojosa',
  duracionMinutos: 45,
  preguntas: [
    {
      id: 'preg-001',
      numero: 1,
      puntos: 20,
      modalidad: 'texto',
      enunciado:
        'Explica con tus propias palabras qué es la complejidad algorítmica Big-O y por qué es importante al diseñar algoritmos eficientes.',
      placeholder: 'Escribe tu respuesta aquí...',
      maxCaracteres: 2000,
    },
    {
      id: 'preg-002',
      numero: 2,
      puntos: 30,
      modalidad: 'codigo',
      lenguaje: 'python',
      enunciado:
        'Implementa una función que reciba una lista de enteros y devuelva el segundo número más grande sin usar funciones de ordenamiento.',
      placeholder: '# Escribe tu solución aquí\n\ndef segundo_mayor(lista):\n    pass',
      maxCaracteres: 5000,
    },
    {
      id: 'preg-003',
      numero: 3,
      puntos: 25,
      modalidad: 'texto',
      enunciado:
        'Describe las diferencias entre una lista enlazada simple y una doblemente enlazada. ¿En qué casos usarías cada una?',
      placeholder: 'Escribe tu respuesta aquí...',
      maxCaracteres: 2000,
    },
    {
      id: 'preg-004',
      numero: 4,
      puntos: 25,
      modalidad: 'codigo',
      lenguaje: 'python',
      enunciado:
        'Implementa una pila (stack) usando una lista enlazada. Incluye los métodos push, pop e isEmpty.',
      placeholder:
        '# Implementa la clase Stack\n\nclass Node:\n    pass\n\nclass Stack:\n    pass',
      maxCaracteres: 5000,
    },
  ],
}

export const feedbackIA: FeedbackIA[] = [
  {
    pregunta: 1,
    feedback:
      'Tu implementación del nodo es correcta. Sin embargo, considera el caso borde cuando la lista esté vacía — ¿qué debería devolver tu función en ese caso?',
    tipo: 'sugerencia',
  },
]
