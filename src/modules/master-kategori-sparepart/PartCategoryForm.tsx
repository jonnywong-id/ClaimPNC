import { zodResolver } from '@hookform/resolvers/zod'
import { useEffect } from 'react'
import { useForm } from 'react-hook-form'
import { z } from 'zod'

import { APIError, NetworkError } from '@/api/client'
import { ErrorCode, PartCategoryErrorCode, type PartCategory } from '@/api/types'
import { Button } from '@/components/Button'
import { ErrorMessage, type ErrorTone } from '@/components/ErrorMessage'
import { Field } from '@/components/Field'

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
          : {
              title: 'Belum dapat disimpan',
              description: error.message,
              tone: 'penolakan',
            }
      case PartCategoryErrorCode.nameTaken:
        // Ia sudah disorot pada isiannya, tetapi TETAP ditampilkan sebagai kotak pesan —
        // dan itu berbeda dari galat validasi biasa.
        //
        // Alasannya: perbaikannya bukan "betulkan isian" melainkan "cari kategori yang
        // sudah ada, atau pakai nama lain", dan yang memakainya bisa jadi baris di tab
        // Reject yang TIDAK terlihat dari tab yang sedang dibuka. Sorotan di bawah isian
        // tidak cukup menjelaskan ke mana pengguna harus mencari.
        return {
          title: 'Nama itu sudah dipakai',
          description:
            'Kategori lain sudah memakai nama ini. Periksa juga tab Reject — kategori ' +
            'yang sudah ditolak pun tetap memakai namanya.',
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
            'Mengulang tidak akan menolong. Hubungi administrator Claim PNC untuk ' +
            'melengkapi kredensial basis datanya.',
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
export function PartCategoryForm({ editing, isSaving, error, onSave, onCancel }: Props) {
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
  useEffect(() => {
    for (const [column, message] of Object.entries(violationsOf(error))) {
      if (column === 'nama_kategori_sparepart') {
        setError(column, { type: 'server', message })
      }
    }
  }, [error, setError])

  const message = messageFor(error)
  const title = editMode ? 'Ubah Kategori Sparepart' : 'Tambah Kategori Sparepart'

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
        <div>
          <span className="block text-sm font-medium text-slate-700">ID Kategori Sparepart</span>
          <p className="mt-1 rounded border border-slate-200 bg-slate-50 px-3 py-2 text-slate-600">
            {editing.id_kategori_sparepart}
            <span className="ml-2 text-xs text-slate-500">(tidak dapat diubah)</span>
          </p>
        </div>
      )}

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
