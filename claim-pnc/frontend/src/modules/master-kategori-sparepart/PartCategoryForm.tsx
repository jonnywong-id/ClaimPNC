import { zodResolver } from '@hookform/resolvers/zod'
import { useEffect } from 'react'
import { useForm } from 'react-hook-form'
import { z } from 'zod'

import { ErrorCode, PartCategoryErrorCode, type PartCategory } from '@/api/types'
import { Field } from '@/components/Field'
import { MasterFormActions, MasterFormFrame, ReadOnlyIdRow, useServerViolations } from '@/components/masterform/BoxForm'
import { type CodeMessages, notFoundMessage, portalMessages, saveErrorMessage, validationMessage } from '@/components/masterform/saveErrorMessage'

/**
 * Batas panjang nama harus sama dengan masterkategorisparepart.MaxNameLength di backend.
 *
 * Ia ASUMSI, bukan angka dari DDL: `POOLDATA.GCNM_M_SPAREPART_CATEGORY` tidak ada DDL-nya
 * di export (`R-08`), dan layar lamanya tidak memasang satu pun `pyMaxLength`. Seratus
 * dipilih agar sama dengan batas nama pada Master Sparepart — nilai kolom ini muncul
 * sebagai label di layar itu, dan dua batas berbeda pada dua layar bertetangga hanya akan
 * membingungkan.
 *
 * Diperiksa di dua tempat dengan sengaja: di sini supaya pengguna tahu sebelum mengirim,
 * dan di server karena API dapat ditembak tanpa melewati layar ini. Server tetap yang
 * berwenang — pemeriksaan di sini hanya kenyamanan.
 *
 * Bila angka ini berubah, `internal/masterkategorisparepart/masterkategorisparepart.go`
 * harus ikut berubah. Uji `TestMaxNameLengthMatchesFrontendForm` yang menjaganya.
 */
const MAX_NAME_LENGTH = 100

const schema = z.object({
  nama_kategori_sparepart: z
    .string()
    .trim()
    .min(1, 'Nama kategori sparepart wajib diisi.')
    .max(
      MAX_NAME_LENGTH,
      `Nama kategori sparepart paling panjang ${MAX_NAME_LENGTH} karakter.`,
    ),
})

export type PartCategoryFormValues = z.infer<typeof schema>

type Props = {
  /** Baris yang disunting; null berarti penambahan baru. */
  editing: PartCategory | null
  isSaving: boolean
  error: unknown
  onSave: (values: PartCategoryFormValues) => void
  onCancel: () => void
}

/** Pesan galat penyimpanan per kode; yang tidak dikenal jatuh ke pesan galat sistem. */
const saveMessages: CodeMessages = {
  // Bila detailnya ada, isiannya sudah disorot satu per satu; kotak pesan hanya akan
  // mengulang hal yang sama.
  [ErrorCode.validationFailed]: validationMessage,
  // Ia sudah disorot pada isiannya, tetapi TETAP ditampilkan sebagai kotak pesan —
  // dan itu berbeda dari galat validasi biasa.
  //
  // Alasannya: perbaikannya bukan "betulkan isian" melainkan "cari kategori yang
  // sudah ada, atau pakai nama lain", dan yang memakainya bisa jadi baris di tab
  // Reject yang TIDAK terlihat dari tab yang sedang dibuka. Sorotan di bawah isian
  // tidak cukup menjelaskan ke mana pengguna harus mencari.
  [PartCategoryErrorCode.nameTaken]: {
    title: 'Nama itu sudah dipakai',
    description:
      'Kategori lain sudah memakai nama ini. Periksa juga tab Reject — kategori ' +
      'yang sudah ditolak pun tetap memakai namanya.',
    tone: 'penolakan',
  },
  [ErrorCode.notFound]: notFoundMessage,
  ...portalMessages,
}

/** Isian yang dapat disorot pelanggaran server. */
const violationFields = ['nama_kategori_sparepart'] as const

