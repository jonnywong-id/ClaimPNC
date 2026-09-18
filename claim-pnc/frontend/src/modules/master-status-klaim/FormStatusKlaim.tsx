import { zodResolver } from '@hookform/resolvers/zod'
import { useEffect, useRef } from 'react'
import { useForm } from 'react-hook-form'
import { z } from 'zod'

import { GalatAPI, GalatJaringan } from '@/api/klien'
import { KodeGalat, type StatusKlaim } from '@/api/tipe'
import { KolomIsian } from '@/components/KolomIsian'
import { PesanGalat, type NadaGalat } from '@/components/PesanGalat'
import { Tombol } from '@/components/Tombol'

import { gunakanSimpanStatusKlaim } from './api'

/** Batas panjang label; sama dengan masterstatus.PanjangLabelMaksimum di backend. */
const PANJANG_LABEL_MAKSIMUM = 100

/**
 * Aturan yang sama dinyatakan dua kali: di sini dan di domain Go.
 *
 * Itu duplikasi yang DISENGAJA, bukan kelalaian. Yang di sini menjawab pengguna tanpa
 * perjalanan jaringan; yang di sana adalah yang menegakkan — karena pemanggilan
 * langsung ke API tidak melewati layar ini sama sekali. Menghapus salah satunya berarti
 * memilih antara layar yang lamban atau API yang tidak terjaga.
 *
 * Keunikan label TIDAK diperiksa di sini: ia menuntut mengetahui seluruh label yang ada,
 * dan jawabannya dapat berubah antara saat layar dimuat dan saat Simpan ditekan.
 * Penegakannya ada di indeks unik basis data, dan galatnya ditampilkan di bawah.
 */
const skema = z.object({
  label: z
    .string()
    .trim()
    .min(1, 'Status wajib diisi.')
    .max(PANJANG_LABEL_MAKSIMUM, `Status paling panjang ${PANJANG_LABEL_MAKSIMUM} karakter.`),
})

type Isian = z.infer<typeof skema>

type Props = {
  /** Null berarti menambah; terisi berarti mengubah baris itu. */
  status: StatusKlaim | null
  tutup: () => void
}

/**
 * Form tambah dan ubah Master Status Klaim.
 *
 * Meniru fungsi form `ListStatusClaim` di Pega — kode read-only, satu isian Status, dan
 * tombol Simpan — dengan dua perbedaan yang disengaja: isiannya wajib diisi, dan nama
 * yang sudah dipakai ditolak (keputusan Work Owner 2026-09-17).
 */
export function FormStatusKlaim({ status, tutup }: Props) {
  const simpan = gunakanSimpanStatusKlaim()
  const sedangMengubah = status !== null
  const isianPertama = useRef<HTMLInputElement | null>(null)

  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<Isian>({
    resolver: zodResolver(skema),
    defaultValues: { label: status?.label ?? '' },
  })

  // Fokus dipindahkan ke isian pertama saat form terbuka. Tanpa ini, pengguna papan
  // ketik harus menekan Tab berkali-kali dari awal halaman untuk mencapainya.
  useEffect(() => {
    isianPertama.current?.focus()
  }, [])

  const { ref: refLabel, ...sisaLabel } = register('label')

  function kirim(isian: Isian) {
    simpan.mutate(
      sedangMengubah ? { kode: status.kode, label: isian.label } : { label: isian.label },
      { onSuccess: tutup },
    )
  }

  return (
    /*
      Panel ini muncul di atas tabel, bukan sebagai dialog melayang.

      Alasannya praktis: pengguna sering perlu melihat status lain yang sudah ada untuk
      memastikan nama yang diketiknya tidak bertabrakan — dan dialog yang menutup layar
      justru menyembunyikan jawabannya. Garis aksen di tepi kiri menandai bahwa panel ini
      keadaan sementara, bukan bagian tetap halaman.
    */
    <form
      onSubmit={handleSubmit(kirim)}
      noValidate
      className="overflow-hidden rounded-kartu border border-slate-200 border-l-4 border-l-blue-500 bg-white shadow-angkat"
      aria-label={sedangMengubah ? 'Ubah status klaim' : 'Tambah status klaim'}
    >
      <div className="border-b border-slate-100 bg-slate-50/70 px-5 py-4">
        <h3 className="text-base font-semibold text-slate-900">
          {sedangMengubah ? 'Ubah Status Klaim' : 'Tambah Status Klaim'}
        </h3>
        <p className="mt-1 text-sm text-slate-600">
          {sedangMengubah
            ? 'Hanya nama status yang dapat diubah. Kode tetap, karena klaim lama menyimpannya.'
            : 'Kode dibuat sistem setelah disimpan, melanjutkan nomor terakhir.'}
        </p>
      </div>

      <div className="space-y-5 p-5">
        {simpan.isError && <PesanGalatSimpan galat={simpan.error} />}

        <div className="grid gap-5 sm:grid-cols-2">
          <div>
            <span className="block text-sm font-medium text-slate-700">Kode</span>
            {/*
              Kode digambar sebagai kotak mati, bukan input ber-`disabled`. Input yang
              dinonaktifkan tetap terlihat seperti isian dan mengundang pengguna
              mengkliknya; kotak ini jelas bukan tempat mengetik.
            */}
            <p className="mt-1.5 flex items-center rounded-kontrol border border-dashed border-slate-300 bg-slate-50 px-3 py-2.5 font-mono text-sm text-slate-500">
              {status?.kode ?? 'Dibuat sistem'}
            </p>
            <p className="mt-1.5 text-xs text-slate-500">
              Kode tidak dapat disunting, sama seperti di sistem lama.
            </p>
          </div>

          <KolomIsian
            id="label"
            label="Status"
            placeholder="Contoh: Reopen Claim"
            maxLength={PANJANG_LABEL_MAKSIMUM}
            autoComplete="off"
            petunjuk={`Paling panjang ${PANJANG_LABEL_MAKSIMUM} karakter, dan belum dipakai status lain.`}
            galat={errors.label?.message}
            disabled={simpan.isPending}
            {...sisaLabel}
            ref={(elemen) => {
              refLabel(elemen)
              isianPertama.current = elemen
            }}
          />
        </div>

        <div className="flex flex-wrap gap-2 border-t border-slate-100 pt-5">
          <Tombol type="submit" nada="utama" disabled={simpan.isPending}>
            {simpan.isPending && <Pemutar />}
            {simpan.isPending ? 'Menyimpan…' : 'Simpan'}
          </Tombol>
          <Tombol nada="halus" onClick={tutup} disabled={simpan.isPending}>
            Batal
          </Tombol>
        </div>
      </div>
    </form>
  )
}

