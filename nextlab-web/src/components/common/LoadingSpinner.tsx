interface Props {
  message?: string
}

export function LoadingSpinner({ message }: Props) {
  return (
    <div className="fixed inset-0 flex flex-col items-center justify-center bg-[#111111]">
      <div className="h-12 w-12 rounded-full border-4 border-[#2e2e2e] border-t-[#E85D24] animate-spin" />
      {message && <p className="mt-4 text-[#888888] text-sm">{message}</p>}
    </div>
  )
}
