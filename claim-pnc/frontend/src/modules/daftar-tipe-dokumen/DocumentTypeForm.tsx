import { zodResolver } from '@hookform/resolvers/zod'
import { useEffect } from 'react'
import { useForm } from 'react-hook-form'
import { z } from 'zod'

import { APIError, NetworkError } from '@/api/client'
import { ErrorCode, type DocumentType } from '@/api/types'
import { Field } from '@/components/Field'
import { ErrorMessage, type ErrorTone } from '@/components/ErrorMessage'
import { Button } from '@/components/Button'

/**
 * Skema ini SENGAJA tidak memuat satu pun aturan, dan itu bukan kelalaian.
 *
 * Work Owner menetapkan 2026-09-21: layar ini meniru Pega apa adanya, TANPA VALIDASI.
 * Bukti dari sistem lama:
 *
 *   - `Section/BrowseListDocumentType-Section.xml` tidak memuat satu pun `pyRequired=true`
 *     maupun `pyMaxLength` pada kedua isiannya.
 *   - `Database/PEGA_LST_DOC_TYPE.prc` menyisipkan dan memperbarui tanpa memeriksa apa pun.
 *   - Tabelnya tidak punya indeks unik atas nama tipe dokumen, sehingga nama ganda memang
 *     sah.
 *
 * Skemanya tetap ada, dan itu pilihan: ia menyatakan BENTUK isian kepada TypeScript, dan ia
 * tempat aturan pertama akan tinggal bila kelak diputuskan. Menghapus skema dan
 * resolver-nya akan membuat penambahan aturan berikutnya terasa seperti mengubah arsitektur
 * form, bukan menambah satu baris.
 *
 * Ketiadaan aturan di sini COCOK dengan backend: `internal/daftartipedokumen` tidak punya
 * fungsi Check sama sekali, dan kontrak galatnya tidak punya `validasi_gagal`. Bila salah
 * satu sisi kelak diperketat, keduanya harus ikut — dan uji di `daftartipedokumen_test.go`
 * yang menjaga keduanya tetap terlihat.
 */
const schema = z.object({
  tipe_dokumen: z.string(),
  status_proses: z.string(),
})

export type DocumentTypeFields = z.infer<typeof schema>

type Props = {
  /** Baris yang disunting; null berarti penambahan baru. */
  edited: DocumentType | null
  isSaving: boolean
  error: unknown
  onSave: (values: DocumentTypeFields) => void
  onCancel: () => void
}

type MessageContent = { title: string; description: string; tone: ErrorTone }

/**
 * Mengubah galat penyimpanan menjadi pesan yang dapat ditindaklanjuti.
 *
 * Tidak ada cabang `validasi_gagal` di sini, dan itu konsekuensi langsung dari layar tanpa
 * validasi: server tidak pernah mengirimkannya untuk modul ini.
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
      case ErrorCode.documentTypeNotFound:
        return {
          title: 'Tipe dokumen ini sudah tidak ada',
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
 * DocumentTypeForm adalah satu form untuk DUA mode — tambah dan ubah.
 *
 * Satu form, bukan dua, mengikuti sistem lama: `CNMSetListDocumentType_act` memuat baris ke
 * halaman `TempDcol` yang sama lalu menandainya "Update", dan
 * `CNMInsertListDocumentType_act` membaca halaman yang sama saat menyimpan. Isian yang
 * dibandingkan pengguna karena itu berada di tempat yang sama pada kedua mode.
 *
 * ID tidak dapat disunting. Di Pega pun begitu — isiannya bertanda
 * `pyEditOptions=Read-only`, procedure penyimpannya memakai ID hanya sebagai penyaring
 * `WHERE`, dan dua master turunan beserta dokumen klaim yang sudah terunggah merujuknya.
 */
