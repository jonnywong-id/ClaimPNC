import { zodResolver } from '@hookform/resolvers/zod'
import { useEffect } from 'react'
import { useForm } from 'react-hook-form'
import { z } from 'zod'

import { APIError, NetworkError } from '@/api/client'
import { AccountErrorCode } from '@/api/types'
import { Field } from '@/components/Field'
import { ErrorMessage, type ErrorTone } from '@/components/ErrorMessage'

import { useSubmitAccount, useBankList, type AccountFields } from './api'

/**
 * Aturan wajib di sini adalah CERMINAN aturan di server, bukan penggantinya.
 *
 * Server tetap memeriksa seluruhnya — validasi peramban hanya mempercepat umpan balik
 * dan dapat dilewati siapa pun dengan memanggil API langsung. Daftar kolomnya diambil
 * dari prasyarat activity CNMUpdateMasterRekening_act, sama dengan yang ditegakkan
 * masterrekening.Account.Check di backend.
 */
const schema = z.object({
  nomorRekening: z
    .string()
    .trim()
    .min(1, 'Nomor rekening wajib diisi.')
    .regex(/^[0-9-]+$/, 'Nomor rekening hanya boleh berisi angka.'),
  namaPemilik: z.string().trim().min(1, 'Nama pemilik rekening wajib diisi.'),
  namaBank: z.string().trim().min(1, 'Nama bank wajib diisi.'),
  cabangBank: z.string().trim().min(1, 'Nama cabang bank wajib diisi.'),
  alamatBank: z.string().trim().min(1, 'Alamat bank wajib diisi.'),
  kodeBank: z.string().trim().min(1, 'Bank wajib dipilih dari daftar.'),
  tipeRekening: z.string().trim().min(1, 'Tipe rekening wajib dipilih.'),
  email: z.string().trim().min(1, 'Email wajib diisi.').email('Format email tidak benar.'),
  telepon: z.string().trim(),
  nik: z.string().trim().min(1, 'NIK pemilik rekening wajib diisi.'),
  idDokumen: z.string().trim(),
  catatan: z.string().trim(),
  aktif: z.boolean(),
})

type FieldValues = z.infer<typeof schema>

/**
 * Tipe rekening yang dapat dipilih — isi kolom ACCOUNT_TYPE.
 *
 * Nilainya diambil apa adanya dari activity SetTipeRekening, yang mengisi daftar
 * pilihan layar lama:
 *
 *	TempTipeBank.pxResults(<APPEND>).Telephone = "BIASA"
 *	TempTipeBank.pxResults(<APPEND>).Telephone = "VA"
 *
 * "VA" berarti Virtual Account. Jadi tipe di sini membedakan BENTUK rekening, bukan
 * jenis pemiliknya — perbedaan yang mudah salah dibaca dari nama kolomnya saja.
 *
 * CATATAN. Daftarnya di sistem lama berasal dari kode, bukan tabel — persis bentuk
 * hardcode yang ADR-0025 perintahkan menjadi master. Ia ditaruh di satu konstanta
 * bernama supaya saat masternya tersedia, yang perlu diubah hanya satu tempat ini.
 */
const ACCOUNT_TYPES = [
  { value: 'BIASA', label: 'BIASA — rekening bank biasa' },
  { value: 'VA', label: 'VA — Virtual Account' },
] as const

type Props = {
  /** Dipanggil setelah pengajuan berhasil tersimpan. */
  onSuccess?: () => void
}

