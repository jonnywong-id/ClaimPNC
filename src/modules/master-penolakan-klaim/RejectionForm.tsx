import { zodResolver } from '@hookform/resolvers/zod'
import { useEffect } from 'react'
import { useForm, useWatch } from 'react-hook-form'
import { z } from 'zod'

import { APIError, NetworkError } from '@/api/client'
import { ErrorCode, type Rejection, type RejectionInput, type RejectionParent } from '@/api/types'
import { Button } from '@/components/Button'
import { ErrorMessage, type ErrorTone } from '@/components/ErrorMessage'
import { Field } from '@/components/Field'
import { SelectField } from '@/components/SelectField'

/**
 * Batas panjang harus sama dengan masterpenolakan.MaxNameLength di backend.
 *
 * Angka ini ASUMSI yang disadari, bukan panjang kolom yang diterima dari DBA: DDL
 * POOLDATA.MST_PENOLAKAN_KLAIM_1 dan _2 belum ada (R-08). Bila berubah,
 * `internal/masterpenolakan/masterpenolakan.go` harus ikut berubah.
 */
const MAX_NAME_LENGTH = 100

/**
 * Nilai pilihan "buat Status Penolakan 1 baru".
 *
 * Ia sengaja bukan string kosong: kosong berarti "belum dipilih", dan keduanya harus dapat
 * dibedakan — kalau tidak, form yang belum disentuh akan terbaca sebagai permintaan induk
 * baru bernama kosong.
 *
 * Nilainya tidak pernah dikirim ke server; ia diterjemahkan di toInput.
 */
export const NEW_PARENT = '__baru__'

const schema = z
  .object({
    pilihan_status_1: z.string().trim().min(1, 'Status Penolakan 1 wajib dipilih.'),
    nama_status_1_baru: z.string().trim(),
    nama: z
      .string()
      .trim()
      .min(1, 'Status Penolakan 2 wajib diisi.')
      .max(MAX_NAME_LENGTH, `Status Penolakan 2 paling panjang ${MAX_NAME_LENGTH} karakter.`),
  })
  // Nama induk baru hanya wajib bila pengguna memang meminta yang baru. Memeriksanya
  // tanpa syarat akan menuntut isian yang sedang tersembunyi di layar.
  .superRefine((values, ctx) => {
    if (values.pilihan_status_1 !== NEW_PARENT) return

    if (values.nama_status_1_baru === '') {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        path: ['nama_status_1_baru'],
        message: 'Status Penolakan 1 baru wajib diisi.',
      })
      return
    }
    if (values.nama_status_1_baru.length > MAX_NAME_LENGTH) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        path: ['nama_status_1_baru'],
        message: `Status Penolakan 1 paling panjang ${MAX_NAME_LENGTH} karakter.`,
      })
    }
  })

export type RejectionFields = z.infer<typeof schema>

/**
 * toInput menerjemahkan isian layar menjadi badan permintaan.
 *
 * Dua isian di layar menjadi dua field yang TEPAT SATU terisi: `id_status_1` bila induk
 * dipilih dari daftar, `nama_status_1` bila induk baru diminta. Server menolak keduanya
 * terisi bersamaan, dan penerjemahan di sini yang menjaga hal itu tidak pernah terjadi
 * dari layar.
 */
export function toInput(values: RejectionFields): RejectionInput {
  if (values.pilihan_status_1 === NEW_PARENT) {
    return { nama: values.nama, id_status_1: '', nama_status_1: values.nama_status_1_baru }
  }
  return { nama: values.nama, id_status_1: values.pilihan_status_1, nama_status_1: '' }
}

type Props = {
  /** Baris yang sedang disunting; null berarti mode tambah. */
  edited: Rejection | null
  parents: RejectionParent[]
  isSaving: boolean
  error: unknown
  onSave: (values: RejectionFields) => void
  onCancel: () => void
}

type MessageContent = { title: string; description: string; tone: ErrorTone }

/**
 * Mengubah galat penyimpanan menjadi pesan yang dapat ditindaklanjuti.
 *
 * Galat validasi TIDAK ditangani di sini — ia disorot per isian (lihat violationsOf).
 * Yang ditampilkan sebagai kotak pesan hanyalah galat yang tidak menunjuk isian tertentu,
 * karena itulah yang tidak dapat diperbaiki pengguna dengan mengetik.
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
        // Bila detailnya ada, isiannya sudah disorot satu per satu; kotak pesan hanya akan
        // mengulang hal yang sama.
        return Object.keys(error.violations()).length > 0
          ? null
          : { title: 'Belum dapat disimpan', description: error.message, tone: 'penolakan' }
      case ErrorCode.notFound:
        return {
          title: 'Baris ini sudah tidak ada',
          description: 'Mungkin sudah diubah petugas lain. Muat ulang daftarnya.',
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

/** Menyusun isian awal dari baris yang sedang disunting. */
function initialValues(edited: Rejection | null): RejectionFields {
  return {
    // Baris yang sedang disunting SELALU menunjuk induk yang sudah ada — ia tidak dapat
    // tersimpan tanpa induk. Karena itu mode "buat baru" tidak pernah menjadi nilai awal.
    pilihan_status_1: edited?.id_status_1 ?? '',
    nama_status_1_baru: '',
    nama: edited?.nama ?? '',
  }
}

