import { useEffect, useRef, useState } from 'react'

import { APIError } from '@/api/client'
import { Button } from '@/components/Button'
import { ErrorMessage } from '@/components/ErrorMessage'

import { useAttachSalvageDocuments } from './api'

/**
 * Batas yang DIBACA PENGGUNA pada modal ini, dan yang benar-benar ditegakkan.
 *
 * Keduanya tertulis merah di layar lama, dan keduanya disalin apa adanya. Mereka konstanta
 * di sini — bukan master data — karena batas teknis unggahan bukan kebijakan bisnis yang
 * berubah tanpa deploy (`D-15` mengatur nilai BISNIS).
 *
 * Yang menegakkannya di layar bukan hiasan: berkas yang melewati batas baru akan ditolak
 * penyimpanan dokumen setelah seluruh isinya terkirim, dan pada koneksi kantor cabang itu
 * berarti menunggu lama untuk sebuah penolakan yang sudah dapat diketahui sejak awal.
 */
const MAX_DOC = 5
const MAX_SIZE_BYTES = 1024 * 1024

/**
 * Modal **"UploadDocument_Salvage"** — di balik tombol "Upload file" pada form salvage.
 *
 * # Artefak yang dibaca
 *
 *	Section/TambahData_Salvage-Section.xml:13259  tombol pemanggil (local action)
 *	Flow Action/UploadDocument_Salvage-FA.xml     pembungkusnya
 *	Section/SalvageUploadDocumentAll-Section.xml  ISI modal ini
 *	Activity/ReViewUploadDocumentSalvage-Act.xml  pra-proses — membuka lampiran
 *
 * Daftar berkas di bawah mengikuti section aslinya: ia `dragDropFileUpload.pxResults`,
 * yakni berkas yang BARU DIPILIH di modal ini dan belum tersimpan, berkolom **"Nama
 * File"** dengan satu ikon pembuang per baris. Kedua kalimat batasnya disalin kata demi
 * kata (`:5413` dan sekitarnya).
 *
 * # Submit MENYIMPAN, mengikuti Pega apa adanya
 *
 * Work Owner memutuskan 2026-10-03 bahwa unggahan ini MENGIKUTI PEGA apa adanya —
 * base64 ke dalam Oracle — dan `D-16`, yang menetapkan dokumen masuk ke API storage
 * internal, dengan sadar DIKECUALIKAN untuk jalur ini.
 *
 * Rantai `SaveFilePenunjangBySalvage` yang DIBAWA:
 *
 *	├─ base64                               GCNMUploadResult64
 *	├─ tabel sementara                      TEMP_SET_ATTACHMENT_64BIT.prc
 *	├─ tabel permanen                       SET_ATTACHMENT_64BIT.prc
 *	├─ riwayat                              InsertDokumentHistoriKlaimPNC
 *	└─ penaut ke pengajuan                  InsertSalvageDocument
 *
 * Yang TIDAK dibawa: konversi PNG/JPG menjadi AVIF lewat layanan `aiimage`, pengiriman
 * ke Google Storage, cabang SIMASBID ke balai lelang, dan `InsertDokumenPNC`. Keempatnya
 * menyentuh sistem di luar modul ini.
 *
 * Nama berkas yang tersimpan DISUSUN SERVER, bukan diambil dari nama aslinya — itulah
 * sebabnya jawabannya membawa `nama_tersimpan` yang berbeda dari yang dipilih pengguna.
 */
