import { useId, useRef, useState } from 'react'

import { APIError, NetworkError } from '@/api/client'
import type { SparepartImportReport } from '@/api/types'
import { Button } from '@/components/Button'
import { ErrorMessage, type ErrorTone } from '@/components/ErrorMessage'

/**
 * Panel "Upload Data Master Sparepart" — unggah CSV berisi banyak sparepart sekaligus.
 *
 * Bentuknya dicerminkan dari `master-panel/ImportCSVPanel`, dan itu disengaja: dari sudut
 * pandang pengguna keduanya mengerjakan hal yang sama — pilih satu berkas, kirim, baca
 * laporannya. Petugas yang sama memakai kedua layar, dan menggambarnya dengan dua bentuk
 * berbeda hanya memaksanya membaca ulang layar yang sudah dikenalnya.
 *
 * Yang BERBEDA dari sana hanya dua: judul kolom penanda pada tabel kegagalan, dan jalur CSV-nya
 * di sini cuma satu (Panel punya dua — master dan lokasi), sehingga labelnya ditulis langsung
 * alih-alih diterima sebagai prop.
 *
 * # Berbeda dari Upload Document: berkasnya DIKIRIM SEKETIKA
 *
 * Unggah dokumen menahan berkas sampai Simpan ditekan, karena dokumennya menempel pada satu
 * baris yang ID-nya baru lahir setelah tersimpan. Unggah CSV tidak punya ketergantungan itu —
 * ia MEMBAWA kuncinya sendiri di dalam berkas (`NO_SPART`), dan setiap barisnya menentukan
 * sendiri apakah ia menambah atau memperbarui.
 *
 * Menahannya sampai Simpan justru akan membingungkan: tidak ada satu sparepart pun yang sedang
 * disunting saat berkas berisi tiga ratus baris diunggah.
 *
 * # Kenapa hasilnya ditampilkan per baris
 *
 * Karena satu baris yang gagal TIDAK membatalkan yang lain — itu perilaku Pega, yang memutar
 * baris satu per satu tanpa transaksi yang membungkus keseluruhan
 * (`Activity/PNCUploadMasterSparepart_Act`). Unggahan yang separuh berhasil karena itu
 * meninggalkan separuh perubahan, dan satu-satunya hal yang membuat itu dapat diterima adalah
 * pengguna tahu PERSIS baris mana yang masuk dan mana yang tidak.
 *
 * Nomor barisnya mengikuti nomor di berkas, header sebagai baris 1, supaya pengguna dapat
 * langsung membukanya di penyunting teksnya.
 */
type Props = {
  isSending: boolean
  report: SparepartImportReport | null
  error: unknown
  onSend: (file: File) => void

  /** Menutup panelnya. Dipakai tombol Batal, sama seperti panel Upload Document. */
  onClose: () => void
}

export function ImportCSVPanel({ isSending, report, error, onSend, onClose }: Props) {
  const fileId = useId()
  const input = useRef<HTMLInputElement>(null)
  const [file, setFile] = useState<File | null>(null)
  const [tooBig, setTooBig] = useState(false)

  // Batas yang sama dengan jalur unggah dokumen — satu angka untuk seluruh aplikasi.
  const maxBytes = 20 * 1024 * 1024

  function pick(chosen: File | null) {
    if (chosen !== null && chosen.size > maxBytes) {
      setTooBig(true)
      setFile(null)
      if (input.current) input.current.value = ''
      return
    }
    setTooBig(false)
    setFile(chosen)
  }

  function send() {
    if (file !== null) onSend(file)
  }

  // Batal mengosongkan pilihan LALU menutup — bentuk yang sama dengan panel dokumen.
  function cancel() {
    setTooBig(false)
    setFile(null)
    if (input.current) input.current.value = ''
    onClose()
  }

  return (
    <div className="space-y-3">
      <div>
        <label htmlFor={fileId} className="block text-sm font-medium text-slate-700">
          Berkas CSV
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
          accept=".csv,text/csv"
          disabled={isSending}
          onChange={(event) => pick(event.target.files?.[0] ?? null)}
          className="sr-only"
        />
        <div className="mt-1 flex flex-wrap items-center gap-3">
          <Button
            tone="kedua"
            onClick={() => {
              input.current?.click()
            }}
            disabled={isSending}
          >
            Pilih Berkas
          </Button>
        </div>
        <p className="mt-1.5 text-xs text-slate-500">Maksimal 20 MB.</p>
      </div>

      {/*
        Daftar "Nama File" — bentuknya SAMA PERSIS dengan panel Upload Document, termasuk
        baris kosongnya.
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
          description="Ukuran berkas melebihi batas 20 MB. Pecah berkasnya lalu unggah bergantian."
          tone="penolakan"
        />
      )}

      {/*
        Batal dan Submit — susunan, urutan, dan perataannya SAMA dengan panel Upload Document,
        mengikuti modal Pega yang juga berpasangan Cancel/Submit.

        Yang BERBEDA tetap ada, dan memang tidak dapat disamakan: Submit di sini MENGIRIM
        berkasnya seketika, sedangkan Submit pada panel dokumen hanya menutup panel karena
        berkasnya menunggu Simpan. Perbedaan itu berasal dari Pega, bukan dari pilihan di sini.
      */}
      <div className="flex flex-wrap justify-end gap-2 border-t border-slate-200 pt-3">
        <Button tone="halus" onClick={cancel} disabled={isSending}>
          Batal
        </Button>
        <Button tone="utama" onClick={send} disabled={file === null || isSending}>
          {isSending ? 'Mengunggah…' : 'Submit'}
        </Button>
      </div>

      {error !== null && error !== undefined && <ImportError error={error} />}
      {report !== null && <ImportReportView report={report} />}
    </div>
  )
}

