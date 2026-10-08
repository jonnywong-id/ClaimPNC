import { zodResolver } from '@hookform/resolvers/zod'
import { useEffect } from 'react'
import { useForm } from 'react-hook-form'
import { z } from 'zod'

import { ErrorCode, type DocumentType } from '@/api/types'
import { Field } from '@/components/Field'
import { MasterFormActions, MasterFormFrame } from '@/components/masterform/BoxForm'
import { type CodeMessages, portalMessages, saveErrorMessage } from '@/components/masterform/saveErrorMessage'

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

/** Pesan galat penyimpanan per kode; yang tidak dikenal jatuh ke pesan galat sistem. */
const saveMessages: CodeMessages = {
  [ErrorCode.documentTypeNotFound]: {
    title: 'Tipe dokumen ini sudah tidak ada',
    description:
      'Mungkin sudah diubah petugas lain. Tutup form ini dan muat ulang daftarnya.',
    tone: 'penolakan',
  },
  ...portalMessages,
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
export function DocumentTypeForm({ edited, isSaving, error, onSave, onCancel }: Readonly<Props>) {
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

  const message = saveErrorMessage(error, saveMessages)

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
    <MasterFormFrame onSubmit={handleSubmit(onSave)} title={title} message={message}>
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

      <MasterFormActions isSaving={isSaving} onCancel={onCancel} />
    </MasterFormFrame>
  )
}