export function UploadDocumentModal({
  claimNo,
  salvageID,
  onClose,
  onSaved,
}: {
  /** Klaim yang dilampiri. Kosong berarti nomor klaimnya belum diisi di form. */
  claimNo: string

  /**
   * Pengajuan yang dilampiri.
   *
   * Kosong pada form pengajuan BARU — pengajuannya belum punya nomor. Dokumennya tetap
   * tersimpan dan menempel pada klaimnya; yang dilewati hanyalah baris penautnya.
   */
  salvageID: string

  onClose: () => void

  /** Dipanggil setelah dokumen benar-benar tersimpan, membawa pesan dari server. */
  onSaved: (message: string) => void
}) {
  const [files, setFiles] = useState<File[]>([])
  const [failure, setFailure] = useState('')

  const input = useRef<HTMLInputElement>(null)
  const attach = useAttachSalvageDocuments()

  // Escape menutup modal, seperti dialog mana pun yang dikenal pengguna — KECUALI saat
  // pengiriman sedang berjalan. Menutupnya di tengah jalan tidak membatalkan apa pun di
  // server, dan pengguna akan mengira unggahannya gagal padahal sedang tersimpan.
  useEffect(() => {
    function onKey(event: KeyboardEvent) {
      if (event.key === 'Escape' && !attach.isPending) onClose()
    }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [onClose, attach.isPending])

  function submit() {
    attach.mutate(
      { nomor_klaim: claimNo, id_salvage: salvageID, berkas: files },
      {
        onSuccess: (result) => {
          onSaved(result.pesan)
          onClose()
        },
      },
    )
  }

  function choose(event: React.ChangeEvent<HTMLInputElement>) {
    const chosen = [...(event.target.files ?? [])]

    // Nilai input dikosongkan supaya berkas yang SAMA dapat dipilih lagi setelah ditolak.
    // Tanpa ini, memilih berkas yang sama dua kali tidak memicu perubahan apa pun.
    event.target.value = ''

    if (chosen.length === 0) return

    // Kedua batas diperiksa terhadap gabungan yang sudah dipilih, bukan terhadap yang
    // baru saja ditambahkan: pengguna dapat memilih tiga berkas lalu tiga lagi.
    const gabungan = [...files, ...chosen]

    if (gabungan.length > MAX_DOC) {
      setFailure(
        `Paling banyak ${MAX_DOC} dokumen. Anda memilih ${gabungan.length}.`,
      )
      return
    }

    const kebesaran = gabungan.filter((file) => file.size > MAX_SIZE_BYTES)
    if (kebesaran.length > 0) {
      setFailure(
        `Setiap berkas paling besar 1 MB. Yang melewati batas: ` +
          kebesaran.map((file) => file.name).join(', ') +
          '.',
      )
      return
    }

    setFailure('')
    setFiles(gabungan)
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-start justify-center bg-slate-900/40 px-4 py-8"
      role="dialog"
      aria-modal="true"
      aria-labelledby="judul-unggah-salvage"
    >
      <div className="max-h-full w-full max-w-sm overflow-y-auto rounded-kartu bg-white shadow-angkat">
        {/* Kepala modal: judul apa adanya beserta tanda silangnya. */}
        <div className="flex items-center justify-between border-b border-slate-200 px-5 py-3">
          <h2 id="judul-unggah-salvage" className="text-base font-medium text-slate-900">
            UploadDocument_Salvage
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

        <div className="space-y-6 px-5 py-5">
          {/*
            Kedua batas disalin kata demi kata dari section aslinya — "Max Doc : 5 Doc."
            dan "Max Size /file : 1 MB". Huruf besarnya datang dari gaya layar, bukan dari
            teksnya, jadi `uppercase` yang menanganinya dan teks di sini tetap apa adanya.
          */}
          <div className="space-y-4 text-sm font-medium text-red-600 uppercase">
            <p>Max Doc : {MAX_DOC} Doc.</p>
            <p>Max Size /file : 1 MB</p>
          </div>

          <div className="py-6 text-center">
            <input
              ref={input}
              type="file"
              multiple
              onChange={choose}
              className="hidden"
              aria-hidden="true"
              tabIndex={-1}
            />
            <button
              type="button"
              onClick={() => input.current?.click()}
              className="rounded-kontrol border border-blue-600 px-3 py-1.5 text-sm text-blue-700 hover:bg-blue-50 focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500/50"
            >
              Select file(s)
            </button>
          </div>

          {failure !== '' && (
            <p className="text-sm text-red-700" role="alert">
              {failure}
            </p>
          )}

          {files.length > 0 && (
            // Judulnya "Nama File", sama seperti kolom tunggal grid pada section aslinya.
            <ul className="space-y-1 text-sm text-slate-700" aria-label="Nama File">
              {files.map((file, index) => (
                <li key={`${file.name}-${index}`} className="flex items-center gap-2">
                  <span className="min-w-0 grow truncate">{file.name}</span>
                  <button
                    type="button"
                    disabled={attach.isPending}
                    onClick={() => {
                      setFailure('')
                      setFiles((current) =>
                        current.filter((_, position) => position !== index),
                      )
                    }}
                    aria-label={`Buang ${file.name}`}
                    className="rounded-kontrol px-2 text-sm text-red-700 hover:bg-red-50 focus:outline-none focus-visible:ring-2 focus-visible:ring-red-500/50"
                  >
                    Buang
                  </button>
                </li>
              ))}
            </ul>
          )}

          {attach.error != null && (
            <ErrorMessage
              title="Dokumen tidak tersimpan"
              description={messageOf(attach.error)}
              tone="penolakan"
            />
          )}

          {/*
            Pengajuan yang belum punya nomor dinyatakan di MUKA, bukan setelah tersimpan.

            Pada form pengajuan baru, berkas diunggah sebelum Submit ditekan, sehingga
            penaut ke pengajuannya memang belum dapat ditulis. Mengatakannya sesudahnya
            membuat pengguna mengira sesuatu gagal.
          */}
          {salvageID === '' && (
            <p className="rounded-kontrol bg-amber-50 px-3 py-2 text-sm text-amber-900">
              Pengajuan ini belum punya nomor, jadi dokumennya menempel pada klaimnya
              lebih dulu.
            </p>
          )}
        </div>

        {/* Kaki modal: Cancel abu, Submit berwarna — sebagaimana di layar lama. */}
        <div className="flex gap-3 bg-slate-100 px-5 py-3">
          <Button
            type="button"
            tone="halus"
            className="grow"
            disabled={attach.isPending}
            onClick={onClose}
          >
            Cancel
          </Button>
          <Button
            type="button"
            className="grow"
            disabled={files.length === 0 || claimNo === '' || attach.isPending}
            onClick={submit}
          >
            {attach.isPending ? 'Mengirim…' : 'Submit'}
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
