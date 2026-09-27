import { zodResolver } from '@hookform/resolvers/zod'
import { useEffect } from 'react'
import { useForm } from 'react-hook-form'
import { z } from 'zod'

import { APIError, NetworkError } from '@/api/client'
import { ErrorCode, type TravelDocument } from '@/api/types'
import { Field } from '@/components/Field'
import { ErrorMessage, type ErrorTone } from '@/components/ErrorMessage'
import { Button } from '@/components/Button'

/**
 * Skema ini SENGAJA tidak memuat satu pun aturan, dan itu bukan kelalaian.
 *
 * Work Owner menetapkan 2026-09-21: layar ini meniru Pega apa adanya, TANPA VALIDASI.
 * Bukti dari sistem lama:
 *
 *   - `Section/BrowseMasterDocumentTravel-Section.xml` tidak memuat satu pun
 *     `pyRequired=true` maupun `pyMaxLength` pada isian judul dokumen.
 *   - `Database/DOCTRAVEL_CVG.prc` menyisipkan dan memperbarui tanpa memeriksa apa pun.
 *   - Tabelnya tidak punya indeks unik atas judul, sehingga judul ganda memang sah.
 *
 * Skemanya tetap ada, dan itu pilihan: ia menyatakan BENTUK isian kepada TypeScript, dan
 * ia tempat aturan pertama akan tinggal bila kelak diputuskan. Menghapus skema dan
 * resolver-nya akan membuat penambahan aturan berikutnya terasa seperti mengubah
 * arsitektur form, bukan menambah satu baris.
 *
 * Ketiadaan aturan di sini COCOK dengan backend: `internal/masterdokumentravel` tidak
 * punya fungsi Check sama sekali, dan kontrak galatnya tidak punya `validasi_gagal`.
 * Bila salah satu sisi kelak diperketat, keduanya harus ikut — dan uji di
 * `masterdokumentravel_test.go` yang menjaga keduanya tetap terlihat.
 */
const schema = z.object({
  judul: z.string(),
})

export type TravelDocumentFields = z.infer<typeof schema>

type Props = {
  /** Baris yang disunting; null berarti penambahan baru. */
  edited: TravelDocument | null
  isSaving: boolean
  error: unknown
  onSave: (values: TravelDocumentFields) => void
  onCancel: () => void
}

type MessageContent = { title: string; description: string; tone: ErrorTone }

/**
 * Mengubah galat penyimpanan menjadi pesan yang dapat ditindaklanjuti.
 *
 * Tidak ada cabang `validasi_gagal` di sini, dan itu konsekuensi langsung dari layar
 * tanpa validasi: server tidak pernah mengirimkannya untuk modul ini.
 */
function messageFor(error: unknown): MessageContent | null {
  if (error instanceof NetworkError) {
    return {
      title: 'Server Claim PNC tidak dapat dihubungi',
      description: 'Isian Anda belum tersimpan. Periksa koneksi jaringan, lalu simpan lagi.',
      tone: 'gangguan',
    }
  }
  if (error instanceof APIError) {
    switch (error.kode) {
      case ErrorCode.travelDocumentNotFound:
        return {
          title: 'Dokumen ini sudah tidak ada',
          description:
            'Mungkin sudah diubah petugas lain. Tutup form ini dan muat ulang daftarnya.',
          tone: 'penolakan',
        }
      case ErrorCode.portalNotStated:
      case ErrorCode.portalUnknown:
        return {
          title: 'Portal entitas belum dipilih',
          description: 'Pilih portal entitas di bagian atas halaman, lalu simpan lagi.',
          tone: 'penolakan',
        }
      case ErrorCode.portalNotReady:
        return {
          title: 'Basis data entitas ini belum tersedia',
          description:
            'Mengulang tidak akan menolong. Hubungi administrator Claim PNC untuk melengkapi kredensial basis datanya.',
          tone: 'gangguan',
        }
      default:
        return {
          title: 'Terjadi kesalahan pada sistem',
          description: 'Isian Anda belum tersimpan. Coba beberapa saat lagi.',
          tone: 'gangguan',
        }
    }
  }
  return null
}

/**
 * TravelDocumentForm adalah satu form untuk DUA mode — tambah dan ubah.
 *
 * Satu form, bukan dua, mengikuti sistem lama: `SetMstDocTravelValue_act` memuat baris
 * ke halaman `TempMstDocTravel` yang sama lalu menandainya "Update", dan
 * `CNMInsertMstDocTravel_act` membaca halaman yang sama saat menyimpan. Isian yang
 * dibandingkan pengguna karena itu berada di tempat yang sama pada kedua mode.
 *
 * ID tidak dapat disunting. Di Pega pun begitu — procedure penyimpannya memakai DOCID
 * hanya sebagai penyaring `WHERE` (`Database/DOCTRAVEL_CVG.prc:37`), dan baris
 * V_LST_DOC_TRAVEL beserta dokumen klaim yang sudah terunggah merujuknya.
 */
