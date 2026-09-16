import { zodResolver } from '@hookform/resolvers/zod'
import { useForm } from 'react-hook-form'
import { useLocation, useNavigate, type Location } from 'react-router-dom'
import { z } from 'zod'

import { GalatAPI, GalatJaringan } from '@/api/klien'
import { KodeGalat } from '@/api/tipe'
import { KolomIsian } from '@/components/KolomIsian'
import { PesanGalat, type NadaGalat } from '@/components/PesanGalat'

import { gunakanMasuk } from './api'

const skema = z.object({
  namaPengguna: z.string().trim().min(1, 'Nama pengguna wajib diisi.'),
  kataSandi: z.string().min(1, 'Kata sandi wajib diisi.'),
})

type Isian = z.infer<typeof skema>

type IsiPesan = {
  judul: string
  keterangan: string
  nada: NadaGalat
}

/**
 * Mengubah galat menjadi pesan yang dapat ditindaklanjuti.
 *
 * Ketiga jenis galat masuk sengaja diberi pesan BERBEDA, karena tindak lanjutnya
 * berbeda: mengetik ulang, menghubungi administrator, atau menunggu. Yang TIDAK
 * dibedakan adalah "pengguna tidak ada" dan "kata sandi salah" — keduanya memakai satu
 * pesan yang sama supaya keberadaan akun tidak bocor.
 */
function pesanUntuk(galat: unknown): IsiPesan {
  if (galat instanceof GalatJaringan) {
    return {
      judul: 'Server Claim PNC tidak dapat dihubungi',
      keterangan: 'Periksa koneksi jaringan Anda, lalu coba lagi.',
      nada: 'gangguan',
    }
  }
  if (galat instanceof GalatAPI) {
    switch (galat.kode) {
      case KodeGalat.kredensialSalah:
        return {
          judul: 'Nama pengguna atau kata sandi salah',
          keterangan: 'Periksa kembali isian Anda, lalu coba masuk lagi.',
          nada: 'penolakan',
        }
      case KodeGalat.penggunaTidakAktif:
        return {
          judul: 'Akun Anda tidak aktif',
          keterangan:
            'Mengetik ulang tidak akan menolong. Hubungi administrator Claim PNC untuk mengaktifkan kembali akun Anda.',
          nada: 'penolakan',
        }
      case KodeGalat.identitasPutus:
        return {
          judul: 'Sistem identitas sedang tidak dapat dihubungi',
          keterangan:
            'Ini bukan kesalahan Anda. Tunggu beberapa saat, lalu coba lagi — mencoba berulang kali tidak mempercepat pemulihan.',
          nada: 'gangguan',
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

type KeadaanRute = { dari?: Location }

export function HalamanMasuk() {
  const navigasi = useNavigate()
  const lokasi = useLocation()
  const masuk = gunakanMasuk()

  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<Isian>({
    resolver: zodResolver(skema),
    defaultValues: { namaPengguna: '', kataSandi: '' },
  })

  // Setelah berhasil masuk, pengguna kembali ke halaman yang tadi ia tuju — bukan
  // selalu ke beranda. Ini yang membuat sesi habis di tengah pekerjaan tidak terasa
  // seperti kehilangan tempat.
  const tujuan = (lokasi.state as KeadaanRute | null)?.dari?.pathname ?? '/'

  const kirim = handleSubmit((isian) => {
    masuk.mutate(isian, { onSuccess: () => navigasi(tujuan, { replace: true }) })
  })

  const galat = masuk.isError ? pesanUntuk(masuk.error) : null

  return (
    <main className="flex min-h-screen items-center justify-center bg-slate-100 px-4 py-10">
      <div className="w-full max-w-md">
        <header className="mb-6 text-center">
          <h1 className="text-2xl font-semibold text-slate-900">Claim PNC</h1>
          <p className="mt-1 text-sm text-slate-600">Masuk dengan akun kantor Anda.</p>
        </header>

        <form
          onSubmit={kirim}
          noValidate
          className="space-y-5 rounded-lg border border-slate-200 bg-white p-6 shadow-sm"
        >
          {galat && (
            <PesanGalat judul={galat.judul} keterangan={galat.keterangan} nada={galat.nada} />
          )}

          <KolomIsian
            id="namaPengguna"
            label="Nama pengguna"
            type="text"
            autoComplete="username"
            autoFocus
            galat={errors.namaPengguna?.message}
            {...register('namaPengguna')}
          />

          <KolomIsian
            id="kataSandi"
            label="Kata sandi"
            type="password"
            autoComplete="current-password"
            galat={errors.kataSandi?.message}
            {...register('kataSandi')}
          />

          <button
            type="submit"
            disabled={masuk.isPending}
            className="w-full rounded bg-slate-900 px-4 py-2 font-medium text-white hover:bg-slate-800 disabled:cursor-not-allowed disabled:bg-slate-400"
          >
            {masuk.isPending ? 'Memeriksa…' : 'Masuk'}
          </button>
        </form>

        <p className="mt-4 text-center text-xs text-slate-500">
          Kesulitan masuk? Hubungi administrator Claim PNC.
        </p>
      </div>
    </main>
  )
}
