import { zodResolver } from '@hookform/resolvers/zod'
import { useEffect } from 'react'
import { useFieldArray, useForm } from 'react-hook-form'
import { z } from 'zod'

import { APIError, NetworkError } from '@/api/client'
import { ErrorCode, type Business, type SimasOnlineCauseOfLoss } from '@/api/types'
import { Field } from '@/components/Field'
import { ComboField } from '@/components/ComboField'
import { ErrorMessage, type ErrorTone } from '@/components/ErrorMessage'
import { Button } from '@/components/Button'

/**
 * Kedua batas harus sama dengan MaxDescriptionLength dan MaxBusinessNameLength di
 * `internal/mastercolsimasonline/mastercolsimasonline.go`.
 *
 * Kolomnya berlebar VARCHAR2(200) di POOLDATA.M_CAUSE_OF_LOSS_ONLINE dan
 * POOLDATA.M_CAUSE_OF_LOSS_ONLINE_DETAIL; angka di bawah sengaja ditahan di separuhnya,
 * dan layar Pega sendiri tidak membatasi apa pun. Angkanya diperiksa di dua tempat
 * dengan sengaja: di sini supaya pengguna tahu
 * sebelum mengirim, dan di server karena API dapat ditembak tanpa melewati layar ini.
 * Server tetap yang berwenang.
 *
 * Bila angka ini berubah, berkas Go di atas harus ikut berubah — dan uji
 * `TestLengthLimitsAreMirroredInTheFrontend` yang menjaga pengingat itu tetap terlihat.
 */
const MAX_NAME_LENGTH = 100
const MAX_BUSINESS_NAME_LENGTH = 100

/**
 * Yang diperiksa di layar hanyalah PANJANG, dan itu disengaja.
 *
 * Layar Pega menandai seluruh isiannya `pyRequired=false` dan tidak punya satu pun
 * Validate rule, sehingga nama kosong maupun bisnis kembar sama-sama sah. Work Owner menetapkan 2026-09-21 layar disamakan dengan Pega.
 *
 * Batas panjang tetap ada karena ia bukan aturan bisnis melainkan penjaga terhadap lebar
 * kolom basis data — tanpa itu, nilai yang kepanjangan ditolak Oracle dengan ORA-12899
 * yang muncul sebagai galat 500 dan tidak dapat dibaca pengguna.
 */
const schema = z.object({
  nama: z
    .string()
    .trim()
    .max(MAX_NAME_LENGTH, `Nama Cause of loss paling panjang ${MAX_NAME_LENGTH} karakter.`),
  // Disimpan sebagai senarai objek, bukan senarai teks, karena useFieldArray menuntut
  // setiap barisnya berupa objek agar dapat memberinya kunci yang stabil.
  bisnis: z.array(
    z.object({
      nama: z
        .string()
        .trim()
        .max(
          MAX_BUSINESS_NAME_LENGTH,
          `Nama bisnis paling panjang ${MAX_BUSINESS_NAME_LENGTH} karakter.`,
        ),
    }),
  ),
})

export type CauseOfLossFields = z.infer<typeof schema>

