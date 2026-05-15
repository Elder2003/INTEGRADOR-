const API_BASE = 'http://localhost:8000'

export interface ChatMessage {
  role: 'user' | 'assistant'
  content: string
}

export async function apiSendMessage(
  message: string,
  history: ChatMessage[],
): Promise<string> {
  const token = localStorage.getItem('access_token')
  if (!token) throw new Error('Not authenticated')

  const res = await fetch(`${API_BASE}/chat`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify({ message, history }),
  })

  if (res.status === 401) {
    localStorage.removeItem('access_token')
    window.location.href = '/login'
    throw new Error('Session expired')
  }

  if (!res.ok) {
    const err = await res.json().catch(() => ({ detail: 'Unknown error' }))
    throw new Error(err.detail ?? 'Request failed')
  }

  const data: { response: string } = await res.json()
  return data.response
}
