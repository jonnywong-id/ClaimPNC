import { zodResolver } from '@hookform/resolvers/zod'
import { useEffect, useState } from 'react'
import { useForm } from 'react-hook-form'
import { z } from 'zod'

import { APIError, NetworkError } from '@/api/client'
import { AutoClaimErrorCode, AutoClaimStatus, ErrorCode, type AutoClaim } from '@/api/types'
import { Button } from '@/components/Button'
import { ErrorMessage, type ErrorTone } from '@/components/ErrorMessage'
import { Field } from '@/components/Field'
import { SelectField } from '@/components/SelectField'

import { useAutoClaimBankList, useBusinessSourceLookup, useClientLookup } from './api'
import { LookupPicker, type LookupRow } from './LookupPicker'

/**
 * Batas panjang harus sama dengan konstanta di `internal/masterautoclaim/masterautoclaim.go`.
 *
 * SELURUHNYA ASUMSI YANG DISADARI: DDL POOLDATA.M_AUTO_CLAIM_PNC belum diterima (R-08),
 * dan sistem lama tidak memeriksa panjang satu pun isian. Batasnya dipasang supaya
 * penolakan datang sebagai kalimat yang menuntun, bukan sebagai ORA-12899.
 *
 * Bila salah satu berubah, KEDUA tempat harus ikut berubah.
 */
const MAX = {
  namaBank: 100,
  noRekening: 30,
  picLapor: 100,
  emailLapor: 100,
  alamatPenerima: 250,
} as const

/**
 * PCT max diperiksa sebagai angka 0–100, dengan koma MAUPUN titik sebagai pemisah
 * desimal — petugas Indonesia mengetik "82,5" sementara basis data menyimpan "82.5".
 *
 * Ini SELISIH YANG DIRENCANAKAN terhadap Pega, yang menerima teks apa pun. Server tetap
 * yang berwenang; pemeriksaan di layar hanya kenyamanan.
 */
const percentSchema = z
  .string()
  .trim()
  .min(1, 'PCT max wajib diisi.')
  .refine((value) => {
    const parsed = Number(value.replace(',', '.'))
    return Number.isFinite(parsed) && parsed >= 0 && parsed <= 100
  }, 'PCT max harus berupa angka antara 0 dan 100, misalnya 100 atau 82,5.')

const schema = z.object({
  nama_bank: z
    .string()
    .trim()
    .min(1, 'Bank penerima wajib dipilih.')
    .max(MAX.namaBank, `Bank penerima paling panjang ${MAX.namaBank} karakter.`),
  no_rekening: z
    .string()
    .trim()
    .min(1, 'Nomor rekening wajib diisi.')
    .max(MAX.noRekening, `Nomor rekening paling panjang ${MAX.noRekening} karakter.`),
  pct_max: percentSchema,
  pic_lapor: z
    .string()
    .trim()
    .min(1, 'PIC lapor wajib diisi.')
    .max(MAX.picLapor, `PIC lapor paling panjang ${MAX.picLapor} karakter.`),
  email_lapor: z
    .string()
    .trim()
    .min(1, 'Email lapor wajib diisi.')
    .max(MAX.emailLapor, `Email lapor paling panjang ${MAX.emailLapor} karakter.`),
  alamat_penerima: z
    .string()
    .trim()
    .min(1, 'Alamat penerima wajib diisi.')
    .max(MAX.alamatPenerima, `Alamat penerima paling panjang ${MAX.alamatPenerima} karakter.`),
})

export type AutoClaimFields = z.infer<typeof schema>

/** Nilai yang dikirim ke pemanggil saat form disimpan. */
export type AutoClaimFormValues = AutoClaimFields & {
  /** Hanya terisi pada mode tambah; pada mode ubah ia null. */
  sumberBisnis: LookupRow | null
  client: LookupRow | null
}