export function TravelDocumentForm({ edited, isSaving, error, onSave, onCancel }: Props) {
  const editMode = edited !== null

  const { register, handleSubmit, reset } = useForm<TravelDocumentFields>({
    resolver: zodResolver(schema),
    defaultValues: { judul: edited?.judul ?? '' },
  })

  // Isian disesuaikan ketika baris yang disunting berganti tanpa form ditutup lebih
  // dulu — misalnya pengguna menekan "Ubah" pada baris lain.
  useEffect(() => {
    reset({ judul: edited?.judul ?? '' })
  }, [edited, reset])

  const message = messageFor(error)

  /**
   * Judulnya SAMA untuk mode tambah maupun ubah, dan itu meniru Pega apa adanya.
   *
   * `Section/BrowseMasterDocumentTravel-Section.xml:10333` memberi wadah form ini
   * `pyTitle = "Memperbaharui Data"`, dan wadah itu dipakai kedua mode: tombol Tambah
   * menjalankan `CNMShowInsertMstDocTravel_dt` yang MENGHAPUS `TempMstDocTravel` lalu
   * menyetel `pyLabel = "Update"` — yaitu syarat tampil wadah yang sama.
   *
   * Akibat yang diterima secara sadar (keputusan Work Owner 2026-09-21): layar TAMBAH
   * berjudul "Memperbaharui Data", yang menyatakan hal yang tidak sedang terjadi.
   * `D-13` menang di sini — petugas yang hafal layar lama membaca teks yang sama persis.
   */
  const title = 'Memperbaharui Data'

  return (
    <form
      onSubmit={handleSubmit(onSave)}
      noValidate
      aria-label={title}
      className="space-y-4 rounded-lg border border-slate-200 bg-white p-5 shadow-sm"
    >
      <h2 className="text-base font-semibold text-slate-900">{title}</h2>

      {message && (
        <ErrorMessage title={message.title} description={message.description} tone={message.tone} />
      )}

      {/* ID hanya ditampilkan saat menyunting, dan tidak dapat diubah. Pada penambahan
          ia belum ada — nomornya diterbitkan server dari urutan basis data entitas. */}
      {/*
        Label "ID Kerugian" DITIRU APA ADANYA dari
        `Section/BrowseMasterDocumentTravel-Section.xml:11035`, tempat ia terikat ke
        `TempMstDocTravel.DOCID`.

        Label itu KELIRU: "Kerugian" adalah loss, dan isian ini memuat ID dokumen
        travel. Ia terbawa karena rule ini hasil klon — `pxMoveOriginalKey` pada
        `RDB List/UpdateMstDocTravel-SQL.xml` masih menunjuk `V_M_SURVEYORS`.

        Tetap ditiru atas keputusan Work Owner 2026-09-21: `D-13` menetapkan teks layar
        mengikuti Pega supaya petugas tidak belajar ulang, dan isian ini read-only
        sehingga label yang keliru tidak dapat menyesatkan pengetikan siapa pun.
        Memperbaikinya adalah keputusan tersendiri, bukan disisipkan modul ini.
      */}
      {editMode && (
        <div>
          <span className="block text-sm font-medium text-slate-700">ID Kerugian</span>
          <p className="mt-1 rounded border border-slate-200 bg-slate-50 px-3 py-2 font-mono text-slate-600">
            {edited.id}
            <span className="ml-2 font-sans text-xs text-slate-500">(tidak dapat diubah)</span>
          </p>
        </div>
      )}

      {/*
        Tanpa `maxLength`, dan itu mengikuti layar lama: `pyMaxLength` tidak diisi pada
        isian ini. Lebar kolom NAMADOKUMEN yang sebenarnya belum diketahui — DDL tabelnya
        tidak ada di export (`R-08`) — sehingga memasang angka di sini berarti menebak
        batas yang belum tentu benar, dan tebakan yang terlalu kecil akan menolak judul
        yang sebenarnya sah.
      */}
      <Field
        id="judul"
        label="Judul Dokumen"
        type="text"
        autoFocus
        autoComplete="off"
        {...register('judul')}
      />

      <div className="flex flex-wrap justify-end gap-2 pt-2">
        <Button tone="halus" onClick={onCancel} disabled={isSaving}>
          Batal
        </Button>
        <Button type="submit" tone="utama" disabled={isSaving}>
          {isSaving ? 'Menyimpan…' : 'Simpan'}
        </Button>
      </div>
    </form>
  )
}
