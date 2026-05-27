import { useState, useRef, type DragEvent, type ChangeEvent } from 'react'
import { Upload, File, X } from 'lucide-react'

interface Props {
  acceptedTypes: string[]
  acceptedLabels: string[]
  onFileSelect: (file: File) => void
  selectedFile?: File | null
  onClear?: () => void
}

export function FileUpload({
  acceptedTypes,
  acceptedLabels,
  onFileSelect,
  selectedFile,
  onClear,
}: Props) {
  const [isDragging, setIsDragging] = useState(false)
  const inputRef = useRef<HTMLInputElement>(null)

  function handleDrop(e: DragEvent<HTMLDivElement>) {
    e.preventDefault()
    setIsDragging(false)
    const file = e.dataTransfer.files[0]
    if (file) onFileSelect(file)
  }

  function handleChange(e: ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0]
    if (file) onFileSelect(file)
  }

  if (selectedFile) {
    return (
      <div className="border border-[#2e2e2e] rounded-lg px-4 py-3 flex items-center justify-between bg-[#1a1a1a]">
        <div className="flex items-center gap-2.5">
          <File size={20} className="text-[#E85D24] shrink-0" />
          <div>
            <div className="text-[#f0f0f0] font-medium text-sm">{selectedFile.name}</div>
            <div className="text-[#888888] text-xs">{(selectedFile.size / 1024).toFixed(0)} KB</div>
          </div>
        </div>
        {onClear && (
          <button
            type="button"
            onClick={onClear}
            className="text-[#888888] hover:text-[#f0f0f0] transition-colors p-1"
            title="Quitar archivo"
          >
            <X size={16} />
          </button>
        )}
      </div>
    )
  }

  return (
    <div
      onClick={() => inputRef.current?.click()}
      onDrop={handleDrop}
      onDragOver={e => {
        e.preventDefault()
        setIsDragging(true)
      }}
      onDragLeave={() => setIsDragging(false)}
      className={`border-2 border-dashed rounded-lg py-8 px-6 text-center cursor-pointer transition-colors ${
        isDragging
          ? 'border-[#E85D24] bg-[#E85D24]/5'
          : 'border-[#2e2e2e] hover:border-[#888888]'
      }`}
    >
      <Upload size={28} className="text-[#888888] mx-auto mb-2.5" />
      <div className="text-[#f0f0f0] font-medium text-sm mb-1.5">
        Arrastra tu archivo aquí o haz clic para seleccionar
      </div>
      <div className="text-[#888888] text-sm">
        Tipos aceptados: {acceptedLabels.join(', ')}
      </div>
      <input
        ref={inputRef}
        type="file"
        accept={acceptedTypes.join(',')}
        onChange={handleChange}
        className="hidden"
      />
    </div>
  )
}
