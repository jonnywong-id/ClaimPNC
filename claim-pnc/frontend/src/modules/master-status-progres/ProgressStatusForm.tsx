import { zodResolver } from '@hookform/resolvers/zod'
import { useEffect } from 'react'
import { useForm } from 'react-hook-form'
import { z } from 'zod'

import { APIError, NetworkError } from '@/api/client'
import { ErrorCode, type ClaimPosition, type ProgressStatus } from '@/api/types'
import { Field } from '@/components/Field'
import { SelectField } from '@/components/SelectField'
import { ErrorMessage, type ErrorTone } from '@/components/ErrorMessage'
import { Button } from '@/components/Button'

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

type MessageContent = { title: string; description: string; tone: ErrorTone }

/**
 * Mengubah galat penyimpanan menjadi pesan yang dapat ditindaklanjuti.
 *
 * Galat validasi TIDAK ditangani di sini — ia disorot per isian (lihat violationsOf).
 * Yang ditampilkan sebagai kotak pesan hanyalah galat yang tidak menunjuk isian
 * tertentu, karena itulah yang tidak dapat diperbaiki pengguna dengan mengetik.
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
      case ErrorCode.validationFailed:
        // Bila detailnya ada, isiannya sudah disorot satu per satu; kotak pesan hanya
        // akan mengulang hal yang sama. Bila detailnya TIDAK ada — misalnya nomor baru
        // bentrok dengan petugas lain — pesan servernya yang ditampilkan.
        return Object.keys(error.violations()).length > 0
          ? null
          : {
              title: 'Belum dapat disimpan',
              description: error.message,
              tone: 'penolakan',
            }
      case ErrorCode.notFound:
        return {
          title: 'Baris ini sudah tidak ada',
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

/** Mengambil pelanggaran per isian dari galat validasi server. */
function violationsOf(error: unknown): Record<string, string> {
  return error instanceof APIError ? error.violations() : {}
}

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
}: Props) {
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
  useEffect(() => {
    for (const [column, message] of Object.entries(violationsOf(error))) {
      if (column === 'nama' || column === 'kode_posisi') {
        setError(column, { type: 'server', message })
      }
    }
  }, [error, setError])

  const message = messageFor(error)
  const title = editMode ? 'Ubah Status Progres' : 'Tambah Status Progres'

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
          ia belum ada — nomornya diterbitkan server dari isi tabel. */}
      {editMode && (
        <div>
          <span className="block text-sm font-medium text-slate-700">ID</span>
          <p className="mt-1 rounded border border-slate-200 bg-slate-50 px-3 py-2 text-slate-600">
            {edited.id}
            <span className="ml-2 text-xs text-slate-500">(tidak dapat diubah)</span>
          </p>
        </div>
      )}

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