/**
 * PartCategoryForm adalah satu form untuk DUA mode — tambah dan ubah.
 *
 * Satu form, bukan dua, mengikuti sistem lama: `Activity/SetMasterKategoriSparepart_act`
 * memuat baris ke form yang sama lalu menandainya "Update"
 * (`TempStsClaim.pyLabel := "Update"`).
 *
 * # Hanya SATU isian yang dapat diketik
 *
 * `Section/MasterKategoriSparepartHEApproval-Section.xml` hanya punya dua label —
 * "ID Kategori Sparepart" dan "Nama Kategori Sparepart" — dan yang pertama tidak dapat
 * diubah. Tabelnya memang hanya punya tiga kolom, dan yang ketiga adalah status yang
 * ditetapkan alur persetujuan, bukan diketik pengguna.
 *
 * # ID tidak dapat disunting, dan pada penambahan ia belum ada
 *
 * `RDB List/UpdateMasterSparepartCategory_sql2-SQL.xml` memakai PART_CATEGORY_ID hanya
 * sebagai penyaring `WHERE`, tidak pernah sebagai kolom yang di-SET. Pada penambahan ia
 * diterbitkan server dari isi tabelnya sendiri, sehingga layar tidak punya cara
 * menebaknya — dan tidak boleh mencoba.
 *
 * # Tanpa isian Catatan
 *
 * Tabelnya tidak punya kolom penampung alasan penolakan. Menggambar isian yang diam-diam
 * membuang isinya lebih buruk daripada tidak menggambarnya.
 */
export function PartCategoryForm({ editing, isSaving, error, onSave, onCancel }: Readonly<Props>) {
  const editMode = editing !== null

  const {
    register,
    handleSubmit,
    reset,
    setError,
    formState: { errors },
  } = useForm<PartCategoryFormValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      nama_kategori_sparepart: editing?.nama_kategori_sparepart ?? '',
    },
  })

  // Isian disesuaikan ketika baris yang disunting berganti tanpa form ditutup lebih dulu —
  // misalnya pengguna menekan "Ubah" pada baris lain.
  useEffect(() => {
    reset({ nama_kategori_sparepart: editing?.nama_kategori_sparepart ?? '' })
  }, [editing, reset])

  // Pelanggaran yang dilaporkan server disorot pada isiannya masing-masing, bukan hanya
  // diringkas di satu kotak pesan. Server mengirim SELURUH pelanggaran sekaligus (P-5),
  // dan itu hanya berguna bila layar menyorotnya satu per satu.
  useServerViolations(error, setError, violationFields)

  const message = saveErrorMessage(error, saveMessages)
  const title = editMode ? 'Ubah Kategori Sparepart' : 'Tambah Kategori Sparepart'

  return (
    <MasterFormFrame onSubmit={handleSubmit(onSave)} title={title} message={message}>
      {/* ID hanya ditampilkan saat menyunting, dan tidak dapat diubah. Pada penambahan ia
          belum ada — nomornya diterbitkan server dari isi tabel. */}
      {editMode && <ReadOnlyIdRow label="ID Kategori Sparepart" value={editing.id_kategori_sparepart} />}

      <Field
        id="nama_kategori_sparepart"
        label="Nama Kategori Sparepart"
        type="text"
        autoFocus
        maxLength={MAX_NAME_LENGTH}
        error={errors.nama_kategori_sparepart?.message}
        {...register('nama_kategori_sparepart')}
      />

      {/* Akibat menyimpan dinyatakan di muka, bukan ditemukan setelah tombol ditekan.
          Baris yang sudah disetujui akan HILANG dari dropdown Kategori pada layar Master
          Sparepart selama ia menunggu persetujuan ulang — dan tidak ada apa pun di layar
          ini yang akan menunjukkannya bila tidak disebutkan di sini. */}
      <p className="rounded-kontrol border border-amber-200 bg-amber-50 px-3 py-2 text-xs text-amber-900">
        Menyimpan mengembalikan kategori ini ke tab <strong>Waiting Approval</strong>, persis
        seperti aplikasi lama. Selama menunggu, kategori ini tidak muncul sebagai pilihan di
        layar Master Sparepart dan Master Tipe Sparepart.
      </p>

      <MasterFormActions isSaving={isSaving} onCancel={onCancel} />
    </MasterFormFrame>
  )
}