/**
 * RejectionForm adalah satu form untuk DUA mode — tambah dan ubah.
 *
 * Satu form, bukan dua, mengikuti sistem lama: modal "Menambahkan Data" dan "Memperbaharui
 * Data" pada `Section/BrowseNoteRejectClaim-Section.xml` memuat isian yang sama. Isian yang
 * dibandingkan pengguna karena itu berada di tempat yang sama pada kedua mode.
 *
 * # Yang BERBEDA dari layar lama, dan kenapa
 *
 * Layar lama memuat DUA KOTAK TEKS BEBAS: "Status Penolakan 1" dan "Status Penolakan 2".
 * Isi kotak pertama selalu diteruskan ke `MasterPenolakanKlaim1`, yang MENYISIPKAN baris
 * baru pada kedua cabang IF-nya (`MASTERPENOLAKANKLAIM1.prc:9` dan `:14`) — sehingga
 * setiap simpan, termasuk setiap pengubahan, menerbitkan satu Status Penolakan 1 baru dan
 * meninggalkan yang lama menggantung.
 *
 * Work Owner memutuskan memperbaikinya pada 2026-09-19: induk DIPILIH dari daftar, dan
 * baris baru hanya lahir bila pengguna memang meminta yang baru. Itulah sebabnya isian
 * pertama di sini berupa daftar pilihan, bukan kotak teks.
 */
export function RejectionForm({ edited, parents, isSaving, error, onSave, onCancel }: Props) {
  const editMode = edited !== null

  const {
    register,
    handleSubmit,
    reset,
    setError,
    control,
    formState: { errors },
  } = useForm<RejectionFields>({
    resolver: zodResolver(schema),
    defaultValues: initialValues(edited),
  })

  // Isian disesuaikan ketika baris yang disunting berganti tanpa form ditutup lebih dulu —
  // misalnya pengguna menekan "Ubah" pada baris lain.
  useEffect(() => {
    reset(initialValues(edited))
  }, [edited, reset])

  // Pelanggaran yang dilaporkan server disorot pada isiannya masing-masing, bukan hanya
  // diringkas di satu kotak pesan. Server mengirim SELURUH pelanggaran sekaligus (P-5),
  // dan itu hanya berguna bila layar menyorotnya satu per satu.
  //
  // Nama isian server diterjemahkan ke nama isian layar: `id_status_1` dan `nama_status_1`
  // keduanya milik isian induk, yang di layar ini terpecah menjadi daftar pilihan dan
  // kotak nama baru.
  useEffect(() => {
    for (const [column, message] of Object.entries(violationsOf(error))) {
      if (column === 'nama') setError('nama', { type: 'server', message })
      if (column === 'id_status_1') setError('pilihan_status_1', { type: 'server', message })
      if (column === 'nama_status_1') setError('nama_status_1_baru', { type: 'server', message })
    }
  }, [error, setError])

  const parentChoice = useWatch({ control, name: 'pilihan_status_1' })
  const creatingParent = parentChoice === NEW_PARENT

  const message = messageFor(error)
  const title = editMode ? 'Ubah Penolakan Klaim' : 'Tambah Penolakan Klaim'

  const options = [
    ...parents.map((parent) => ({ value: parent.id, label: `${parent.id} — ${parent.nama}` })),
    { value: NEW_PARENT, label: '+ Status Penolakan 1 baru…' },
  ]

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
          belum ada — nomornya diterbitkan server dari isi tabel. */}
      {editMode && (
        <p className="text-sm text-slate-600">
          ID: <span className="font-medium text-slate-900">{edited.id}</span>
        </p>
      )}

      {/* Induk dipilih LEBIH DULU, mengikuti urutan isian pada layar lama. Ia juga yang
          menentukan nama induk yang disalin server ke kolom NOTE_ST. */}
      <SelectField
        id="pilihan_status_1"
        label="Status Penolakan 1"
        options={options}
        error={errors.pilihan_status_1?.message}
        {...register('pilihan_status_1')}
      />

      {creatingParent && (
        <Field
          id="nama_status_1_baru"
          label="Nama Status Penolakan 1 baru"
          type="text"
          maxLength={MAX_NAME_LENGTH}
          hint="Baris baru diterbitkan hanya bila isian ini terisi."
          error={errors.nama_status_1_baru?.message}
          {...register('nama_status_1_baru')}
        />
      )}

      <Field
        id="nama"
        label="Status Penolakan 2"
        type="text"
        maxLength={MAX_NAME_LENGTH}
        error={errors.nama?.message}
        {...register('nama')}
      />

      {/* Akibat menyimpan dinyatakan SEBELUM pengguna menekan Simpan, bukan ditemukan
          setelah terjadi. `MASTERPENOLAKANKLAIM2.prc:14` menyetel STATUS='0' pada setiap
          pengubahan, dan perilaku itu dipertahankan atas keputusan Work Owner. */}
      {editMode && edited.status !== '0' && (
        <ErrorMessage
          title="Menyimpan akan membatalkan persetujuan"
          description={`Baris ini berstatus ${edited.status_label}. Setelah disimpan, statusnya kembali MENUNGGU dan harus disetujui ulang lewat Inbox Manager.`}
          tone="penolakan"
        />
      )}

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
