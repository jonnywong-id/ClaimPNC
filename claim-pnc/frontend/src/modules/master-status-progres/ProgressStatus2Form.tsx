import { zodResolver } from '@hookform/resolvers/zod'
import { useEffect } from 'react'
import { useForm } from 'react-hook-form'
import { z } from 'zod'

import { ErrorCode, type ProgressStatus2, type ProgressStatus2Parent } from '@/api/types'
import { Field } from '@/components/Field'
import { SelectField } from '@/components/SelectField'
import { MasterFormActions, MasterFormFrame, ReadOnlyIdRow, useServerViolations } from '@/components/masterform/BoxForm'
import {
  portalMessages,
  saveErrorMessage,
  validationMessage,
  type CodeMessages,
} from '@/components/masterform/saveErrorMessage'

/**
 * Batas panjang nama harus sama dengan masterstatusprogres.MaxNameLength2 di backend.
 *
 * BERBEDA dari tingkat 1, angka ini adalah ASUMSI yang disadari — bukan panjang kolom yang
 * diterima dari Work Owner. DDL POOLDATA.GCNM_MST_PROGRESS belum ada (R-08), dan kedua
 * kolom menyimpan hal yang sejenis pada tabel yang sekerabat.
 *
 * Bila angka ini berubah, `internal/masterstatusprogres/masterstatusprogres2.go` harus ikut
 * berubah.
 */
const MAX_NAME_LENGTH = 100

const schema = z.object({
  nama: z
    .string()
    .trim()
    .min(1, 'Nama status progres 2 wajib diisi.')
    .max(MAX_NAME_LENGTH, `Nama status progres 2 paling panjang ${MAX_NAME_LENGTH} karakter.`),
  id_induk: z.string().trim().min(1, 'Status Progres 1 wajib dipilih.'),
})

export type ProgressStatus2Fields = z.infer<typeof schema>

type Props = {
  /** Baris yang disunting; null berarti penambahan baru. */
  edited: ProgressStatus2 | null
  parents: ProgressStatus2Parent[]
  isSaving: boolean
  error: unknown
  onSave: (values: ProgressStatus2Fields) => void
  onCancel: () => void
}

/**
 * Pesan galat penyimpanan. Galat validasi tidak ditampilkan sebagai kotak bila isiannya
 * sudah disorot; kode "baris tidak ada" sengaja tidak punya pesan sendiri di modul ini.
 */
const saveMessages: CodeMessages = {
  [ErrorCode.validationFailed]: validationMessage,
  ...portalMessages,
}

/** Isian yang dapat disorot pelanggaran server. */
const violationFields = ['nama', 'id_induk'] as const

/**
 * ProgressStatus2Form adalah satu form untuk DUA mode — tambah dan ubah.
 *
 * Satu form, bukan dua, mengikuti tingkat 1 dan mengikuti sistem lama: modal yang sama
 * dipakai kedua mode, hanya judulnya yang berganti.
 *
 * # Kenapa mode ubah ADA, padahal sistem lama tidak punya penyuntingan yang bekerja
 *
 * Tombol "Update" pada layar lama menulis ke TABEL LAIN — `UpdateStatusProgress2_sql`
 * menyentuh `GCNM_PROGRESS_CLAIM` — lewat dua page klipboard yang tidak pernah diisi,
 * sehingga tabel master tidak berubah sama sekali.
 *
 * Work Owner memutuskan menambahkan penyuntingan yang benar-benar bekerja pada
 * 2026-09-20, setelah ditunjukkan bahwa tanpanya salah ketik nama tidak dapat diperbaiki
 * — tombol hapus pun tidak ada. Ia **perilaku baru**, bukan pemindahan, dan selisihnya
 * pada uji kesetaraan gerbang 1 dinyatakan di muka sebagai perbaikan terencana.
 *
 * Dua isian saja, persis seperti layar lama: `Section/BrowseStatusProgress2-Section.xml`
 * hanya mengikat induk (dropdown) dan nama. ID diterbitkan server dari isi tabel;
 * nama induk disalin server dari baris induknya; TIPE tidak pernah ditulis.
 */
export function ProgressStatus2Form({
  edited,
  parents,
  isSaving,
  error,
  onSave,
  onCancel,
}: Readonly<Props>) {
  const editMode = edited !== null

  const {
    register,
    handleSubmit,
    reset,
    setError,
    formState: { errors },
  } = useForm<ProgressStatus2Fields>({
    resolver: zodResolver(schema),
    defaultValues: {
      nama: edited?.nama ?? '',
      id_induk: edited?.id_induk ?? '',
    },
  })

  // Isian disesuaikan ketika baris yang disunting berganti tanpa form ditutup lebih dulu
  // — misalnya pengguna menekan "Ubah" pada baris lain.
  useEffect(() => {
    reset({ nama: edited?.nama ?? '', id_induk: edited?.id_induk ?? '' })
  }, [edited, reset])

  // Pelanggaran yang dilaporkan server disorot pada isiannya masing-masing, bukan hanya
  // diringkas di satu kotak pesan. Server mengirim SELURUH pelanggaran sekaligus (P-5),
  // dan itu hanya berguna bila layar menyorotnya satu per satu.
  useServerViolations(error, setError, violationFields)

  const message = saveErrorMessage(error, saveMessages)
  const title = editMode ? 'Ubah Status Progres 2' : 'Tambah Status Progres 2'

  return (
    <MasterFormFrame onSubmit={handleSubmit(onSave)} title={title} message={message}>
      {/* ID hanya ditampilkan saat menyunting, dan tidak dapat diubah. Pada penambahan ia
          belum ada — nomornya diterbitkan server dari isi tabel.

          Ia juga tidak akan pernah dapat diubah: `ID_MST` dirujuk
          `GCNM_PROGRESS_CLAIM.STATUS_PROGRESS2` pada data klaim yang sudah berjalan. */}
      {/* Sebutannya "No", menyamai judul kolom pertama pada grid — satu layar tidak
          boleh menyebut hal yang sama dengan dua nama. */}
      {editMode && <ReadOnlyIdRow label="No" value={edited.id} />}

      {/* Induk dipilih LEBIH DULU, mengikuti urutan isian pada layar lama. Ia juga yang
          menentukan nama induk yang disalin server ke kolom STS_PROGRESS1. */}
      <SelectField
        id="id_induk"
        label="Status Progres 1"
        options={parents.map((p) => ({ value: p.id, label: `${p.id} — ${p.nama}` }))}
        error={errors.id_induk?.message}
        {...register('id_induk')}
      />

      <Field
        id="nama"
        label="Status Progres 2"
        type="text"
        maxLength={MAX_NAME_LENGTH}
        error={errors.nama?.message}
        {...register('nama')}
      />

      <MasterFormActions isSaving={isSaving} onCancel={onCancel} />
    </MasterFormFrame>
  )
}
