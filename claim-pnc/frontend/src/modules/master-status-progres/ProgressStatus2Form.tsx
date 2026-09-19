import { zodResolver } from '@hookform/resolvers/zod'
import { useEffect } from 'react'
import { useForm } from 'react-hook-form'
import { z } from 'zod'

import { APIError, NetworkError } from '@/api/client'
import { ErrorCode, type ProgressStatus2Parent } from '@/api/types'
import { Field } from '@/components/Field'
import { SelectField } from '@/components/SelectField'
import { ErrorMessage, type ErrorTone } from '@/components/ErrorMessage'
import { Button } from '@/components/Button'

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
  parents: ProgressStatus2Parent[]
  isSaving: boolean
  error: unknown
  onSave: (values: ProgressStatus2Fields) => void
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
        // mengulang hal yang sama. Induk yang sudah tidak ada pun sampai ke sini sebagai
        // pelanggaran pada isian `id_induk`, sehingga keterangannya menempel di dropdown.
        return Object.keys(error.violations()).length > 0
          ? null
          : {
              title: 'Belum dapat disimpan',
              description: error.message,
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
 * ProgressStatus2Form adalah form penambahan Master Status Progres 2.
 *
 * SATU mode saja — tambah. Berbeda dari tingkat 1 yang memakai satu form untuk dua mode,
 * di sini tidak ada mode ubah karena sistem lama tidak punya penyuntingan yang benar-benar
 * bekerja: tombol "Update" pada layar lama menulis ke TABEL LAIN
 * (`UpdateStatusProgress2_sql` menyentuh GCNM_PROGRESS_CLAIM) dengan dua page klipboard
 * yang tidak pernah diisi, sehingga tabel master tidak berubah sama sekali.
 *
 * Dua isian saja, persis seperti layar lama: `Section/BrowseStatusProgress2-Section.xml`
 * hanya mengikat induk (dropdown) dan nama. ID diterbitkan server dari isi tabel;
 * nama induk disalin server dari baris induknya; TIPE tidak pernah ditulis.
 */
export function ProgressStatus2Form({ parents, isSaving, error, onSave, onCancel }: Props) {
  const {
    register,
    handleSubmit,
    setError,
    formState: { errors },
  } = useForm<ProgressStatus2Fields>({
    resolver: zodResolver(schema),
    defaultValues: { nama: '', id_induk: '' },
  })

  // Pelanggaran yang dilaporkan server disorot pada isiannya masing-masing, bukan hanya
  // diringkas di satu kotak pesan. Server mengirim SELURUH pelanggaran sekaligus (P-5),
  // dan itu hanya berguna bila layar menyorotnya satu per satu.
  useEffect(() => {
    for (const [column, message] of Object.entries(violationsOf(error))) {
      if (column === 'nama' || column === 'id_induk') {
        setError(column, { type: 'server', message })
      }
    }
  }, [error, setError])

  const message = messageFor(error)

  return (
    <form
      onSubmit={handleSubmit(onSave)}
      noValidate
      aria-label="Tambah Status Progres 2"
      className="space-y-4 rounded-lg border border-slate-200 bg-white p-5 shadow-sm"
    >
      <h2 className="text-base font-semibold text-slate-900">Tambah Status Progres 2</h2>

      {message && (
        <ErrorMessage title={message.title} description={message.description} tone={message.tone} />
      )}

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
