import { useEffect, useState } from 'react'

import { APIError } from '@/api/client'
import { Button } from '@/components/Button'
import { ErrorMessage } from '@/components/ErrorMessage'

import { useUploadSalvageDetail } from './api'
import type { UploadResponse } from './types'

/**
 * Modal **"UploadDetailSalvage"** — di balik tombol "Upload Detail Salvage" pada form
 * salvage.
 *
 * # Artefak yang dibaca
 *
 *	Section/TambahData_Salvage-Section.xml   tombol pemanggil (local action)
 *	Flow Action/UploadDetailSalvage-FA.xml   pembungkusnya
 *	Activity/UploadDetailSalvage-Act.xml     yang membaca berkasnya
 *
 * Isi modal ini TIDAK punya section sendiri: flow action-nya memakai section bawaan Pega
 * `pxUploadCSVResults`, dan itulah yang menggambar kalimat "Select a local CSV file to
 * load and import" beserta pemilih berkas bawaan peramban dan sepasang tombol
 * Cancel/Submit. Ketiganya disalin apa adanya, termasuk bahasanya — ia teks bawaan Pega,
 * bukan kalimat yang pernah diterjemahkan.
 *
 * # Judulnya TANPA garis bawah
 *
 * `UploadDetailSalvage`, bukan `UploadDetail_Salvage`. Berbeda dari modal dokumen di
 * sebelahnya (`UploadDocument_Salvage`) yang memang bergaris bawah. Keduanya disalin dari
 * `pyRuleName` masing-masing, dan perbedaan itu memang ada di sistem lama.
 *
 * # Ia TIDAK menyimpan apa pun
 *
 * Itu yang paling mudah disalahpahami, dan `Activity/UploadDetailSalvage-Act.xml`
 * membuktikannya: ketiga langkahnya hanya menyalin isi berkas CSV ke grid DI DALAM form.
 * Penyimpanan baru terjadi saat Submit pada form ditekan — bukan Submit pada modal ini.
 *
 * Karena itu barisnya DITAMBAHKAN ke yang sudah ada, bukan menggantikannya: activity
 * aslinya memakai `<APPEND>` pada setiap baris, sehingga dua berkas yang diunggah
 * berturut-turut menghasilkan gabungan keduanya.
 */
export function UploadDetailSalvageModal({
  onClose,
  onLoaded,
}: {
  onClose: () => void

  /** Dipanggil setelah berkasnya terbaca, membawa baris yang dihasilkannya. */
  onLoaded: (result: UploadResponse) => void
}) {
  const [file, setFile] = useState<File | null>(null)
  const upload = useUploadSalvageDetail()

  // Escape menutup modal, seperti dialog mana pun yang dikenal pengguna — KECUALI saat
  // berkasnya sedang dibaca. Menutupnya di tengah jalan membuang hasil yang sebentar lagi
  // tiba, dan pengguna akan mengira unggahannya gagal.
  useEffect(() => {
    function onKey(event: KeyboardEvent) {
      if (event.key === 'Escape' && !upload.isPending) onClose()
    }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [onClose, upload.isPending])

  function submit() {
    if (file === null) return

    upload.mutate(file, {
      onSuccess: (result) => {
        onLoaded(result)
        onClose()
      },
    })
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-start justify-center bg-slate-900/40 px-4 py-8"
      role="dialog"
      aria-modal="true"
      aria-labelledby="judul-unggah-detail-salvage"
    >
      <div className="max-h-full w-full max-w-sm overflow-y-auto rounded-kartu bg-white shadow-angkat">
        {/* Kepala modal: judul apa adanya beserta tanda silangnya. */}
        <div className="flex items-center justify-between border-b border-slate-200 px-5 py-3">
          <h2
            id="judul-unggah-detail-salvage"
            className="text-base font-medium text-slate-900"
          >
            UploadDetailSalvage
          </h2>
          <button
            type="button"
            onClick={onClose}
            aria-label="Tutup"
            className="rounded-kontrol px-2 text-xl leading-none text-slate-500 hover:text-slate-800 focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500/50"
          >
            ×
          </button>
        </div>

        <div className="space-y-3 px-5 py-5">
          {/*
            Pemilih berkas di sini adalah `<input type="file">` BAWAAN peramban, bukan
            tombol bergaya seperti pada modal dokumen di sebelahnya.

            Begitulah tampilannya di layar lama — "Choose File / No file chosen" apa
            adanya — dan itu pula yang membuat nama berkas terpilih terbaca tanpa perlu
            kita menggambarnya sendiri.

            Labelnya karena itu MENEMPEL pada input-nya, sehingga kalimat yang terbaca
            pengguna adalah pula nama yang terbaca pembaca layar.
          */}
          <label
            htmlFor="unggah-detail-salvage"
            className="block text-sm font-semibold text-slate-900"
          >
            Select a local CSV file to load and import
          </label>
          <input
            id="unggah-detail-salvage"
            type="file"
            accept=".csv,text/csv"
            disabled={upload.isPending}
            onChange={(event) => setFile(event.target.files?.[0] ?? null)}
            className="block w-full text-sm text-slate-700 file:mr-3 file:rounded-kontrol file:border file:border-slate-300 file:bg-slate-50 file:px-3 file:py-1.5 file:text-sm file:text-slate-800 hover:file:bg-slate-100 disabled:cursor-not-allowed"
          />

          {upload.error != null && (
            <ErrorMessage
              title="Berkas tidak dapat dibaca"
              description={messageOf(upload.error)}
              tone="penolakan"
            />
          )}
        </div>

        {/* Kaki modal: Cancel abu, Submit berwarna — sebagaimana di layar lama. */}
        <div className="flex gap-3 bg-slate-100 px-5 py-3">
          <Button
            type="button"
            tone="halus"
            className="grow"
            disabled={upload.isPending}
            onClick={onClose}
          >
            Cancel
          </Button>
          <Button
            type="button"
            className="grow"
            disabled={file === null || upload.isPending}
            onClick={submit}
          >
            {upload.isPending ? 'Membaca berkas…' : 'Submit'}
          </Button>
        </div>
      </div>
    </div>
  )
}

/** messageOf mengambil kalimat yang layak dibaca dari sebuah galat. */
function messageOf(error: unknown): string {
  if (error instanceof APIError) return error.message
  if (error instanceof Error && error.message !== '') return error.message
  return 'Coba lagi beberapa saat lagi. Bila terus berulang, hubungi tim teknis.'
}