/** Ringkasan hasil, beserta daftar baris yang gagal. */
function ImportReportView({ report }: { report: SparepartImportReport }) {
  const berhasil = report.baru + report.diperbarui
  const gagal = report.baris.filter((row) => row.hasil === 'gagal')

  return (
    <div className="space-y-3 rounded-kontrol border border-slate-200 bg-slate-50 p-3">
      <p className="text-sm text-slate-800">
        <span className="font-medium">{report.total}</span> baris dibaca —{' '}
        <span className="font-medium text-emerald-700">{report.baru}</span> ditambahkan,{' '}
        <span className="font-medium text-emerald-700">{report.diperbarui}</span> diperbarui,{' '}
        <span className={report.gagal > 0 ? 'font-medium text-rose-700' : 'font-medium'}>
          {report.gagal}
        </span>{' '}
        gagal.
      </p>

      {berhasil > 0 && (
        <p className="text-xs text-slate-600">
          {/* Dinyatakan terang-terangan: baris yang berhasil TETAP tersimpan meski ada yang
              gagal. Tanpa ini pengguna akan mengunggah ulang seluruh berkas dan menimpa
              baris yang sudah benar. */}
          {berhasil} baris yang berhasil <span className="font-medium">tetap tersimpan</span>.
          Perbaiki baris yang gagal saja, lalu unggah ulang bagian itu.
        </p>
      )}

      {gagal.length > 0 && (
        <div className="max-h-60 overflow-auto rounded border border-slate-200 bg-white">
          <table className="w-full text-left text-xs">
            <thead className="bg-slate-100 text-slate-700">
              <tr>
                <th className="px-2 py-1.5 font-medium">Baris</th>
                <th className="px-2 py-1.5 font-medium">Nomor Sparepart</th>
                <th className="px-2 py-1.5 font-medium">Sebab</th>
              </tr>
            </thead>
            <tbody>
              {gagal.map((row) => (
                <tr key={row.baris} className="border-t border-slate-100">
                  <td className="px-2 py-1.5 tabular-nums text-slate-900">{row.baris}</td>
                  <td className="px-2 py-1.5 text-slate-900">{row.nomor_sparepart || '—'}</td>
                  <td className="px-2 py-1.5 text-slate-700">{row.pesan}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}

/**
 * Pesan galat yang membatalkan SELURUH berkas.
 *
 * Dibedakan dari baris yang gagal: di sini tidak ada satu baris pun yang masuk, sehingga yang
 * perlu diketahui pengguna adalah berkasnya harus diperbaiki — bukan sebagiannya.
 */
function ImportError({ error }: { error: unknown }) {
  let title = 'Berkas gagal diunggah'
  let description = 'Tidak ada baris yang tersimpan. Periksa berkasnya lalu unggah ulang.'
  let tone: ErrorTone = 'gangguan'

  if (error instanceof NetworkError) {
    description =
      'Sambungan terputus sebelum berkas terkirim. Tidak ada baris yang tersimpan; silakan unggah ulang.'
  } else if (error instanceof APIError) {
    switch (error.kode) {
      case 'csv_tidak_sah':
        title = 'Berkas tidak dapat dibaca'
        description = error.message
        tone = 'penolakan'
        break
      case 'csv_terlalu_banyak_baris':
        title = 'Berkas terlalu banyak baris'
        description = error.message
        tone = 'penolakan'
        break
      case 'berkas_terlalu_besar':
        title = 'Berkas terlalu besar'
        description = error.message
        tone = 'penolakan'
        break
      default:
        description = error.message
    }
  }

  return <ErrorMessage title={title} description={description} tone={tone} />
}