export function DocumentTypeForm({ edited, isSaving, error, onSave, onCancel }: Props) {
  const editMode = edited !== null

  const { register, handleSubmit, reset } = useForm<DocumentTypeFields>({
    resolver: zodResolver(schema),
    defaultValues: {
      tipe_dokumen: edited?.tipe_dokumen ?? '',
      status_proses: edited?.status_proses ?? '',
    },
  })

  // Isian disesuaikan ketika baris yang disunting berganti tanpa form ditutup lebih dulu —
  // misalnya pengguna menekan "Update Data" pada baris lain.
  useEffect(() => {
    reset({
      tipe_dokumen: edited?.tipe_dokumen ?? '',
      status_proses: edited?.status_proses ?? '',
    })
  }, [edited, reset])

  const message = messageFor(error)

  /**
   * Judulnya SAMA untuk mode tambah maupun ubah, dan itu meniru Pega apa adanya.
   *
   * `Section/BrowseListDocumentType-Section.xml` menampilkan wadah form ini dengan syarat
   * `TempDcol.pyLabel='Update'`, dan wadah itu dipakai kedua mode: tombol Tambah
   * menjalankan `CNMShowInsertListDocumentType_dt` yang MENGOSONGKAN seluruh isian
   * `TempDcol` lalu menyetel `pyLabel = "Update"` — yaitu syarat tampil wadah yang sama.
   *
   * Akibat yang diterima secara sadar: layar TAMBAH berjudul "Update Data", yang menyatakan
   * hal yang tidak sedang terjadi. `D-13` menang di sini — petugas yang hafal layar lama
   * membaca teks yang sama persis. Perlakuannya sama dengan Master Dokumen Travel, yang
   * layar tambahnya pun berjudul "Memperbaharui Data".
   */
  const title = 'Update Data'

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

      {/* ID hanya ditampilkan saat menyunting, dan tidak dapat diubah. Pada penambahan ia
          belum ada — nomornya diterbitkan server dari urutan basis data entitas. */}
      {editMode && (
        <div>
          <span className="block text-sm font-medium text-slate-700">ID</span>
          <p className="mt-1 rounded border border-slate-200 bg-slate-50 px-3 py-2 font-mono text-slate-600">
            {edited.id}
            <span className="ml-2 font-sans text-xs text-slate-500">(tidak dapat diubah)</span>
          </p>
        </div>
      )}

      {/*
        Label "Jenis Dokumen" DITIRU APA ADANYA dari `Section/ListDocumentType-Section.xml`,
        dan ia sengaja BERBEDA dari header kolom gridnya yang berbunyi "Tipe Dokumen".
        Kedua teks itu memang tidak sama di layar lama, dan keduanya ditiru (`D-13`).

        Tanpa `maxLength`, mengikuti layar lama: `pyMaxLength` tidak diisi pada isian ini.
        Lebar kolom TYPE_DOCUMENT yang sebenarnya belum diketahui — DDL tabelnya tidak ada
        di export (`R-08`) — sehingga memasang angka di sini berarti menebak batas yang
        belum tentu benar, dan tebakan yang terlalu kecil akan menolak nama yang sah.
      */}
      <Field
        id="tipe_dokumen"
        label="Jenis Dokumen"
        type="text"
        autoFocus
        autoComplete="off"
        {...register('tipe_dokumen')}
      />

      {/*
        Isian TEKS BEBAS, bukan daftar pilihan — dan itu bukan penyederhanaan.

        Di Pega ia `pyEditOptions=Auto` tanpa satu pun daftar pilihan, dan layar Arsip
        Dokumen membacanya sebagai catatan bebas beralias "NoteKasir"
        (`Activity/SearchDataArchiveFilling-Act.xml`). Menjadikannya dropdown berisi nilai
        yang kita karang sendiri akan menolak isian yang selama ini sah, dan menyempitkan
        kolom yang dibaca layar lain.

        Keterangan di bawahnya ada supaya petugas yang membaca "Status Proses" tidak mengira
        ini sakelar aktif/non-aktif.
      */}
      <Field
        id="status_proses"
        label="Status Proses"
        type="text"
        autoComplete="off"
        hint="Catatan bebas, bukan status aktif/non-aktif. Boleh dikosongkan."
        {...register('status_proses')}
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
