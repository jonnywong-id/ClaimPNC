import { zodResolver } from '@hookform/resolvers/zod'
import { useForm } from 'react-hook-form'
import { useLocation, useNavigate, type Location } from 'react-router-dom'
import { z } from 'zod'

import { GalatAPI, GalatJaringan } from '@/api/klien'
import { KodeGalat } from '@/api/tipe'
import { IkonKunci, IkonPerisai } from '@/components/Ikon'
import { KolomIsian } from '@/components/KolomIsian'
import { PesanGalat, type NadaGalat } from '@/components/PesanGalat'
import { Tombol } from '@/components/Tombol'

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
    /*
      Dua panel pada layar lebar, satu kolom pada layar sempit.

      Panel kiri bukan hiasan: ia yang memberi tahu pengguna bahwa ia berada di sistem
      yang benar sebelum mengetikkan kata sandinya — pengenalan yang pada layar masuk
      justru bagian dari keamanan, bukan estetika. Pada layar sempit panel itu menyusut
      menjadi lambang dan satu baris judul, karena ruang yang ada lebih berguna untuk
      isian daripada untuk pesan merek.
    */
    <main className="flex min-h-screen flex-col lg:flex-row">
      <PanelMerek />

      <div className="flex flex-1 items-center justify-center bg-slate-50 px-4 py-10 sm:px-6">
        <div className="w-full max-w-md">
          <header className="mb-7">
            <h1 className="text-2xl font-semibold tracking-tight text-slate-900">
              Masuk ke Claim PNC
            </h1>
            <p className="mt-1.5 text-sm text-slate-600">
              Gunakan akun kantor Anda. Broker dan surveyor independen memakai akun yang
              diberikan administrator.
            </p>
          </header>

          <form
            onSubmit={kirim}
            noValidate
            className="space-y-5 rounded-kartu border border-slate-200 bg-white p-6 shadow-angkat sm:p-7"
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
              placeholder="NIK atau ID login"
              ikon={<IkonPerisai className="h-4 w-4" />}
              galat={errors.namaPengguna?.message}
              {...register('namaPengguna')}
            />

            <KolomIsian
              id="kataSandi"
              label="Kata sandi"
              type="password"
              autoComplete="current-password"
              placeholder="••••••••"
              ikon={<IkonKunci className="h-4 w-4" />}
              galat={errors.kataSandi?.message}
              {...register('kataSandi')}
            />

            <Tombol type="submit" nada="utama" disabled={masuk.isPending} className="w-full !py-2.5">
              {masuk.isPending ? (
                <>
                  <Pemutar />
                  Memeriksa…
                </>
              ) : (
                'Masuk'
              )}
            </Tombol>
          </form>

          <p className="mt-5 text-center text-xs text-slate-500">
            Kesulitan masuk? Hubungi administrator Claim PNC.
          </p>
        </div>
      </div>
    </main>
  )
}

/**
 * Panel merek di sisi kiri.
 *
 * Lingkaran-lingkaran kabur di latarnya digambar dengan `div` ber-blur, bukan gambar:
 * tidak ada berkas yang perlu diunduh, dan aplikasi ini berjalan di jaringan tertutup
 * tanpa jaminan akses internet.
 */
function PanelMerek() {
  return (
    <aside className="relative isolate overflow-hidden bg-gradient-to-br from-blue-600 via-blue-700 to-blue-900 px-6 py-8 text-white lg:w-[42%] lg:px-12 lg:py-16">
      <div
        aria-hidden="true"
        className="pointer-events-none absolute -right-24 -top-24 h-72 w-72 rounded-full bg-blue-400/30 blur-3xl"
      />
      <div
        aria-hidden="true"
        className="pointer-events-none absolute -bottom-32 -left-20 h-80 w-80 rounded-full bg-sky-400/20 blur-3xl"
      />

      <div className="relative flex h-full flex-col justify-between gap-10">
        <div className="flex items-center gap-3">
          <span className="flex h-11 w-11 shrink-0 items-center justify-center rounded-kontrol bg-white/15 ring-1 ring-white/25 backdrop-blur">
            <IkonPerisai className="h-6 w-6" />
          </span>
          <div>
            <p className="text-sm font-semibold leading-tight">Claim PNC</p>
            <p className="text-xs leading-tight text-blue-200">Asuransi Sinar Mas</p>
          </div>
        </div>

        <div className="hidden lg:block">
          <h2 className="text-3xl font-semibold leading-tight tracking-tight">
            Penanganan klaim
            <br />
            non-motor, satu tempat.
          </h2>
          <p className="mt-4 max-w-sm text-sm leading-relaxed text-blue-100">
            Dari laporan kerugian sampai pembayaran ganti rugi — registrasi, survei,
            komite, akseptasi, dan pemberitahuan reasuransi.
          </p>
        </div>

        <p className="hidden text-xs text-blue-200/80 lg:block">
          Akses terbatas pada pengguna terdaftar. Setiap perubahan bernilai bisnis tercatat.
        </p>
      </div>
    </aside>
  )
}

/** Pemutar kecil pada tombol yang sedang bekerja. */
function Pemutar() {
  return (
    <svg viewBox="0 0 16 16" aria-hidden="true" className="h-4 w-4 animate-spin">
      <circle cx="8" cy="8" r="6.5" fill="none" stroke="currentColor" strokeOpacity="0.3" strokeWidth="2" />
      <path d="M8 1.5a6.5 6.5 0 0 1 6.5 6.5" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" />
    </svg>
  )
}