/** AccountForm adalah formulir pengajuan rekening baru. */
export function AccountForm({ onSuccess }: Props) {
  const submit = useSubmitAccount()
  const bank = useBankList()

  const {
    register,
    handleSubmit,
    reset,
    setError,
    formState: { errors },
  } = useForm<FieldValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      nomorRekening: '',
      namaPemilik: '',
      namaBank: '',
      cabangBank: '',
      alamatBank: '',
      kodeBank: '',
      tipeRekening: '',
      email: '',
      telepon: '',
      nik: '',
      idDokumen: '',
      catatan: '',
      aktif: true,
    },
  })

  // Galat validasi dari server dipindahkan ke kolomnya masing-masing. Tanpa langkah
  // ini, pengguna melihat satu kotak merah berisi daftar kolom dan harus mencocokkan
  // sendiri kalimat mana milik kolom mana.
  useEffect(() => {
    const error = submit.error
    if (!(error instanceof APIError) || error.kode !== AccountErrorCode.invalidInput) return

<<<<<<< HEAD:claim-pnc/frontend/src/modules/master-rekening/FormRekening.tsx
    for (const pelanggaran of galat.detail) {
      const kolom = PETA_KOLOM[pelanggaran.field]
      if (kolom) setError(kolom, { type: 'server', message: pelanggaran.pesan })
=======
    // violations() menyatukan kedua bentuk pelanggaran yang dipakai backend. Modul ini
    // mengirimkannya sebagai PETA `field` (internal/masterrekening/http/dto.go:151),
    // bukan sebagai senarai `detail` seperti kedua modul master lainnya.
    for (const [field, pesan] of Object.entries(error.violations())) {
      const column = COLUMN_MAP[field]
      if (column) setError(column, { type: 'server', message: pesan })
>>>>>>> origin/master:claim-pnc/frontend/src/modules/master-rekening/AccountForm.tsx
    }
  }, [submit.error, setError])

  function send(values: FieldValues) {
    submit.mutate(values as AccountFields, {
      onSuccess: () => {
        reset()
        onSuccess?.()
      },
    })
  }

  return (
    <form onSubmit={handleSubmit(send)} className="space-y-4" noValidate>
      {submit.isError && <SubmitErrorMessage error={submit.error} />}

      {submit.isSuccess && (
        <div role="status" className="rounded border border-green-200 bg-green-50 p-3 text-sm text-green-800">
          <p className="font-medium">Rekening diajukan</p>
          <p className="mt-1">
            Rekening menunggu keputusan komite. Ia belum dapat dipakai membayar klaim
            sampai komite menyetujuinya.
          </p>
        </div>
      )}

      <div className="grid gap-4 sm:grid-cols-2">
        <Field
          id="nomorRekening"
          label="Nomor rekening"
          inputMode="numeric"
          error={errors.nomorRekening?.message}
          {...register('nomorRekening')}
        />
        <Field
          id="namaPemilik"
          label="Nama pemilik rekening"
          error={errors.namaPemilik?.message}
          {...register('namaPemilik')}
        />

        <div>
          <label htmlFor="kodeBank" className="block text-sm font-medium text-slate-700">
            Bank
          </label>
          <select
            id="kodeBank"
            aria-invalid={errors.kodeBank ? 'true' : 'false'}
            className={
              'mt-1 w-full rounded border px-3 py-2 text-slate-900 focus:outline-none ' +
              (errors.kodeBank
                ? 'border-red-400 focus:border-red-500'
                : 'border-slate-300 focus:border-slate-500')
            }
            {...register('kodeBank')}
          >
            <option value="">— pilih bank —</option>
            {bank.data?.bank.map((b) => (
              <option key={b.kode} value={b.kode}>
                {b.nama}
              </option>
            ))}
          </select>
          {bank.isError && (
            <p className="mt-1 text-sm text-amber-800">
              Daftar bank tidak dapat dimuat. Muat ulang halaman, lalu coba lagi.
            </p>
          )}
          {errors.kodeBank && (
            <p className="mt-1 text-sm text-red-700">{errors.kodeBank.message}</p>
          )}
        </div>

        <Field
          id="namaBank"
          label="Nama bank"
          error={errors.namaBank?.message}
          {...register('namaBank')}
        />
        <Field
          id="cabangBank"
          label="Cabang bank"
          error={errors.cabangBank?.message}
          {...register('cabangBank')}
        />
        <Field
          id="alamatBank"
          label="Alamat bank"
          error={errors.alamatBank?.message}
          {...register('alamatBank')}
        />

        <div>
          <label htmlFor="tipeRekening" className="block text-sm font-medium text-slate-700">
            Tipe rekening
          </label>
          <select
            id="tipeRekening"
            aria-invalid={errors.tipeRekening ? 'true' : 'false'}
            className={
              'mt-1 w-full rounded border px-3 py-2 text-slate-900 focus:outline-none ' +
              (errors.tipeRekening
                ? 'border-red-400 focus:border-red-500'
                : 'border-slate-300 focus:border-slate-500')
            }
            {...register('tipeRekening')}
          >
            <option value="">— pilih tipe —</option>
            {ACCOUNT_TYPES.map((t) => (
              <option key={t.value} value={t.value}>
                {t.label}
              </option>
            ))}
          </select>
          {errors.tipeRekening && (
            <p className="mt-1 text-sm text-red-700">{errors.tipeRekening.message}</p>
          )}
        </div>

        <Field
          id="nik"
          label="NIK pemilik rekening"
          inputMode="numeric"
          error={errors.nik?.message}
          {...register('nik')}
        />
        <Field
          id="email"
          label="Email"
          type="email"
          error={errors.email?.message}
          {...register('email')}
        />
        <Field
          id="telepon"
          label="Telepon (opsional)"
          inputMode="tel"
          error={errors.telepon?.message}
          {...register('telepon')}
        />
        <Field
          id="idDokumen"
          label="ID dokumen buku rekening"
          error={errors.idDokumen?.message}
          {...register('idDokumen')}
        />
      </div>

      <div>
        <label htmlFor="catatan" className="block text-sm font-medium text-slate-700">
          Catatan
        </label>
        <textarea
          id="catatan"
          rows={3}
          className="mt-1 w-full rounded border border-slate-300 px-3 py-2 text-slate-900 focus:border-slate-500 focus:outline-none"
          {...register('catatan')}
        />
      </div>

      <label className="flex items-center gap-2 text-sm text-slate-700">
        <input type="checkbox" className="rounded border-slate-300" {...register('aktif')} />
        Account aktif
      </label>

      <p className="text-sm text-slate-600">
        Buku rekening wajib diunggah dan description approval atasan wajib diisi before
        komite dapat menyetujui rekening ini.
      </p>

      <button
        type="submit"
        disabled={submit.isPending}
        className="rounded bg-slate-900 px-4 py-2 text-sm font-medium text-white hover:bg-slate-700 disabled:cursor-not-allowed disabled:opacity-60"
      >
        {submit.isPending ? 'Menyimpan…' : 'Ajukan rekening'}
      </button>
    </form>
  )
}