type Props = {
  /**
   * Baris yang sedang disunting, atau null bila menambah.
   *
   * Mode menentukan dua hal sekaligus: Sumber Bisnis dapat dipilih atau tidak, dan
   * teks tombol simpannya.
   */
  editing: AutoClaim | null

  /**
   * Tombol Simpan ditampilkan. Ia mengirim status `"0"` — perubahan mengembalikan baris
   * ke antrean persetujuan, persis seperti `stsapprove="0"` pada layar lama.
   */
  canSave: boolean

  /**
   * Tombol Approve dan Reject ditampilkan. Keduanya hanya ada di tab Komite Approval,
   * dan di Pega pun keduanya berada DI FORM — bukan di baris grid.
   */
  canDecide: boolean

  isSaving: boolean
  error: unknown
  onSave: (values: AutoClaimFormValues, status: string) => void
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
      case AutoClaimErrorCode.initialTaken:
        return {
          title: 'Sumber bisnis ini sudah ada di Master Auto Claim',
          description:
            'Buka barisnya lewat tab yang sesuai untuk mengubahnya, jangan menambahkannya lagi.',
          tone: 'penolakan',
        }
      case ErrorCode.validationFailed:
        // Bila detailnya ada, isiannya sudah disorot satu per satu; kotak pesan hanya
        // akan mengulang hal yang sama.
        return Object.keys(error.violations()).length > 0
          ? null
          : { title: 'Belum dapat disimpan', description: error.message, tone: 'penolakan' }
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

const FIELD_NAMES = [
  'nama_bank',
  'no_rekening',
  'pct_max',
  'pic_lapor',
  'email_lapor',
  'alamat_penerima',
] as const

/**
 * AutoClaimForm adalah form tambah dan ubah Master Auto Claim.
 *
 * Susunan isiannya mengikuti `Section/BrowseAutoKlaim-Section.xml` apa adanya (`D-13`:
 * alur dan tata letak ditiru supaya pengguna tidak perlu belajar ulang).
 *
 * # Tiga isian layar lama yang TIDAK ada di sini, dan kenapa
 *
 * **CLAIM ALLOWED.** Layar lama punya kotaknya, tetapi isinya dibuang: activity
 * menimpanya dengan "1" pada setiap penambahan. Keputusan Work Owner 2026-09-19
 * menjadikannya selalu "1" dan tidak dapat diubah, sehingga kotaknya dicabut — kotak
 * yang isinya selalu diabaikan lebih buruk daripada tidak ada kotak sama sekali.
 *
 * **KOMITE.** Diisi sistem dari POOLDATA.EMAILKOMITE, tidak pernah diketik.
 *
 * **APPROVAL.** Bukan isian melainkan akibat tombol yang ditekan — lihat AutoClaimPage.
 *
 * # Sumber Bisnis dikunci saat mengubah
 *
 * Ia kunci baris, dan `UpdateAutoClaim-SQL.xml` tidak pernah memindahkannya. Nama
 * penerima pun tidak dapat diubah, karena kueri lama tidak menyebut kolomnya —
 * keputusan Work Owner 2026-09-19. Salah pilih sumber bisnis diperbaiki dengan menolak
 * barisnya lalu menambah yang baru, bukan dengan menyuntingnya.
 */
export function AutoClaimForm({
  editing,
  canSave,
  canDecide,
  isSaving,
  error,
  onSave,
  onCancel,
}: Props) {
  const isEditing = editing !== null

  // Tab Waiting Approval tidak punya satu pun tombol simpan di Pega — ia baca saja.
  // Isiannya ikut dinonaktifkan supaya keadaan itu terlihat, bukan hanya terasa saat
  // pengguna mencari tombol yang tidak ada.
  const isReadOnly = !canSave && !canDecide

  const [sumberBisnis, setSumberBisnis] = useState<LookupRow | null>(null)
  const [client, setClient] = useState<LookupRow | null>(
    editing && editing.id_client !== ''
      ? { id: editing.id_client, nama: editing.nama_client }
      : null,
  )
  const [sourceKeyword, setSourceKeyword] = useState('')
  const [clientKeyword, setClientKeyword] = useState('')

  const sourceLookup = useBusinessSourceLookup(sourceKeyword)
  const clientLookup = useClientLookup(clientKeyword)
  const bankList = useAutoClaimBankList()

  const {
    register,
    handleSubmit,
    setError,
    formState: { errors },
  } = useForm<AutoClaimFields>({
    resolver: zodResolver(schema),
    defaultValues: {
      nama_bank: editing?.nama_bank ?? '',
      no_rekening: editing?.no_rekening ?? '',
      pct_max: editing?.pct_max ?? '',
      pic_lapor: editing?.pic_lapor ?? '',
      email_lapor: editing?.email_lapor ?? '',
      alamat_penerima: editing?.alamat_penerima ?? '',
    },
  })

  // Pelanggaran yang dilaporkan server disorot pada isiannya masing-masing, bukan hanya
  // diringkas di satu kotak pesan. Server mengirim SELURUH pelanggaran sekaligus (P-5),
  // dan itu hanya berguna bila layar menyorotnya satu per satu.
  useEffect(() => {
    for (const [column, message] of Object.entries(violationsOf(error))) {
      if ((FIELD_NAMES as readonly string[]).includes(column)) {
        setError(column as (typeof FIELD_NAMES)[number], { type: 'server', message })
      }
    }
  }, [error, setError])

  const violation = violationsOf(error)
  const message = messageFor(error)

  /**
   * submitWith menyusun penangan untuk satu tombol beserta status yang dikirimnya.
   *
   * `handleSubmit` mengembalikan penangan yang menjalankan pemeriksaan isian LEBIH DULU,
   * lalu memanggil callback-nya. Dengan begitu Approve pun tidak dapat mengirim isian
   * yang belum lolos pemeriksaan — bukan hanya tombol Simpan.
   */
  const submitWith = (status: string) =>
    handleSubmit((values) => onSave({ ...values, sumberBisnis, client }, status))

  // Isian dinonaktifkan saat menyimpan DAN saat baca saja. Keduanya disatukan supaya
  // tidak ada satu isian pun yang lupa menyertakan salah satunya.
  const isLocked = isSaving || isReadOnly

  return (
    <form
      onSubmit={submitWith(AutoClaimStatus.menunggu)}
      noValidate
      aria-label={isEditing ? 'Ubah Master Auto Claim' : 'Tambah Master Auto Claim'}
      className="space-y-4 rounded-kartu border border-slate-200 bg-white p-5 shadow-lembut"
    >
      <h2 className="text-base font-semibold text-slate-900">
        {isEditing ? `Ubah ${editing.inisial}` : 'Tambah Master Auto Claim'}
      </h2>

      {message && (
        <ErrorMessage title={message.title} description={message.description} tone={message.tone} />
      )}

      {isEditing ? (
        /*
          Sumber Bisnis ditampilkan sebagai keterangan, bukan isian. Ia kunci baris, dan
          nama penerimanya pun tidak dapat diperbarui — kueri UPDATE sistem lama tidak
          menyebut kolomnya.
        */
        <div className="rounded-kontrol border border-slate-200 bg-slate-50 px-3 py-2.5">
          <span className="block text-xs font-medium uppercase tracking-wide text-slate-500">
            Sumber Bisnis
          </span>
          <span className="mt-0.5 block text-sm text-slate-900">
            {editing.nama_penerima}
            <span className="ml-2 text-xs text-slate-500">{editing.inisial}</span>
          </span>
          <span className="mt-1 block text-xs text-slate-500">
            Tidak dapat diubah. Sumber bisnis yang keliru diperbaiki dengan menolak baris ini,
            lalu menambah yang baru.
          </span>
        </div>
      ) : (
        <LookupPicker
          label="Sumber Bisnis"
          selected={sumberBisnis}
          rows={sourceLookup.data?.sumber_bisnis ?? []}
          isSearching={sourceLookup.isFetching}
          isError={sourceLookup.isError}
          keyword={sourceKeyword}
          onKeywordChange={setSourceKeyword}
          onPick={setSumberBisnis}
          onClear={() => setSumberBisnis(null)}
          error={violation['inisial'] ?? violation['nama_penerima']}
          hint="Kode dan namanya diambil dari master Sumber Bisnis; keduanya tidak dapat diketik."
          disabled={isLocked}
        />
      )}

      <SelectField
        id="nama_bank"
        label="Bank penerima"
        options={(bankList.data?.bank ?? []).map((b) => ({
          // Yang disimpan adalah NAMANYA — tabel ini tidak punya kolom kode bank.
          // Kodenya ikut ditampilkan supaya dua bank bernama mirip dapat dibedakan.
          value: b.nama,
          label: `${b.nama} (${b.kode})`,
        }))}
        error={errors.nama_bank?.message}
        disabled={isLocked}
        {...register('nama_bank')}
      />

      <div className="grid gap-4 sm:grid-cols-2">
        <Field
          id="no_rekening"
          label="Nomor rekening"
          type="text"
          inputMode="numeric"
          maxLength={MAX.noRekening}
          error={errors.no_rekening?.message}
          disabled={isLocked}
          {...register('no_rekening')}
        />
        <Field
          id="pct_max"
          label="PCT max"
          type="text"
          inputMode="decimal"
          hint="Persentase maksimum, 0–100."
          error={errors.pct_max?.message}
          disabled={isLocked}
          {...register('pct_max')}
        />
      </div>

      <div className="grid gap-4 sm:grid-cols-2">
        <Field
          id="pic_lapor"
          label="PIC lapor"
          type="text"
          maxLength={MAX.picLapor}
          error={errors.pic_lapor?.message}
          disabled={isLocked}
          {...register('pic_lapor')}
        />
        <Field
          id="email_lapor"
          label="Email lapor"
          type="email"
          maxLength={MAX.emailLapor}
          error={errors.email_lapor?.message}
          disabled={isLocked}
          {...register('email_lapor')}
        />
      </div>

      <Field
        id="alamat_penerima"
        label="Alamat penerima"
        type="text"
        maxLength={MAX.alamatPenerima}
        error={errors.alamat_penerima?.message}
        disabled={isLocked}
        {...register('alamat_penerima')}
      />

      {/*
        Client boleh kosong — sistem lama pun menyisipkannya kosong, dan tidak ada satu
        pun prasyarat yang mewajibkannya. Yang tidak boleh adalah setengah terisi, dan
        itu mustahil di sini karena ID dan nama selalu dipilih berpasangan.
      */}
      <LookupPicker
        label="Client"
        selected={client}
        rows={clientLookup.data?.client ?? []}
        isSearching={clientLookup.isFetching}
        isError={clientLookup.isError}
        keyword={clientKeyword}
        onKeywordChange={setClientKeyword}
        onPick={setClient}
        onClear={() => setClient(null)}
        error={violation['id_client'] ?? violation['nama_client']}
        hint="Boleh dikosongkan. Bila diisi, pilih dari master Client."
        disabled={isLocked}
      />

      {canSave && (
        <p className="text-xs text-slate-500">
          Baris yang disimpan selalu kembali ke{' '}
          <span className="font-medium">Waiting Approval</span> dan menunggu keputusan komite —
          persetujuan sebelumnya tidak berlaku atas isi yang sudah berubah.
        </p>
      )}

      {isReadOnly && (
        <p className="text-xs text-slate-500">
          Baca saja. Baris yang menunggu hanya dapat diputuskan komite yang ditunjuk —
          lihat kolom <span className="font-medium">KOMITE</span> — lewat tab Komite Approval.
        </p>
      )}

      <div className="flex flex-wrap justify-end gap-2 pt-2">
        <Button tone="halus" onClick={onCancel} disabled={isSaving}>
          {isReadOnly ? 'Tutup' : 'Batal'}
        </Button>

        {canSave && (
          <Button
            type="submit"
            tone="utama"
            disabled={isSaving || (!isEditing && sumberBisnis === null)}
          >
            {isSaving ? 'Menyimpan…' : 'Simpan'}
          </Button>
        )}

        {/*
          Approve dan Reject berada DI FORM, bukan di baris grid — itu letaknya di Pega,
          dan letak itu yang membuat alurnya masuk akal: komite menekan Update pada
          barisnya, isian termuat, lalu ia memutuskan atas isi yang benar-benar dilihatnya.

          Keduanya `type="button"` dengan penangan sendiri, bukan submit: masing-masing
          mengirim status yang berbeda, sementara sebuah form hanya punya satu onSubmit.
          Pemeriksaan isian tetap berjalan — lihat submitWith.
        */}
        {canDecide && (
          <>
            <Button
              tone="kedua"
              onClick={submitWith(AutoClaimStatus.ditolak)}
              disabled={isSaving}
            >
              Reject
            </Button>
            <Button
              tone="utama"
              onClick={submitWith(AutoClaimStatus.disetujui)}
              disabled={isSaving}
            >
              Approve
            </Button>
          </>
        )}
      </div>
    </form>
  )
}
