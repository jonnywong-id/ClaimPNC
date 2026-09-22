import { useRef, useState } from 'react'

import { APIError, NetworkError } from '@/api/client'
import { ErrorCode, type AutoClaimUploadResponse } from '@/api/types'
import { Button } from '@/components/Button'
import { ErrorMessage } from '@/components/ErrorMessage'

import { useAutoClaimUploadTemplate, useUploadAutoClaim } from './api'

type Props = {
  /** Tab tujuan unggahan; tiap tab menulis ke TABEL yang berbeda. */
  source: string

  onClose: () => void
  onUploaded: (result: AutoClaimUploadResponse) => void
}

/**
 * Form "Upload Data Klaim".
 *
 * # Apa yang menggantikan apa
 *
 * Di sistem lama tombol ini membuka LOCAL ACTION pada
 * `Section/Inbox_AS_KREDIT_Sect-Section.xml`, yaitu flow action `PNCUploadClaimCSV`.
 *
 * # Daftar kolom yang PERTAMA saya karang, dan yang ternyata benar
 *
 * Versi pertama layar ini mengarang judul kolomnya dari nama kolom
 * TMP_BATCH_AUTO_CLAIM, karena flow action-nya belum ada di export. Ia tiba 2026-09-19
 * bersama `Activity/InsertKlaimToTable_Other-Act.xml`, dan membuktikan karangan itu
 * meminta TIGA kolom yang tidak pernah ada di berkas:
 *
 *   inisialid  kode perusahaan  -> diturunkan dari polis
 *   prodke     nomor produk     -> dicari dari polis
 *   currency   mata uang        -> diambil dari snapshot polis
 *
 * Ketiganya hasil pencarian, bukan isian. Meminta pengunggah mengisinya berarti meminta
 * nilai yang tidak ia ketahui — dan yang lebih buruk, membiarkannya salah tanpa satu pun
 * pemeriksaan yang dapat menangkapnya.
 *
 * Daftar kolomnya datang dari server, bukan diketik di sini, dan justru itulah yang
 * membuat koreksinya hanya menyentuh satu tempat.
 */
