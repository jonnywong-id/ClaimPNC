import { useId, useRef, useState } from 'react'

import { APIError, NetworkError } from '@/api/client'
import type { SparepartDocument } from '@/api/types'
import { Button } from '@/components/Button'
import { ErrorMessage, type ErrorTone } from '@/components/ErrorMessage'

/**
 * Panel "Upload Document" Master Sparepart.
 *
 * # Ke mana berkasnya pergi
 *
 * Ke jalur penyimpanan dokumen yang SUDAH ADA di aplikasi ini — modul `dokumenpenunjang`,
 * yang meniru rantai `InsertDokumenPNC` Pega sampai ujungnya:
 *
 *	UploadDocumentToGoogleStorage  menyusun nama berkas, lalu memanggil
 *	 -> InsertDokumenPNC           isi berkas dikirim ke layanan penyimpanan internal
 *	 -> Connect REST UploadDokumenPNC   POST /api/v1/upload
 *	 -> InsertDataPNCStorage       URL dan masa berlakunya ke GENERAL.T_STORAGE_IMAGE
 *
 * Termasuk konvensi namanya, yang sudah terpasang di `dokumenpenunjang.NamaUnggah`:
 * `yyMdhm-sS` + `-<jenis dokumen>-` + `<nama berkas>`. Awalan waktu itu yang membuat nama
 * unik — trigger `GENERAL.TBIU_STORAGE_IMAGE` menolak nama yang sudah ada LINTAS aplikasi,
 * bukan hanya lintas klaim.
 *
 * Modul ini TIDAK membangun rantainya ulang; ia menyambung lewat seam `DocumentUploader`.
 *
 * # Satu sparepart, satu dokumen
 *
 * `RDB List/GetIDDokumenSparepart-SQL.xml` membacanya dari satu kolom:
 *
 *	SELECT DOKUMENID AS "CoverID" FROM POOLDATA.SPAREPART_HE WHERE ID = ?
 *
 * Mengunggah berkas kedua MENGGANTI yang pertama. Dinyatakan di layar sebelum pengguna
 * memilih berkas, bukan dilaporkan sesudahnya.
 *
 * # Bentuknya mengikuti modal Pega
 *
 * `Flow Action/UploadDocument-FA.xml` berpasangan **Cancel/Submit** (`pyCancelLabel`), dan
 * `Section/UploadDocument-Section.xml` menampilkan daftar **"Nama File"** atas
 * `dragDropFileUpload.pxResults` — baris kosongnya "Data Tidak Ada".
 *
 * **Submit TIDAK mengirim apa pun.** `Activity/SaveFilePenunjang` hanya menitipkan berkas ke
 * halaman sementara; yang menyimpannya adalah jalur SIMPAN —
 * `Activity/UpdateSparepartHE_act` memutar halaman itu
 * (`pyStepsObjectName = dragDropFileUpload.pxResults`). Di sini pun begitu: berkas ditahan
 * sampai sparepart disimpan.
 *
 * # Widget berkas bawaan disembunyikan
 *
 * Pada `<input type="file">` bawaan, SELURUH kontrolnya satu sasaran klik — termasuk teks
 * "No file chosen", yang terbaca sebagai keterangan dan bukan tombol. `sr-only` menyisakan
 * tombol sendiri sebagai satu-satunya pemicu, tanpa melepas tautan label untuk pembaca layar.
 */
type Props = {
  /** Dokumen yang sudah menempel pada sparepart ini, bila ada. */
  current: SparepartDocument | null

  /** Jalur unggah siap dipakai. Dibaca dari `unggah_tersedia` pada endpoint pilihan. */
  available: boolean

  /** Berkas yang sedang ditahan, beserta catatannya. */
  file: File | null
  note: string

  onPick: (file: File | null) => void
  onNote: (note: string) => void

  isSaving: boolean

  /** Galat unggahan, dipisahkan dari galat penyimpanan sparepart. */
  error: unknown

  /** Menutup panelnya. Dipakai Submit dan Batal, sama seperti modal Pega. */
  onClose: () => void
}

