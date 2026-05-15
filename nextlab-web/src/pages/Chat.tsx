import { useState, useRef, useEffect, type FormEvent, type KeyboardEvent } from 'react'
import { useNavigate } from 'react-router-dom'
import { apiSendMessage, type ChatMessage } from '../api/chat'

export default function Chat() {
  const navigate = useNavigate()
  const [messages, setMessages] = useState<ChatMessage[]>([])
  const [input, setInput] = useState('')
  const [isLoading, setIsLoading] = useState(false)
  const bottomRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages, isLoading])

  function handleLogout() {
    localStorage.removeItem('access_token')
    navigate('/login')
  }

  async function handleSend() {
    const text = input.trim()
    if (!text || isLoading) return

    const userMsg: ChatMessage = { role: 'user', content: text }
    const next = [...messages, userMsg]
    setMessages(next)
    setInput('')
    setIsLoading(true)

    try {
      const response = await apiSendMessage(text, messages)
      setMessages([...next, { role: 'assistant', content: response }])
    } catch (err) {
      const detail = err instanceof Error ? err.message : 'Something went wrong'
      setMessages([...next, { role: 'assistant', content: `Error: ${detail}` }])
    } finally {
      setIsLoading(false)
    }
  }

  function handleSubmit(e: FormEvent) {
    e.preventDefault()
    handleSend()
  }

  function handleKeyDown(e: KeyboardEvent<HTMLTextAreaElement>) {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      handleSend()
    }
  }

  return (
    <div
      style={{
        width: '100%',
        height: '100svh',
        display: 'flex',
        flexDirection: 'column',
      }}
    >
      {/* Header */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          padding: '12px 20px',
          background: 'var(--card-bg)',
          borderBottom: '1px solid var(--border)',
          flexShrink: 0,
        }}
      >
        <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
          <div
            style={{
              width: 24,
              height: 24,
              borderRadius: 6,
              background: 'var(--accent)',
            }}
          />
          <span style={{ color: 'var(--text-h)', fontWeight: 700, fontSize: 16 }}>
            Nextlab
          </span>
        </div>
        <button
          onClick={handleLogout}
          style={{
            padding: '7px 16px',
            background: 'transparent',
            color: 'var(--text)',
            border: '1px solid var(--border)',
            borderRadius: 8,
            fontSize: 13,
            fontWeight: 500,
            cursor: 'pointer',
            transition: 'border-color 0.15s, color 0.15s',
          }}
          onMouseEnter={e => {
            e.currentTarget.style.borderColor = 'var(--accent)'
            e.currentTarget.style.color = 'var(--accent)'
          }}
          onMouseLeave={e => {
            e.currentTarget.style.borderColor = 'var(--border)'
            e.currentTarget.style.color = 'var(--text)'
          }}
        >
          Log out
        </button>
      </div>

      {/* Messages */}
      <div
        style={{
          flex: 1,
          overflowY: 'auto',
          padding: '24px 20px',
          display: 'flex',
          flexDirection: 'column',
          gap: 12,
        }}
      >
        {messages.length === 0 && (
          <div
            style={{
              margin: 'auto',
              textAlign: 'center',
              color: 'var(--text)',
              fontSize: 14,
            }}
          >
            Send a message to start chatting.
          </div>
        )}

        {messages.map((msg, i) => {
          const isUser = msg.role === 'user'
          const isError = msg.content.startsWith('Error:')
          return (
            <div
              key={i}
              style={{
                display: 'flex',
                justifyContent: isUser ? 'flex-end' : 'flex-start',
              }}
            >
              <div
                style={{
                  maxWidth: '75%',
                  padding: '10px 14px',
                  borderRadius: isUser ? '16px 16px 4px 16px' : '16px 16px 16px 4px',
                  background: isUser
                    ? 'var(--accent)'
                    : isError
                      ? 'rgba(239,68,68,0.1)'
                      : 'var(--card-bg)',
                  border: isUser
                    ? 'none'
                    : isError
                      ? '1px solid rgba(239,68,68,0.3)'
                      : '1px solid var(--border)',
                  color: isUser ? 'var(--accent-text)' : isError ? 'var(--error)' : 'var(--text-h)',
                  fontSize: 14,
                  lineHeight: 1.6,
                  whiteSpace: 'pre-wrap',
                  wordBreak: 'break-word',
                }}
              >
                {msg.content}
              </div>
            </div>
          )
        })}

        {isLoading && (
          <div style={{ display: 'flex', justifyContent: 'flex-start' }}>
            <div
              style={{
                padding: '10px 14px',
                borderRadius: '16px 16px 16px 4px',
                background: 'var(--card-bg)',
                border: '1px solid var(--border)',
                color: 'var(--text)',
                fontSize: 14,
              }}
            >
              ···
            </div>
          </div>
        )}

        <div ref={bottomRef} />
      </div>

      {/* Input bar */}
      <form
        onSubmit={handleSubmit}
        style={{
          display: 'flex',
          gap: 10,
          padding: '14px 20px',
          background: 'var(--card-bg)',
          borderTop: '1px solid var(--border)',
          flexShrink: 0,
        }}
      >
        <textarea
          rows={1}
          value={input}
          onChange={e => setInput(e.target.value)}
          onKeyDown={handleKeyDown}
          disabled={isLoading}
          placeholder="Message Nextlab… (Enter to send, Shift+Enter for newline)"
          style={{
            flex: 1,
            padding: '10px 12px',
            background: 'var(--input-bg)',
            border: '1px solid var(--input-border)',
            borderRadius: 8,
            color: 'var(--text-h)',
            fontSize: 14,
            outline: 'none',
            resize: 'none',
            fontFamily: 'inherit',
            lineHeight: 1.5,
            transition: 'border-color 0.15s',
          }}
          onFocus={e => (e.currentTarget.style.borderColor = 'var(--input-focus)')}
          onBlur={e => (e.currentTarget.style.borderColor = 'var(--input-border)')}
        />
        <button
          type="submit"
          disabled={isLoading || !input.trim()}
          style={{
            padding: '10px 20px',
            background:
              isLoading || !input.trim() ? '#7c3e14' : 'var(--accent)',
            color: 'var(--accent-text)',
            border: 'none',
            borderRadius: 8,
            fontSize: 14,
            fontWeight: 600,
            cursor: isLoading || !input.trim() ? 'not-allowed' : 'pointer',
            transition: 'background 0.15s',
            alignSelf: 'flex-end',
          }}
          onMouseEnter={e => {
            if (!isLoading && input.trim())
              e.currentTarget.style.background = 'var(--accent-hover)'
          }}
          onMouseLeave={e => {
            if (!isLoading && input.trim())
              e.currentTarget.style.background = 'var(--accent)'
          }}
        >
          Send
        </button>
      </form>
    </div>
  )
}
