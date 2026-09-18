import { zodResolver } from '@hookform/resolvers/zod'
import { useEffect } from 'react'
import { useForm } from 'react-hook-form'
import { z } from 'zod'

import { GalatAPI, GalatJaringan } from '@/api/klien'
import { KodeGalatRekening } from '@/api/tipe'
import { KolomIsian } from '@/components/KolomIsian'
import { PesanGalat, type NadaGalat } from '@/components/PesanGalat'

import { gunakanAjukanRekening, gunakanDaftarBank, type IsianRekening } from './api'

/**
 * Aturan wajib di sini adalah CERMINAN aturan di server, bukan penggantinya.
 *
 * Server tetap memeriksa seluruhnya — validasi peramban hanya mempercepat umpan balik
 * dan dapat dilewati siapa pun dengan memanggil API langsung. Daftar kolomnya diambil
 * dari prasyarat activity CNMUpdateMasterRekening_act, sama dengan yang ditegakkan
 * masterrekening.Rekening.Periksa di backend.
 */
const skema = z.object({
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

type Isian = z.infer<typeof skema>

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
const TIPE_REKENING = [
  { nilai: 'BIASA', label: 'BIASA — rekening bank biasa' },
  { nilai: 'VA', label: 'VA — Virtual Account' },
] as const

type Props = {
  /** Dipanggil setelah pengajuan berhasil tersimpan. */
  padaBerhasil?: () => void
}

/** FormRekening adalah formulir pengajuan rekening baru. */
export function FormRekening({ padaBerhasil }: Props) {
  const ajukan = gunakanAjukanRekening()
  const bank = gunakanDaftarBank()

  const {
    register,
    handleSubmit,
    reset,
    setError,
    formState: { errors },
  } = useForm<Isian>({
    resolver: zodResolver(skema),
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
    const galat = ajukan.error
    if (!(galat instanceof GalatAPI) || galat.kode !== KodeGalatRekening.isianTidakSah) return

    for (const pelanggaran of galat.detail) {
      const kolom = PETA_KOLOM[pelanggaran.field]
      if (kolom) setError(kolom, { type: 'server', message: pelanggaran.pesan })
    }
  }, [ajukan.error, setError])

  function kirim(isian: Isian) {
    ajukan.mutate(isian as IsianRekening, {
      onSuccess: () => {
        reset()
        padaBerhasil?.()
      },
    })
  }

  return (
    <form onSubmit={handleSubmit(kirim)} className="space-y-4" noValidate>
      {ajukan.isError && <PesanGalatPengajuan galat={ajukan.error} />}

      {ajukan.isSuccess && (
        <div role="status" className="rounded border border-green-200 bg-green-50 p-3 text-sm text-green-800">
          <p className="font-medium">Rekening diajukan</p>
          <p className="mt-1">
            Rekening menunggu keputusan komite. Ia belum dapat dipakai membayar klaim
            sampai komite menyetujuinya.
          </p>
        </div>
      )}

      <div className="grid gap-4 sm:grid-cols-2">
        <KolomIsian
          id="nomorRekening"
          label="Nomor rekening"
          inputMode="numeric"
          galat={errors.nomorRekening?.message}
          {...register('nomorRekening')}
        />
        <KolomIsian
          id="namaPemilik"
          label="Nama pemilik rekening"
          galat={errors.namaPemilik?.message}
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

        <KolomIsian
          id="namaBank"
          label="Nama bank"
          galat={errors.namaBank?.message}
          {...register('namaBank')}
        />
        <KolomIsian
          id="cabangBank"
          label="Cabang bank"
          galat={errors.cabangBank?.message}
          {...register('cabangBank')}
        />
        <KolomIsian
          id="alamatBank"
          label="Alamat bank"
          galat={errors.alamatBank?.message}
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
            {TIPE_REKENING.map((t) => (
              <option key={t.nilai} value={t.nilai}>
                {t.label}
              </option>
            ))}
          </select>
          {errors.tipeRekening && (
            <p className="mt-1 text-sm text-red-700">{errors.tipeRekening.message}</p>
          )}
        </div>

        <KolomIsian
          id="nik"
          label="NIK pemilik rekening"
          inputMode="numeric"
          galat={errors.nik?.message}
          {...register('nik')}
        />
        <KolomIsian
          id="email"
          label="Email"
          type="email"
          galat={errors.email?.message}
          {...register('email')}
        />
        <KolomIsian
          id="telepon"
          label="Telepon (opsional)"
          inputMode="tel"
          galat={errors.telepon?.message}
          {...register('telepon')}
        />
        <KolomIsian
          id="idDokumen"
          label="ID dokumen buku rekening"
          galat={errors.idDokumen?.message}
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
        Rekening aktif
      </label>

      <p className="text-sm text-slate-600">
        Buku rekening wajib diunggah dan keterangan approval atasan wajib diisi sebelum
        komite dapat menyetujui rekening ini.
      </p>

      <button
        type="submit"
        disabled={ajukan.isPending}
        className="rounded bg-slate-900 px-4 py-2 text-sm font-medium text-white hover:bg-slate-700 disabled:cursor-not-allowed disabled:opacity-60"
      >
        {ajukan.isPending ? 'Menyimpan…' : 'Ajukan rekening'}
      </button>
    </form>
  )
}

/** PETA_KOLOM memetakan nama field server ke nama field formulir. */
const PETA_KOLOM: Record<string, keyof Isian> = {
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

function PesanGalatPengajuan({ galat }: { galat: unknown }) {
  const isi = pesanUntuk(galat)
  return <PesanGalat judul={isi.judul} keterangan={isi.keterangan} nada={isi.nada} />
}

function pesanUntuk(galat: unknown): { judul: string; keterangan: string; nada: NadaGalat } {
  if (galat instanceof GalatJaringan) {
    return {
      judul: 'Server Claim PNC tidak dapat dihubungi',
      keterangan: 'Periksa koneksi jaringan Anda, lalu coba lagi.',
      nada: 'gangguan',
    }
  }
  if (galat instanceof GalatAPI) {
    switch (galat.kode) {
      case KodeGalatRekening.sudahAda:
        return {
          judul: 'Nomor rekening sudah terdaftar',
          keterangan:
            'Nomor ini sudah ada dan belum ditolak komite. Gunakan data yang sudah ada, atau tunggu keputusan komite atas pengajuan sebelumnya.',
          nada: 'penolakan',
        }
      case KodeGalatRekening.isianTidakSah:
        return {
          judul: 'Ada isian yang belum benar',
          keterangan: 'Periksa kolom yang ditandai di bawah, lalu kirim ulang.',
          nada: 'penolakan',
        }
      default:
        return {
          judul: 'Terjadi kesalahan pada sistem',
          keterangan: 'Coba beberapa saat lagi. Bila berulang, hubungi administrator Claim PNC.',
          nada: 'gangguan',
        }
    }
  }
  return {
    judul: 'Terjadi kesalahan pada sistem',
    keterangan: 'Coba beberapa saat lagi.',
    nada: 'gangguan',
  }
}
