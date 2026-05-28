import { useEffect, useRef, useState } from 'react'
import { Link } from 'react-router-dom'
import {
  ingestCurricular,
  listCurriculumDocs,
  deleteCurriculumDoc,
} from '../../api/rag'
import type { CurriculumDocument } from '../../api/rag'
import { ErrorMessage } from '../../components/common/ErrorMessage'

const SEMESTRES = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]

export function CurriculumPage() {
  const [docs, setDocs] = useState<CurriculumDocument[]>([])
  const [loadingDocs, setLoadingDocs] = useState(true)
  const [fetchError, setFetchError] = useState<string | null>(null)

  // upload form state
  const fileRef = useRef<HTMLInputElement>(null)
  const [file, setFile] = useState<File | null>(null)
  const [carrera, setCarrera] = useState('')
  const [semestre, setSemestre] = useState<number>(1)
  const [codigoMateria, setCodigoMateria] = useState('')
  const [isRubric, setIsRubric] = useState(false)
  const [uploading, setUploading] = useState(false)
  const [uploadError, setUploadError] = useState<string | null>(null)
  const [uploadSuccess, setUploadSuccess] = useState<string | null>(null)

  // delete state
  const [deletingId, setDeletingId] = useState<string | null>(null)
  const [deleteError, setDeleteError] = useState<string | null>(null)

  useEffect(() => {
    loadDocs()
  }, [])

  async function loadDocs() {
    setLoadingDocs(true)
    setFetchError(null)
    try {
      const { documents } = await listCurriculumDocs()
      setDocs(documents)
    } catch (err: unknown) {
      setFetchError((err as Error).message)
    } finally {
      setLoadingDocs(false)
    }
  }

  async function handleUpload(e: React.FormEvent) {
    e.preventDefault()
    if (!file || !carrera.trim() || !codigoMateria.trim()) return
    setUploadError(null)
    setUploadSuccess(null)
    setUploading(true)
    try {
      const result = await ingestCurricular(file, carrera.trim(), semestre, codigoMateria.trim(), isRubric)
      setUploadSuccess(`Documento indexado: ${result.chunk_count} fragmentos (${result.status})`)
      setFile(null)
      setCarrera('')
      setCodigoMateria('')
      setIsRubric(false)
      setSemestre(1)
      if (fileRef.current) fileRef.current.value = ''
      await loadDocs()
    } catch (err: unknown) {
      setUploadError((err as Error).message)
    } finally {
      setUploading(false)
    }
  }

  async function handleDelete(id: string) {
    if (!window.confirm('¿Eliminar este documento del índice curricular?')) return
    setDeletingId(id)
    setDeleteError(null)
    try {
      await deleteCurriculumDoc(id)
      setDocs(prev => prev.filter(d => d.id !== id))
    } catch (err: unknown) {
      setDeleteError((err as Error).message)
    } finally {
      setDeletingId(null)
    }
  }

  return (
    <div className="p-8 max-w-4xl space-y-8">
      <div>
        <Link
          to="/admin/dashboard"
          className="text-sm text-[#888888] hover:text-[#E85D24] transition-colors"
        >
          ← Panel de administración
        </Link>
        <h1 className="text-2xl font-semibold text-[#f0f0f0] mt-4">Currículo — Gestión de Documentos</h1>
        <p className="text-sm text-[#888888] mt-1">
          Sube documentos curriculares (PDF, DOCX, TXT) para poblar la base de conocimiento de IA.
        </p>
      </div>

      {/* Upload form */}
      <form onSubmit={handleUpload} className="bg-[#1a1a1a] border border-[#2e2e2e] rounded-xl p-5 space-y-4">
        <h2 className="text-sm font-semibold text-[#E85D24] uppercase tracking-wide">Subir documento</h2>

        <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
          <div className="space-y-1">
            <label className="text-xs text-[#888888]">Carrera</label>
            <input
              type="text"
              value={carrera}
              onChange={e => setCarrera(e.target.value)}
              placeholder="Ej: Ingeniería de Sistemas"
              disabled={uploading}
              className="w-full bg-[#111111] border border-[#2e2e2e] rounded-lg px-3 py-2 text-sm text-[#f0f0f0] placeholder-[#888888] focus:outline-none focus:border-[#E85D24] disabled:opacity-50"
            />
          </div>

          <div className="space-y-1">
            <label className="text-xs text-[#888888]">Semestre</label>
            <select
              value={semestre}
              onChange={e => setSemestre(Number(e.target.value))}
              disabled={uploading}
              className="w-full bg-[#111111] border border-[#2e2e2e] rounded-lg px-3 py-2 text-sm text-[#f0f0f0] focus:outline-none focus:border-[#E85D24] disabled:opacity-50"
            >
              {SEMESTRES.map(s => (
                <option key={s} value={s}>{s}</option>
              ))}
            </select>
          </div>

          <div className="space-y-1">
            <label className="text-xs text-[#888888]">Código de Materia</label>
            <input
              type="text"
              value={codigoMateria}
              onChange={e => setCodigoMateria(e.target.value)}
              placeholder="Ej: INF-301"
              disabled={uploading}
              className="w-full bg-[#111111] border border-[#2e2e2e] rounded-lg px-3 py-2 text-sm text-[#f0f0f0] placeholder-[#888888] focus:outline-none focus:border-[#E85D24] disabled:opacity-50"
            />
          </div>
        </div>

        <div className="space-y-1">
          <label className="text-xs text-[#888888]">Archivo (PDF, DOCX, TXT)</label>
          <input
            ref={fileRef}
            type="file"
            accept=".pdf,.docx,.txt"
            disabled={uploading}
            onChange={e => setFile(e.target.files?.[0] ?? null)}
            className="w-full text-sm text-[#f0f0f0] file:mr-3 file:py-1.5 file:px-3 file:rounded file:border-0 file:bg-[#2e2e2e] file:text-[#f0f0f0] file:text-sm file:cursor-pointer hover:file:bg-[#3a3a3a] disabled:opacity-50"
          />
        </div>

        <label className="flex items-center gap-2 cursor-pointer select-none">
          <input
            type="checkbox"
            checked={isRubric}
            onChange={e => setIsRubric(e.target.checked)}
            disabled={uploading}
            className="accent-[#E85D24]"
          />
          <span className="text-sm text-[#888888]">Es rúbrica de evaluación</span>
        </label>

        <button
          type="submit"
          disabled={uploading || !file || !carrera.trim() || !codigoMateria.trim()}
          className="px-5 py-2 bg-[#E85D24] text-white text-sm font-medium rounded-lg hover:bg-[#d4521f] disabled:opacity-50 transition-colors flex items-center gap-2"
        >
          {uploading && (
            <span className="h-4 w-4 rounded-full border-2 border-white border-t-transparent animate-spin" />
          )}
          {uploading ? 'Indexando...' : 'Subir e indexar'}
        </button>

        {uploadError && <ErrorMessage message={uploadError} onDismiss={() => setUploadError(null)} />}
        {uploadSuccess && (
          <p className="text-sm text-green-400">{uploadSuccess}</p>
        )}
      </form>

      {/* Document list */}
      <div className="space-y-3">
        <div className="flex items-center justify-between">
          <h2 className="text-base font-semibold text-[#f0f0f0]">
            Documentos indexados {docs.length > 0 && `(${docs.length})`}
          </h2>
          <button
            onClick={loadDocs}
            disabled={loadingDocs}
            className="text-xs text-[#888888] hover:text-[#E85D24] transition-colors disabled:opacity-50"
          >
            {loadingDocs ? 'Cargando...' : 'Actualizar'}
          </button>
        </div>

        {fetchError && <ErrorMessage message={fetchError} onDismiss={() => setFetchError(null)} />}
        {deleteError && <ErrorMessage message={deleteError} onDismiss={() => setDeleteError(null)} />}

        {loadingDocs ? (
          <div className="flex justify-center py-8">
            <span className="h-6 w-6 rounded-full border-2 border-[#E85D24] border-t-transparent animate-spin" />
          </div>
        ) : docs.length === 0 ? (
          <p className="text-sm text-[#888888]">
            No hay documentos indexados. Sube el primer documento para activar el asistente curricular.
          </p>
        ) : (
          <div className="space-y-2">
            {docs.map(doc => (
              <DocRow
                key={doc.id}
                doc={doc}
                deleting={deletingId === doc.id}
                onDelete={() => handleDelete(doc.id)}
              />
            ))}
          </div>
        )}
      </div>
    </div>
  )
}

