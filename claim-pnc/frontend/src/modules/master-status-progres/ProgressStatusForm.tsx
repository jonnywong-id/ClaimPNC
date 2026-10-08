import { zodResolver } from '@hookform/resolvers/zod'
import { useEffect } from 'react'
import { useForm } from 'react-hook-form'
import { z } from 'zod'

import { type ClaimPosition, type ProgressStatus } from '@/api/types'
import { Field } from '@/components/Field'
import { SelectField } from '@/components/SelectField'
import { MasterFormActions, MasterFormFrame, ReadOnlyIdRow, useServerViolations } from '@/components/masterform/BoxForm'
import { saveErrorMessage } from '@/components/masterform/saveErrorMessage'

/**
 * Batas panjang nama harus sama dengan masterstatusprogres.MaxNameLength di backend.
 *
 * Nilainya panjang kolom STS_PROGRESS1 yang ditetapkan Work Owner 2026-09-17 — bukan
 * tebakan. Diperiksa di dua tempat dengan sengaja: di sini supaya pengguna tahu sebelum
 * mengirim, dan di server karena API dapat ditembak tanpa melewati layar ini. Server
 * tetap yang berwenang — pemeriksaan di sini hanya kenyamanan.
 *
 * Bila angka ini berubah, `internal/masterstatusprogres/masterstatusprogres.go` harus
 * ikut berubah.
 */
const MAX_NAME_LENGTH = 100

const schema = z.object({
  nama: z
    .string()
    .trim()
    .min(1, 'Nama status progres wajib diisi.')
    .max(MAX_NAME_LENGTH, `Nama status progres paling panjang ${MAX_NAME_LENGTH} karakter.`),
  kode_posisi: z.string().trim().min(1, 'Posisi klaim wajib dipilih.'),
})

export type ProgressStatusFields = z.infer<typeof schema>

type Props = {
  /** Baris yang disunting; null berarti penambahan baru. */
  edited: ProgressStatus | null
  positions: ClaimPosition[]
  isSaving: boolean
  error: unknown
  onSave: (values: ProgressStatusFields) => void
  onCancel: () => void
}

/** Isian yang dapat disorot pelanggaran server. */
const violationFields = ['nama', 'kode_posisi'] as const

/**
 * ProgressStatusForm adalah satu form untuk DUA mode — tambah dan ubah.
 *
 * Satu form, bukan dua, mengikuti sistem lama: `UpdateStatusProgress1_act` memuat baris
 * ke modal yang sama lalu menandainya "Update" (`TempDcol.pyLabel`). Isian yang
 * dibandingkan pengguna karena itu berada di tempat yang sama pada kedua mode.
 *
 * ID tidak dapat disunting. Di Pega pun begitu: `UpdateStatusProgress1_sql` memakai
 * ID_PROGRESS hanya sebagai penyaring `WHERE`, tidak pernah sebagai kolom yang di-SET —
 * ia dirujuk `GCNM_PROGRESS_CLAIM.STATUS_PROGRESS1` dan `GCNM_MST_PROGRESS.ID_PROGRESS`.
 */
export function ProgressStatusForm({
  edited,
  positions,
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
  } = useForm<ProgressStatusFields>({
    resolver: zodResolver(schema),
    defaultValues: {
      nama: edited?.nama ?? '',
      kode_posisi: edited?.kode_posisi ?? '',
    },
  })

  // Isian disesuaikan ketika baris yang disunting berganti tanpa form ditutup lebih
  // dulu — misalnya pengguna menekan "Ubah" pada baris lain.
  useEffect(() => {
    reset({ nama: edited?.nama ?? '', kode_posisi: edited?.kode_posisi ?? '' })
  }, [edited, reset])

  // Pelanggaran yang dilaporkan server disorot pada isiannya masing-masing, bukan
  // hanya diringkas di satu kotak pesan. Server mengirim SELURUH pelanggaran sekaligus
  // (P-5), dan itu hanya berguna bila layar menyorotnya satu per satu.
  useServerViolations(error, setError, violationFields)

  const message = saveErrorMessage(error)
  const title = editMode ? 'Ubah Status Progres' : 'Tambah Status Progres'

  return (
    <MasterFormFrame onSubmit={handleSubmit(onSave)} title={title} message={message}>
      {/* ID hanya ditampilkan saat menyunting, dan tidak dapat diubah. Pada penambahan
          ia belum ada — nomornya diterbitkan server dari isi tabel. */}
      {editMode && <ReadOnlyIdRow value={edited.id} />}

      <Field
        id="nama"
        label="Status Progres"
        type="text"
        autoFocus
        maxLength={MAX_NAME_LENGTH}
        error={errors.nama?.message}
        {...register('nama')}
      />

      <SelectField
        id="kode_posisi"
        label="Posisi"
        options={positions.map((p) => ({ value: p.kode, label: p.nama }))}
        error={errors.kode_posisi?.message}
        {...register('kode_posisi')}
      />

      <MasterFormActions isSaving={isSaving} onCancel={onCancel} />
    </MasterFormFrame>
  )
}