/** PETA_KOLOM memetakan nama field server ke nama field formulir. */
const COLUMN_MAP: Record<string, keyof FieldValues> = {
  nomor_rekening: 'nomorRekening',
  nama_pemilik: 'namaPemilik',
  nama_bank: 'namaBank',
  cabang_bank: 'cabangBank',
  alamat_bank: 'alamatBank',
  kode_bank: 'kodeBank',
  tipe_rekening: 'tipeRekening',
  email: 'email',
  nik: 'nik',
}

function SubmitErrorMessage({ error }: { error: unknown }) {
  const content = messageFor(error)
  return <ErrorMessage title={content.title} description={content.description} tone={content.tone} />
}

function messageFor(error: unknown): { title: string; description: string; tone: ErrorTone } {
  if (error instanceof NetworkError) {
    return {
      title: 'Server Claim PNC tidak dapat dihubungi',
      description: 'Periksa koneksi jaringan Anda, lalu coba lagi.',
      tone: 'gangguan',
    }
  }
  if (error instanceof APIError) {
    switch (error.kode) {
      case AccountErrorCode.alreadyExists:
        return {
          title: 'Nomor rekening sudah terdaftar',
          description:
            'Nomor ini sudah ada dan belum ditolak komite. Gunakan data yang sudah ada, atau tunggu keputusan komite atas pengajuan sebelumnya.',
          tone: 'penolakan',
        }
      case AccountErrorCode.invalidInput:
        return {
          title: 'Ada isian yang belum benar',
          description: 'Periksa kolom yang ditandai di bawah, lalu kirim ulang.',
          tone: 'penolakan',
        }
      default:
        return {
          title: 'Terjadi kesalahan pada sistem',
          description: 'Coba beberapa saat lagi. Bila berulang, hubungi administrator Claim PNC.',
          tone: 'gangguan',
        }
    }
  }
  return {
    title: 'Terjadi kesalahan pada sistem',
    description: 'Coba beberapa saat lagi.',
    tone: 'gangguan',
  }
}
