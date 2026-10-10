import { useId, useRef, useState } from 'react'

import { APIError, NetworkError } from '@/api/client'
import type { PanelDocument } from '@/api/types'
import { Button } from '@/components/Button'
import { ErrorMessage, type ErrorTone } from '@/components/ErrorMessage'

/**
 * Isian "Upload Document" pada layar Master Panel.
 *
 * # Berkasnya pergi ke GCS, dan rantainya sudah ada
 *
 * Modul ini TIDAK membangun jalur penyimpanan sendiri. Adapter
 * `masterpanel/repo/dokumenlink` menyambung ke modul `dokumenpenunjang`, yang mengirim
 * berkasnya ke layanan penyimpanan internal — padanan Connect REST `UploadDokumenPNC`,
 * `POST /api/v1/upload` — lalu mencatat URL beserta masa berlakunya di
 * `GENERAL.T_STORAGE_IMAGE`.
 *
 * Yang tersimpan di `POOLDATA.DATA_ATTACHFILE` hanyalah METADATA beserta `IMAGEID`, dan
 * `IMAGEID` itulah satu-satunya tali ke berkas yang sebenarnya (`D-16`).
 *
 * # Kenapa bukan DokumenPenunjangPanel yang dipakai di sini
 *
 * Komponen itu ada dan memang panel unggah bersama — tetapi ia berkunci **nomor klaim**
 * (`/api/klaim/{nomor}/dokumen-penunjang`) dan menampilkan BANYAK dokumen.
 *
 * Master Panel bukan klaim, dan `PANEL_HE.DOKUMENID` hanya SATU kolom: satu panel memegang
 * satu dokumen. Memakainya di sini akan menaruh seluruh dokumen master di bawah satu nomor
 * klaim semu (`"-"`), dan tidak satu pun dapat ditelusuri milik panel yang mana.
 *
 * Yang dipakai bersama adalah **jalur penyimpanannya**, bukan komponennya.
 *
 * # Letaknya dibaca dari offset di section, bukan ditebak
 *
 * `Section/BrowsePanelHEApprove-Section.xml` menaruh ketiga tombolnya SEBELUM elemen Repeat
 * yang menggambar grid:
 *
 *	Upload Document  25.020
 *	Simpan           44.154
 *	Ubah             57.631
 *	Repeat (grid)    62.666
 *
 * Ketiganya tombol area form, bukan aksi baris maupun toolbar daftar.
 *
 * Ia hanya ada di tab Approve dan Reject: `Section/BrowsePanelHEApproval` — tab Waiting
 * Approval — TIDAK merujuk local action `UploadDocument` sama sekali.
 *
 * # Kenapa berkasnya DITAHAN, bukan langsung dikirim
 *
 * Pega menahannya juga. `Activity/SaveFilePenunjang` hanya menaruh berkas di halaman
 * sementara; yang menyimpannya adalah `CNMUpdatePanelHE_act` saat panel disimpan. Menirunya
 * berarti satu perilaku yang sama untuk Tambah dan Ubah — dan pada Tambah ia memang
 * satu-satunya yang mungkin, karena ID panel diterbitkan server dan baru lahir setelah
 * tersimpan.
 *
 * Akibat yang disadari: bila panel tersimpan tetapi unggahannya gagal, barisnya ada tanpa
 * dokumen. Itu persis perilaku sistem lama, dan pesannya menyebutkannya terang-terangan
 * alih-alih melaporkan "gagal" tanpa menjelaskan apa yang sudah terjadi.
 */
type Props = {
  /** Dokumen yang sudah menempel pada panel ini, bila ada. */
  current: PanelDocument | null

  /** Jalur unggah siap dipakai. Dibaca dari `unggah_tersedia` pada endpoint pilihan. */
  available: boolean

  /** Berkas yang sedang ditahan, beserta catatannya. */
  file: File | null
  note: string

  onPick: (file: File | null) => void
  onNote: (note: string) => void

  isSaving: boolean

  /** Galat unggahan, dipisahkan dari galat penyimpanan panel. */
  error: unknown

  /** Menutup panelnya. Dipakai tombol Submit dan Batal, sama seperti modal Pega. */
  onClose: () => void
}