export function UploadForm({ source, onClose, onUploaded }: Props) {
  const template = useAutoClaimUploadTemplate()
  const upload = useUploadAutoClaim(source)

  const [file, setFile] = useState<File | null>(null)
  const inputRef = useRef<HTMLInputElement>(null)

  // Pelanggaran per baris datang sebagai peta `baris N · kolom` → pesan. Ia ditampilkan
  // sebagai daftar, bukan satu paragraf: pengguna memperbaiki berkasnya baris per baris.
  const violation = upload.error instanceof APIError ? upload.error.violations() : {}
  const violationList = Object.entries(violation)

  function submit(event: React.FormEvent) {
    event.preventDefault()
    if (!file) return

    upload.mutate(file, {
      onSuccess: (result) => {
        onUploaded(result)
        setFile(null)
        if (inputRef.current) inputRef.current.value = ''
      },
    })
  }

  return (
    <form
      onSubmit={submit}
      className="rounded-kartu border border-slate-200 bg-white p-5 shadow-lembut"
    >
      <h2 className="text-base font-semibold text-slate-900">Upload Data Klaim</h2>
      <p className="mt-1 text-sm text-slate-600">
        Berkas CSV berisi klaim borongan dari perusahaan rekanan. Nomor batch diterbitkan sistem
        setelah berkas diterima.
      </p>

      <div className="mt-4">
        <label htmlFor="berkas-unggahan" className="block text-sm font-medium text-slate-700">
          Berkas CSV
        </label>
        <input
          id="berkas-unggahan"
          ref={inputRef}
          type="file"
          accept=".csv,text/csv"
          onChange={(e) => setFile(e.target.files?.[0] ?? null)}
          className={[
            'mt-1 w-full rounded-kontrol border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900',
            'transition-[border-color,box-shadow] duration-150 ease-halus',
            'file:mr-3 file:rounded-kontrol file:border-0 file:bg-slate-100 file:px-3 file:py-1.5',
            'file:text-sm file:font-medium file:text-slate-700 hover:file:bg-slate-200',
            'focus:border-blue-500 focus:outline-none focus-visible:ring-4 focus-visible:ring-blue-500/20',
          ].join(' ')}
        />
      </div>

      {template.data && (
        <div className="mt-4 rounded-kartu border border-slate-200 bg-slate-50/70 p-4 text-sm">
          <p className="font-medium text-slate-800">Judul kolom yang harus ada di baris pertama</p>
          <p className="mt-2 break-words font-mono text-xs text-slate-700">
            {template.data.kolom_wajib.join(', ')}
          </p>
          <p className="mt-3 text-xs text-slate-600">
            Boleh ditambahkan:{' '}
            <span className="font-mono">{template.data.kolom_opsional.join(', ')}</span>
          </p>
          <ul className="mt-3 list-disc space-y-1 pl-5 text-xs text-slate-600">
            <li>
              Tanggal berformat <span className="font-mono">dd/mm/yyyy</span>, misalnya{' '}
              <span className="font-mono">17/09/2026</span>.
            </li>
            <li>
              Nilai klaim tanpa pemisah ribuan; pakai titik untuk desimal —{' '}
              <span className="font-mono">12500000.00</span>.
            </li>
            <li>Pemisah kolom boleh koma atau titik koma.</li>
            <li>
              Paling banyak {template.data.batas_baris.toLocaleString('id-ID')} baris sekali unggah.
            </li>
          </ul>

          {/* Keterangan ini ada karena ketiadaan kolomnya justru yang paling
              membingungkan: petugas yang terbiasa mengisi kode perusahaan akan mencari
              kolomnya dan mengira daftarnya kurang. */}
          <p className="mt-3 border-t border-slate-200 pt-3 text-xs text-slate-600">
            Kode perusahaan, nomor produk, dan mata uang{' '}
            <span className="font-medium">tidak perlu diisi</span> — ketiganya dicari sistem dari
            nomor polisnya. Satu berkas boleh memuat polis dari beberapa perusahaan sekaligus;
            masing-masing mendapat nomor batch-nya sendiri.
          </p>
          <p className="mt-2 text-xs text-slate-600">
            Baris yang polisnya tidak lolos pemeriksaan{' '}
            <span className="font-medium">tetap tersimpan</span> beserta keterangan gagalnya, supaya
            terlihat di grid dan ikut keluar di Export Gagal.
          </p>
        </div>
      )}

      {upload.isError && (
        <div className="mt-4 space-y-3">
          <ErrorMessage
            title={errorTitle(upload.error)}
            description={errorDescription(upload.error)}
            tone={upload.error instanceof NetworkError ? 'gangguan' : 'penolakan'}
          />
          {violationList.length > 0 && (
            <div className="rounded-kartu border border-red-200 bg-red-50/70 p-4">
              <p className="text-sm font-medium text-red-900">
                {violationList.length} baris perlu diperbaiki
              </p>
              {/* Daftarnya dapat digulir: berkas berisi ratusan baris cacat tidak boleh
                  mendorong tombol Simpan keluar dari layar. */}
              <ul className="mt-2 max-h-56 space-y-1 overflow-y-auto pr-1 text-sm text-red-800">
                {violationList.map(([kolom, pesan]) => (
                  <li key={kolom}>
                    <span className="font-mono text-xs">{kolom}</span> — {pesan}
                  </li>
                ))}
              </ul>
            </div>
          )}
        </div>
      )}

      <div className="mt-5 flex flex-wrap items-center gap-2">
        <Button type="submit" tone="utama" disabled={!file || upload.isPending}>
          {upload.isPending ? 'Mengunggah…' : 'Unggah'}
        </Button>
        <Button tone="halus" onClick={onClose} disabled={upload.isPending}>
          Batal
        </Button>
        {file && <span className="text-xs text-slate-500">{file.name}</span>}
      </div>
    </form>
  )
}

function errorTitle(error: unknown): string {
  if (error instanceof NetworkError) return 'Server Claim PNC tidak dapat dihubungi'
  if (error instanceof APIError) {
    switch (error.kode) {
      case ErrorCode.validationFailed:
        return 'Berkas belum dapat diterima'
      case ErrorCode.emptyUpload:
        return 'Berkas tidak memuat baris data'
      case ErrorCode.malformedRequest:
        return 'Berkas tidak dapat dibaca'
      case ErrorCode.portalNotStated:
      case ErrorCode.portalUnknown:
        return 'Portal entitas belum dipilih'
      case ErrorCode.portalNotReady:
        return 'Basis data entitas ini belum tersedia'
      default:
        return 'Unggahan gagal'
    }
  }
  return 'Unggahan gagal'
}

function errorDescription(error: unknown): string {
  if (error instanceof NetworkError) {
    return 'Periksa koneksi jaringan Anda, lalu coba unggah lagi.'
  }
  if (error instanceof APIError) {
    // Pesan dari server dipakai apa adanya: ia sudah menjelaskan apa yang salah, dan
    // menggantinya di sini berarti dua tempat memegang satu penjelasan.
    return error.message
  }
  return 'Coba beberapa saat lagi. Bila berulang, hubungi administrator Claim PNC.'
}