export function DocumentField(props: Props) {
  const { current, available, file, note, onPick, onNote, isSaving, error, onClose } = props
  const fileId = useId()
  const noteId = useId()
  const input = useRef<HTMLInputElement>(null)
  const [tooBig, setTooBig] = useState(false)

  // Batas yang sama dengan `dokumenpenunjang.BatasUkuranBerkas`. Diperiksa di peramban JUGA,
  // bukan hanya di server: berkas 2 GB yang ditolak server tetap harus diunggah seluruhnya
  // lebih dulu, dan pada koneksi kantor itu menit-menit yang terbuang.
  const maxBytes = 20 * 1024 * 1024

  function pick(chosen: File | null) {
    if (chosen !== null && chosen.size > maxBytes) {
      setTooBig(true)
      onPick(null)
      if (input.current) input.current.value = ''
      return
    }
    setTooBig(false)
    onPick(chosen)
  }

  function clear() {
    setTooBig(false)
    onPick(null)
    onNote('')
    if (input.current) input.current.value = ''
  }

  return (
    <div className="space-y-3">
      {!available && (
        <ErrorMessage
          title="Unggah dokumen belum tersedia"
          description="Layanan penyimpanan dokumen belum terpasang pada lingkungan ini. Sparepart tetap dapat disimpan; dokumennya menyusul setelah layanan itu tersedia."
          tone="gangguan"
        />
      )}

      {current !== null && (
        <p className="text-sm text-slate-600">
          Dokumen saat ini:{' '}
          <span className="font-medium text-slate-800">{current.nama_berkas}</span>
          {current.diunggah_oleh !== '' && (
            <span className="text-slate-500"> — diunggah {current.diunggah_oleh}</span>
          )}
        </p>
      )}

      <div>
        <label htmlFor={fileId} className="block text-sm font-medium text-slate-700">
          Berkas dokumen
        </label>
        <input
          id={fileId}
          ref={input}
          type="file"
          disabled={!available || isSaving}
          onChange={(event) => pick(event.target.files?.[0] ?? null)}
          className="sr-only"
        />
        <div className="mt-1 flex flex-wrap items-center gap-3">
          <Button
            tone="kedua"
            onClick={() => {
              input.current?.click()
            }}
            disabled={!available || isSaving}
          >
            Pilih Berkas
          </Button>
        </div>
        <p className="mt-1.5 text-xs text-slate-500">Maksimal 20 MB.</p>
      </div>

      {/* Daftar "Nama File" — meniru grid modal Pega beserta baris kosongnya. */}
      <div>
        <p className="text-sm font-medium text-slate-700">Nama File</p>
        <div className="mt-1 rounded-kontrol border border-slate-200">
          <p className="px-3 py-2 text-sm text-slate-600">
            {file === null ? 'Data tidak ada' : file.name}
          </p>
        </div>
      </div>

      {tooBig && (
        <ErrorMessage
          title="Berkas terlalu besar"
          description="Ukuran berkas melebihi batas 20 MB. Perkecil berkasnya lalu pilih ulang."
          tone="penolakan"
        />
      )}

      {file !== null && (
        <div>
          <label htmlFor={noteId} className="block text-sm font-medium text-slate-700">
            Catatan dokumen
          </label>
          <input
            id={noteId}
            type="text"
            value={note}
            maxLength={250}
            disabled={isSaving}
            onChange={(event) => onNote(event.target.value)}
            className="mt-1 block w-full rounded border border-slate-300 px-2 py-1.5 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500 disabled:bg-slate-50"
          />
        </div>
      )}

      {error !== null && error !== undefined && <UploadError error={error} />}

      <div className="flex flex-wrap justify-end gap-2 border-t border-slate-200 pt-3">
        <Button
          tone="halus"
          onClick={() => {
            clear()
            onClose()
          }}
          disabled={isSaving}
        >
          Batal
        </Button>
        <Button tone="utama" onClick={onClose} disabled={file === null || isSaving}>
          Submit
        </Button>
      </div>
    </div>
  )
}

/**
 * Pesan galat unggahan.
 *
 * Dipisahkan dari galat penyimpanan sparepart karena yang dibutuhkan pengguna berbeda: di
 * sini pertanyaan pertamanya adalah "apakah sparepart saya tersimpan" dan "bolehkah saya
 * mengulang" — dan jawaban keduanya berbeda per golongan galat.
 */
function UploadError({ error }: { error: unknown }) {
  let title = 'Dokumen gagal diunggah'
  let description =
    'Sparepart sudah tersimpan. Dokumennya belum terkirim — buka kembali sparepart ini lalu unggah ulang.'
  let tone: ErrorTone = 'gangguan'

  if (error instanceof NetworkError) {
    description =
      'Sparepart sudah tersimpan, tetapi dokumennya tidak terkirim karena sambungan terputus. Buka kembali sparepart ini lalu unggah ulang.'
  } else if (error instanceof APIError) {
    switch (error.kode) {
      case 'unggah_separuh_jalan':
        // Satu-satunya golongan yang pengulangannya BERBAHAYA — mengulang menumpuk berkas
        // ganda di layanan penyimpanan. Nadanya dinaikkan supaya terbaca berbeda.
        title = 'Dokumen terkirim tetapi belum tercatat'
        description =
          'Berkasnya sudah sampai di penyimpanan, tetapi catatannya gagal disimpan sehingga belum tertaut ke sparepart. JANGAN unggah ulang — laporkan ke administrator.'
        tone = 'penolakan'
        break
      case 'berkas_terlalu_besar':
        title = 'Berkas terlalu besar'
        description = `Sparepart sudah tersimpan. ${error.message}`
        tone = 'penolakan'
        break
      case 'unggah_tidak_sah':
        title = 'Berkas tidak dapat diterima'
        description = `Sparepart sudah tersimpan. ${error.message}`
        tone = 'penolakan'
        break
      case 'layanan_unggah_tidak_tersedia':
        title = 'Layanan penyimpanan dokumen sedang tidak tersedia'
        description = `Sparepart sudah tersimpan. ${error.message} Buka kembali sparepart ini lalu unggah ulang.`
        break
      default:
        description = `Sparepart sudah tersimpan. ${error.message}`
    }
  }

  return <ErrorMessage title={title} description={description} tone={tone} />
}
