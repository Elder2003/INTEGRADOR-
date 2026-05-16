import { useState, useRef, type DragEvent, type ChangeEvent } from 'react'
import { Upload, File, X } from 'lucide-react'

interface Props {
  acceptedTypes: string[]
  acceptedLabels: string[]
  onFileSelect: (file: File) => void
  selectedFile?: File | null
  onClear?: () => void
}

export default function FileUpload({
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
      <div
        style={{
          border: '1px solid var(--color-success)',
          borderRadius: 8,
          padding: '14px 18px',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          backgroundColor: 'rgba(63,166,110,0.05)',
        }}
      >
        <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
          <File size={20} color="var(--color-success)" />
          <div>
            <div
              style={{
                color: 'var(--color-text-primary)',
                fontWeight: 500,
                marginBottom: 2,
              }}
            >
              {selectedFile.name}
            </div>
            <div style={{ fontSize: 'var(--font-size-xs)', color: 'var(--color-text-secondary)' }}>
              {(selectedFile.size / 1024).toFixed(0)} KB
            </div>
          </div>
        </div>
        {onClear && (
          <button
            onClick={onClear}
            style={{ color: 'var(--color-text-muted)', padding: 4 }}
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
      style={{
        border: `2px dashed ${isDragging ? 'var(--color-accent)' : 'var(--color-border)'}`,
        borderRadius: 8,
        padding: '32px 24px',
        textAlign: 'center',
        cursor: 'pointer',
        backgroundColor: isDragging ? 'var(--color-accent-dim)' : 'transparent',
        transition: 'border-color 0.2s, background-color 0.2s',
      }}
    >
      <Upload
        size={28}
        color="var(--color-text-muted)"
        style={{ marginBottom: 10, display: 'block', margin: '0 auto 10px' }}
      />
      <div
        style={{
          color: 'var(--color-text-primary)',
          fontWeight: 500,
          marginBottom: 6,
        }}
      >
        Arrastra tu archivo aquí o haz clic para seleccionar
      </div>
      <div style={{ fontSize: 'var(--font-size-sm)', color: 'var(--color-text-secondary)' }}>
        Tipos aceptados: {acceptedLabels.join(', ')}
      </div>
      <input
        ref={inputRef}
        type="file"
        accept={acceptedTypes.join(',')}
        onChange={handleChange}
        style={{ display: 'none' }}
      />
    </div>
  )
}