type Props = {
  /** Baris yang disunting; null berarti penambahan baru. */
  edited: SimasOnlineCauseOfLoss | null
  /** Saran bisnis, dibaca dari POOLDATA.BUSINESS milik entitas yang aktif. */
  businesses: Business[]
  /** Pemetaan bisnis baris yang disunting masih dimuat. */
  isLoadingBusinessMapping: boolean
  isSaving: boolean
  error: unknown
  onSave: (values: CauseOfLossFields) => void
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
        // Bila detailnya ada, isiannya sudah disorot satu per satu; kotak pesan hanya
        // akan mengulang hal yang sama.
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
          description: 'Mungkin sudah diubah petugas lain. Tutup form ini dan muat ulang daftarnya.',
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

/** Menyusun nilai awal form dari baris yang disunting. */
function valuesOf(edited: SimasOnlineCauseOfLoss | null): CauseOfLossFields {
  return {
    nama: edited?.nama ?? '',
    bisnis: (edited?.bisnis ?? []).map((b) => ({ nama: b.nama })),
  }
}

/**
 * CauseOfLossForm adalah satu form untuk DUA mode — tambah dan ubah.
 *
 * Satu form, bukan dua, mengikuti sistem lama: `PageNewForSetSimasOnlineCOL` dan
 * `SetDataCauseofflossOnline` sama-sama mengisi satu halaman `TempCauseOfLoss` lalu
 * menandai modusnya lewat `pyLabel`. Isian yang dibandingkan pengguna karena itu berada
 * di tempat yang sama pada kedua mode.
 *
 * # Susunan isiannya disamakan PERSIS dengan Pega
 *
 * Diambil dari urutan kemunculannya di `Section/Online_BrowseCauseOfLoss-Section.xml`
 * (`D-13`: tata letak ditiru supaya pengguna tidak perlu belajar ulang):
 *
 *	baris 4301  ID Kerugian          M_COL_ID     read-only
 *	baris 4813  Nama Cause of loss   COL_DESC
 *	baris 5874  Bisnis               BISNISID     repeat grid, banyak baris
 *
 * Isian "ID Master Kerugian" (MST_COL_ID) yang dulu berada di antara keduanya DICABUT
 * 2026-09-23 atas keputusan Work Owner — kolomnya sudah tidak dipakai.
 *
 * Perhatikan label isian pertama adalah "ID Kerugian", bukan "ID". Itu sempat keliru pada
 * versi pertama modul ini dan dikoreksi setelah Work Owner memeriksanya pada 2026-09-21.
 *
 * Judulnya pun SAMA untuk kedua modus — "Memperbaharui Data Simas Online" — persis
 * seperti layar lama, yang menandai modusnya lewat `pyLabel` dan bukan lewat judul.
 */
export function CauseOfLossForm({
  edited,
  businesses,
  isLoadingBusinessMapping,
  isSaving,
  error,
  onSave,
  onCancel,
}: Props) {
  const editMode = edited !== null

  const {
    control,
    register,
    handleSubmit,
    reset,
    setError,
    formState: { errors },
  } = useForm<CauseOfLossFields>({
    resolver: zodResolver(schema),
    defaultValues: valuesOf(edited),
  })

  // `bisnis` adalah grid, bukan satu isian. useFieldArray yang memegang barisnya supaya
  // setiap baris punya kunci stabil — tanpa itu, menghapus baris tengah akan membuat
  // React memakai ulang elemen input dan nilai baris berikutnya ikut bergeser.
  const { fields, append, remove } = useFieldArray({ control, name: 'bisnis' })

  // Isian disesuaikan ketika baris yang disunting berganti tanpa form ditutup lebih dulu
  // — misalnya pengguna menekan "Ubah" pada baris lain, atau saat pemetaan bisnisnya
  // baru selesai dimuat dari server.
  useEffect(() => {
    reset(valuesOf(edited))
  }, [edited, reset])

  // Pelanggaran yang dilaporkan server disorot pada isiannya masing-masing, bukan hanya
  // diringkas di satu kotak pesan. Server mengirim SELURUH pelanggaran sekaligus
  // (`P-5`), dan itu hanya berguna bila layar menyorotnya satu per satu.
  useEffect(() => {
    for (const [column, message] of Object.entries(violationsOf(error))) {
      if (column === 'nama') {
        setError(column, { type: 'server', message })
      } else if (column === 'bisnis') {
        setError('bisnis', { type: 'server', message })
      }
    }
  }, [error, setError])

  const message = messageFor(error)

  // Judul yang sama untuk kedua modus, mengikuti layar lama. Beda modus tetap terlihat
  // dari ada-tidaknya baris "ID Kerugian" di bawahnya.
  const title = 'Memperbaharui Data Simas Online'

  const businessSuggestions = businesses.map((b) => b.nama)

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

      {/* ID Kerugian hanya ditampilkan saat menyunting, dan tidak dapat diubah. Pada
          penambahan ia belum ada — kodenya diterbitkan server dari urutan basis data.
          Di Pega pun isiannya `pyReadOnly=true`. */}
      {editMode && (
        <div>
          <span className="block text-sm font-medium text-slate-700">ID Kerugian</span>
          <p className="mt-1 rounded border border-slate-200 bg-slate-50 px-3 py-2 text-slate-600">
            {edited.id}
            <span className="ml-2 text-xs text-slate-500">(tidak dapat diubah)</span>
          </p>
        </div>
      )}

      <Field
        id="nama"
        label="Nama Cause of loss"
        type="text"
        autoFocus
        maxLength={MAX_NAME_LENGTH}
        error={errors.nama?.message}
        {...register('nama')}
      />

      {/* Bisnis adalah GRID, bukan satu isian. Di Pega ia
          `pyPageListProperty = TempCauseOfLoss.BISNISID` berkelas
          ASM-FW-GISFW-Int-BUSINESS — satu cause of loss dapat dipakai banyak bisnis. */}
      <fieldset className="rounded-kontrol border border-slate-200 p-4">
        <legend className="px-1 text-sm font-medium text-slate-700">Bisnis</legend>

        {isLoadingBusinessMapping ? (
          <p className="text-sm text-slate-500">Memuat bisnis yang sudah dipilih…</p>
        ) : fields.length === 0 ? (
          <p className="text-sm text-slate-500">
            Belum ada bisnis yang dipilih. Cause of loss ini tetap dapat disimpan.
          </p>
        ) : (
          <ul className="space-y-2">
            {fields.map((row, index) => (
              <li key={row.id} className="flex items-end gap-2">
                <div className="grow">
                  {/* ComboField, bukan SelectField: isian Bisnis di Pega
                      ber-`pyAllowFreeFormInput=true`, sehingga nama di luar daftar
                      TETAP boleh diketik dan disimpan. */}
                  <ComboField
                    id={`bisnis-${index}`}
                    label={`Bisnis baris ${index + 1}`}
                    options={businessSuggestions}
                    maxLength={MAX_BUSINESS_NAME_LENGTH}
                    error={errors.bisnis?.[index]?.nama?.message}
                    {...register(`bisnis.${index}.nama` as const)}
                  />
                </div>
                <Button
                  tone="halus"
                  onClick={() => remove(index)}
                  aria-label={`Hapus bisnis baris ${index + 1}`}
                >
                  Hapus
                </Button>
              </li>
            ))}
          </ul>
        )}

        {/* Pelanggaran grid dilaporkan di bawah gridnya, bukan di salah satu barisnya:
            server menyebutnya sebagai satu isian bernama "bisnis", dan pesannya
            menyangkut susunan barisnya secara keseluruhan (nama kembar). */}
        {errors.bisnis?.message && (
          <p className="mt-2 text-sm text-red-600" role="alert">
            {errors.bisnis.message}
          </p>
        )}

        <div className="mt-3">
          <Button tone="kedua" onClick={() => append({ nama: '' })}>
            Tambah Bisnis
          </Button>
          {/* Daftar saran yang gagal dimuat TIDAK menghalangi apa pun: namanya memang
              boleh diketik sendiri. Yang hilang hanya kenyamanan memilih. */}
          {businessSuggestions.length === 0 && (
            <p className="mt-2 text-sm text-slate-500">
              Daftar bisnis belum dapat dimuat, jadi tidak ada saran. Nama bisnis tetap dapat
              diketik sendiri.
            </p>
          )}
        </div>
      </fieldset>

      <div className="flex flex-wrap justify-end gap-2 pt-2">
        <Button tone="halus" onClick={onCancel} disabled={isSaving}>
          Batal
        </Button>
        {/* Label tombolnya mengikuti layar Pega, yang memakai "Simpan" dan "Ubah" untuk
            kedua modusnya. */}
        <Button type="submit" tone="utama" disabled={isSaving || isLoadingBusinessMapping}>
          {isSaving ? 'Menyimpan…' : editMode ? 'Ubah' : 'Simpan'}
        </Button>
      </div>
    </form>
  )
}
