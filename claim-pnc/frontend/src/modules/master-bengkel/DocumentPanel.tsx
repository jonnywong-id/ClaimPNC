import { useRef, useState } from 'react'

import { APIError, simpanBerkas } from '@/api/client'
import type { Workshop } from '@/api/types'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'
import { Button } from '@/components/Button'
import { ErrorMessage } from '@/components/ErrorMessage'

import { unduhDokumenBengkel, useUploadWorkshopDocument, useWorkshopDocument } from './api'

/**
 * Panel dokumen lampiran satu bengkel.
 *
 * # Apa yang digantikannya di Pega
 *
 * DUA tombol layar lama sekaligus, keduanya `@baseclass` ruleset GCNMFW 01-01-89:
 *
 *	Section/BrowseMasterHE          "Upload Document"  -> Flow Action UploadDocument
 *	Section/ApprovalMasterBengkelHE  lihat dokumen     -> Flow Action ViewDocumentMasterBengkel
 *
 * Isi modal unggahnya di Pega hanya satu pemilih berkas berlabel "Nama File", di bawah
 * judul "Select a file to load and import" — tidak ada kategori, tidak ada catatan. Bentuk
 * itu diikuti apa adanya; yang ditambahkan hanya keterangan dokumen yang sedang terlampir,
 * karena tanpa itu pengguna tidak punya cara tahu bahwa unggahan berikutnya MENGGANTIKAN
 * yang sekarang.
 *
 * # Kenapa per BARIS, bukan tombol layar
 *
 * Di Pega "Upload Document" adalah tombol tingkat layar, padahal yang disimpannya adalah
 * `BENGKEL_HE.DOKUMENID` — kolom milik SATU baris. Pada layar daftar, tombol tingkat layar
 * tidak menyatakan baris mana yang dilampiri.
 *
 * Penempatannya karena itu mengikuti bentuk datanya, bukan letak tombolnya. Ini penyimpangan
 * letak yang sama sifatnya dengan bilah Approve/Reject, dan dicatat dengan alasan yang sama.
 *
 * # Satu lampiran per bengkel
 *
 * `BENGKEL_HE` hanya punya satu kolom `DOKUMENID`. Unggahan berikutnya menggantikan
 * tautannya; baris lampiran yang lama tidak dihapus (`D-66`), ia hanya tidak lagi tertaut.
 */