/** Pemutar kecil pada tombol yang sedang bekerja. */
function Pemutar() {
  return (
    <svg viewBox="0 0 16 16" aria-hidden="true" className="h-4 w-4 animate-spin">
      <circle cx="8" cy="8" r="6.5" fill="none" stroke="currentColor" strokeOpacity="0.3" strokeWidth="2" />
      <path
        d="M8 1.5a6.5 6.5 0 0 1 6.5 6.5"
        fill="none"
        stroke="currentColor"
        strokeWidth="2"
        strokeLinecap="round"
      />
    </svg>
  )
}

/**
 * Galat simpan dibedakan menurut KODE-nya, bukan teks pesannya.
 *
 * Tiga jenis menuntut tindak lanjut berbeda: nama yang bentrok dapat diperbaiki
 * pengguna, validasi server menunjuk kolom tertentu, dan gangguan sistem tidak dapat
 * ditolong dengan mencoba ulang berkali-kali.
 */
function PesanGalatSimpan({ galat }: { galat: unknown }) {
  if (galat instanceof GalatJaringan) {
    return (
      <PesanGalat
        judul="Tidak dapat menghubungi server"
        keterangan="Perubahan belum tersimpan. Periksa koneksi lalu coba lagi."
        nada="gangguan"
      />
    )
  }

  if (!(galat instanceof GalatAPI)) {
    return (
      <PesanGalat
        judul="Gagal menyimpan"
        keterangan="Terjadi kesalahan yang tidak terduga. Coba beberapa saat lagi."
        nada="gangguan"
      />
    )
  }

  const { judul, keterangan, nada } = uraikan(galat)
  return <PesanGalat judul={judul} keterangan={keterangan} nada={nada} />
}

function uraikan(galat: GalatAPI): { judul: string; keterangan: string; nada: NadaGalat } {
  switch (galat.kode) {
    case KodeGalat.labelStatusSudahDipakai:
      return {
        judul: 'Nama status sudah dipakai',
        keterangan: 'Sudah ada status dengan nama itu. Pakai nama lain.',
        nada: 'penolakan',
      }

    case KodeGalat.validasiGagal:
      return {
        judul: 'Isian belum benar',
        // Pesan dari server dipakai apa adanya: ia yang tahu aturan mana yang dilanggar,
        // dan menerjemahkannya ulang di sini akan membuat keduanya dapat berbeda.
        keterangan: galat.detail.map((d) => d.pesan).join(' ') || galat.message,
        nada: 'penolakan',
      }

    case KodeGalat.statusKlaimTidakDitemukan:
      return {
        judul: 'Status tidak ditemukan',
        keterangan: 'Baris ini mungkin sudah diubah orang lain. Muat ulang daftarnya.',
        nada: 'penolakan',
      }

    case KodeGalat.kodeStatusSudahDipakai:
      return {
        judul: 'Kode bentrok',
        keterangan: 'Kode yang dibuat sistem sudah dipakai. Coba simpan sekali lagi.',
        nada: 'gangguan',
      }

    default:
      return {
        judul: 'Gagal menyimpan',
        keterangan: galat.message,
        nada: 'gangguan',
      }
  }
}