function DocRow({
  doc,
  deleting,
  onDelete,
}: {
  doc: CurriculumDocument
  deleting: boolean
  onDelete: () => void
}) {
  const date = new Date(doc.created_at).toLocaleDateString('es-BO', {
    day: '2-digit',
    month: 'short',
    year: 'numeric',
  })

  return (
    <div className="bg-[#1a1a1a] border border-[#2e2e2e] rounded-lg px-4 py-3 flex items-center justify-between gap-4">
      <div className="min-w-0 flex-1 space-y-0.5">
        <div className="flex items-center gap-2 flex-wrap">
          <span className="text-sm font-medium text-[#f0f0f0] truncate">{doc.original_filename}</span>
          {doc.is_rubric && (
            <span className="text-xs bg-[#2e2e2e] text-[#888888] px-1.5 py-0.5 rounded">Rúbrica</span>
          )}
        </div>
        <p className="text-xs text-[#888888]">
          {doc.codigo_materia} · {doc.carrera} · Sem {doc.semestre} · {date}
        </p>
      </div>
      <button
        onClick={onDelete}
        disabled={deleting}
        className="shrink-0 text-xs text-[#888888] hover:text-red-400 transition-colors disabled:opacity-50 flex items-center gap-1"
      >
        {deleting ? (
          <span className="h-3.5 w-3.5 rounded-full border-2 border-current border-t-transparent animate-spin" />
        ) : (
          'Eliminar'
        )}
      </button>
    </div>
  )
}
