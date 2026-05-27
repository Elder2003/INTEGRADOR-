interface Props {
  message: string
  onDismiss?: () => void
}

export function ErrorMessage({ message, onDismiss }: Props) {
  if (!message) return null

  return (
    <div className="flex items-start gap-3 rounded bg-[#1a1a1a] border-l-4 border-[#E85D24] px-4 py-3">
      <p className="flex-1 text-sm text-[#f0f0f0]">{message}</p>
      {onDismiss && (
        <button
          type="button"
          onClick={onDismiss}
          className="text-[#888888] hover:text-[#f0f0f0] text-sm leading-none"
          aria-label="Dismiss"
        >
          ✕
        </button>
      )}
    </div>
  )
}