export function DocumentField(props: Props) {
  const { current, available, file, note, onPick, onNote, isSaving, error, onClose } = props
  const fileId = useId()
  const noteId = useId()
  const input = useRef<HTMLInputElement>(null)
  const [tooBig, setTooBig] = useState(false)

  // Batas yang sama dengan `dokumenpenunjang.BatasUkuranBerkas`. Diperiksa di peramban
  // JUGA, bukan hanya di server: berkas 2 GB yang ditolak server tetap harus diunggah
  // seluruhnya lebih dulu, dan pada koneksi kantor itu menit-menit yang terbuang.
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
          description="Layanan penyimpanan dokumen belum terpasang pada lingkungan ini. Panel tetap dapat disimpan; dokumennya menyusul setelah layanan itu tersedia."
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
        {/*
          Labelnya "Berkas dokumen", BUKAN "Upload Document" — kata itu sudah menjadi judul
          panelnya, dan mengulanginya tepat di bawahnya membuat layar terbaca seolah ada dua
          hal berbeda.
        */}
        <label htmlFor={fileId} className="block text-sm font-medium text-slate-700">
          Berkas dokumen
        </label>
        {/*
          Widget berkas bawaan DISEMBUNYIKAN, bukan digaya ulang.

          Pada `<input type="file">` bawaan, SELURUH kontrolnya satu sasaran klik — termasuk
          teks "No file chosen" di sebelah tombolnya. Pengguna yang menekan teks itu pun
          membuka manajer berkas, dan itu mengejutkan karena teks itu terbaca sebagai
          keterangan, bukan tombol.

          `sr-only`, bukan `hidden`: inputnya tetap terbaca pembaca layar dan tetap tertaut
          ke labelnya.
        */}
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

      {/*
        Daftar "Nama File" — modal Pega menampilkannya sebagai grid atas
        `dragDropFileUpload.pxResults`, dan baris kosongnya berbunyi "Data Tidak Ada".
        Ditiru apa adanya supaya petugas yang berpindah dari layar lama mengenalinya.

        SATU baris, bukan daftar panjang: Pega memang mengizinkan beberapa berkas, tetapi
        `PANEL_HE.DOKUMENID` hanya SATU kolom — berkas kedua dan seterusnya akan tersimpan
        di DATA_ATTACHFILE tanpa ada yang menunjuknya.
      */}
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
        <>
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
          <div className="flex items-center gap-2">
            <Button tone="halus" onClick={clear} disabled={isSaving}>
              Batalkan berkas
            </Button>
          </div>
        </>
      )}

      {/*
        Submit dan Batal — modal Pega punya keduanya (`pyCancelLabel = Cancel`).

        Submit TIDAK menyimpan apa pun ke basis data, dan itu bukan kelalaian:
        `Activity/SaveFilePenunjang` hanya menitipkan berkas ke halaman sementara
        `dragDropFileUpload`, dan yang menyimpannya adalah jalur SIMPAN. Jadi Submit di sini
        berarti "berkas ini saya pilih" — persis seperti di Pega.
      */}
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

      {error !== null && error !== undefined && <UploadError error={error} />}
    </div>
  )
}

/**
 * Pesan galat unggahan.
 *
 * Dipisahkan dari pesan galat penyimpanan panel karena yang dibutuhkan pengguna berbeda: di
 * sini pertanyaan pertamanya adalah "apakah panel saya tersimpan" dan "bolehkah saya
 * mengulang" — dan jawaban keduanya berbeda per golongan galat.
 */
function UploadError({ error }: { error: unknown }) {
  let title = 'Dokumen gagal diunggah'
  let description =
    'Panel sudah tersimpan. Dokumennya belum terkirim — buka kembali panel ini lalu unggah ulang.'
  let tone: ErrorTone = 'gangguan'

  if (error instanceof NetworkError) {
    description =
      'Panel sudah tersimpan, tetapi dokumennya tidak terkirim karena sambungan terputus. Buka kembali panel ini lalu unggah ulang.'
  } else if (error instanceof APIError) {
    switch (error.kode) {
      case 'unggah_separuh_jalan':
        // Satu-satunya golongan yang pengulangannya BERBAHAYA — mengulang menumpuk berkas
        // ganda di layanan penyimpanan. Nadanya dinaikkan supaya terbaca berbeda.
        title = 'Dokumen terkirim tetapi belum tercatat'
        description =
          'Berkasnya sudah sampai di penyimpanan, tetapi catatannya gagal disimpan sehingga belum tertaut ke panel. JANGAN unggah ulang — laporkan ke administrator.'
        tone = 'penolakan'
        break
      case 'berkas_terlalu_besar':
        title = 'Berkas terlalu besar'
        description = `Panel sudah tersimpan. ${error.message}`
        tone = 'penolakan'
        break
      case 'unggah_tidak_sah':
        title = 'Berkas tidak dapat diterima'
        description = `Panel sudah tersimpan. ${error.message}`
        tone = 'penolakan'
        break
      case 'layanan_unggah_tidak_tersedia':
        title = 'Layanan penyimpanan dokumen sedang tidak tersedia'
        description = `Panel sudah tersimpan. ${error.message} Buka kembali panel ini lalu unggah ulang.`
        break
      default:
        description = `Panel sudah tersimpan. ${error.message}`
    }
  }

  return <ErrorMessage title={title} description={description} tone={tone} />
}