export function DocumentPanel({ workshop, onClose }: { workshop: Workshop; onClose: () => void }) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  const document = useWorkshopDocument(workshop.id_bengkel, true)
  const upload = useUploadWorkshopDocument()

  const [chosen, setChosen] = useState<File | null>(null)
  const [downloadError, setDownloadError] = useState<string | null>(null)
  const [isDownloading, setDownloading] = useState(false)
  const fileInput = useRef<HTMLInputElement>(null)

  const attached = document.data?.dokumen ?? null

  // 404 di sini berarti "belum ada dokumen", bukan kerusakan — dan itu keadaan yang wajar
  // bagi sebagian besar bengkel. Ia karena itu TIDAK digambar sebagai galat.
  const loadFailed =
    document.isError && !(document.error instanceof APIError && document.error.status === 404)

  async function download() {
    if (!attached) return
    setDownloadError(null)
    setDownloading(true)
    try {
      simpanBerkas(await unduhDokumenBengkel(workshop.id_bengkel, token, portal))
    } catch (error) {
      setDownloadError(
        error instanceof APIError
          ? error.message
          : 'Berkas tidak dapat diunduh. Coba beberapa saat lagi.',
      )
    } finally {
      setDownloading(false)
    }
  }

  function submit() {
    if (!chosen) return
    upload.mutate(
      { id: workshop.id_bengkel, berkas: chosen },
      {
        onSuccess: () => {
          setChosen(null)
          if (fileInput.current) fileInput.current.value = ''
        },
      },
    )
  }

  return (
    <section
      className="mt-5 rounded-lg border border-slate-200 bg-white p-5"
      aria-label={`Dokumen bengkel ${workshop.nama_bengkel}`}
    >
      <header className="flex items-start justify-between gap-4">
        <div>
          <h2 className="text-base font-semibold text-slate-900">Dokumen bengkel</h2>
          <p className="mt-0.5 text-sm text-slate-600">
            {workshop.id_bengkel} · {workshop.nama_bengkel}
          </p>
        </div>
        <Button tone="halus" onClick={onClose}>
          Tutup
        </Button>
      </header>

      <div className="mt-4">
        {document.isPending ? (
          <p className="text-sm text-slate-500">Memuat keterangan dokumen…</p>
        ) : loadFailed ? (
          <ErrorMessage
            title="Keterangan dokumen tidak dapat dimuat"
            description="Coba beberapa saat lagi. Bila berulang, hubungi administrator Claim PNC."
            tone="gangguan"
          />
        ) : attached ? (
          <AttachedDocument
            name={attached.nama_berkas}
            size={attached.ukuran_byte}
            hasContent={attached.berisi}
            uploadedBy={attached.diunggah_oleh}
            uploadedAt={attached.diunggah_pada}
            isDownloading={isDownloading}
            onDownload={() => { download() }}
          />
        ) : (
          <p className="text-sm text-slate-600">Bengkel ini belum punya dokumen terlampir.</p>
        )}
      </div>

      {downloadError && (
        <div className="mt-4">
          <ErrorMessage title="Unduhan gagal" description={downloadError} tone="gangguan" />
        </div>
      )}

      <div className="mt-5 border-t border-slate-200 pt-5">
        {/*
          Judul dan label mengikuti layar lama apa adanya: "Select a file to load and import"
          dan "Nama File" keduanya dibaca dari Section/UploadDocument.
        */}
        <h3 className="text-sm font-semibold text-slate-900">
          Select a file to load and import
        </h3>

        <label className="mt-3 block text-sm font-medium text-slate-700" htmlFor="berkas-bengkel">
          Nama File
        </label>
        <input
          ref={fileInput}
          id="berkas-bengkel"
          type="file"
          className="mt-1 block w-full text-sm text-slate-700 file:mr-3 file:rounded file:border-0 file:bg-slate-100 file:px-3 file:py-1.5 file:text-sm file:font-medium file:text-slate-700"
          onChange={(event) => {
            setChosen(event.target.files?.[0] ?? null)
            upload.reset()
          }}
        />

        {attached && (
          <p className="mt-2 text-sm text-amber-700">
            Bengkel ini sudah punya dokumen. Unggahan baru akan menggantikannya.
          </p>
        )}

        {upload.isError && (
          <div className="mt-3">
            <ErrorMessage
              title="Unggahan gagal"
              description={
                upload.error instanceof APIError
                  ? upload.error.message
                  : 'Berkas tidak dapat diunggah. Coba beberapa saat lagi.'
              }
              tone="gangguan"
            />
          </div>
        )}

        <div className="mt-4 flex items-center gap-3">
          <Button onClick={submit} disabled={chosen === null || upload.isPending}>
            {upload.isPending ? 'Mengunggah…' : 'Unggah'}
          </Button>
          {chosen && <span className="text-sm text-slate-600">{chosen.name}</span>}
        </div>
      </div>
    </section>
  )
}

/** AttachedDocument menggambar keterangan dokumen yang sedang tertaut. */
function AttachedDocument({
  name,
  size,
  hasContent,
  uploadedBy,
  uploadedAt,
  isDownloading,
  onDownload,
}: {
  name: string
  size: number
  hasContent: boolean
  uploadedBy: string
  uploadedAt: string
  isDownloading: boolean
  onDownload: () => void
}) {
  return (
    <div className="rounded border border-slate-200 bg-slate-50 p-4">
      <p className="text-sm font-medium text-slate-900">{name}</p>
      <p className="mt-1 text-sm text-slate-600">
        {hasContent ? formatSize(size) : 'tanpa isi'}
        {uploadedBy && ` · diunggah ${uploadedBy}`}
        {uploadedAt && ` · ${formatMoment(uploadedAt)}`}
      </p>

      {hasContent ? (
        <div className="mt-3">
          <Button tone="halus" onClick={onDownload} disabled={isDownloading}>
            {isDownloading ? 'Mengunduh…' : 'Unduh'}
          </Button>
        </div>
      ) : (
        /*
          Dokumen warisan: barisnya ada, isinya tidak pernah tersimpan karena jalur unggah
          sistem lama tidak menulis kolom isinya. Menawarkan tombol unduh di sini hanya
          menghasilkan berkas nol byte yang tampak seperti unduhan berhasil.
        */
        <p className="mt-3 text-sm text-amber-700">
          Isi berkas ini tidak tersimpan. Dokumen yang diunggah lewat sistem lama hanya
          mencatat keterangannya. Unggah ulang berkasnya bila masih dibutuhkan.
        </p>
      )}
    </div>
  )
}

/** formatSize menuliskan ukuran berkas dalam satuan yang terbaca orang. */
function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${Math.round(bytes / 1024)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

/**
 * formatMoment menuliskan waktu unggah dalam WIB.
 *
 * Waktunya datang sebagai RFC 3339 dari server; yang ditampilkan adalah waktu lokal
 * peramban, sama seperti seluruh layar lain modul ini.
 */
function formatMoment(value: string): string {
  const moment = new Date(value)
  if (Number.isNaN(moment.getTime())) return value
  return moment.toLocaleString('id-ID', { dateStyle: 'short', timeStyle: 'short' })
}
