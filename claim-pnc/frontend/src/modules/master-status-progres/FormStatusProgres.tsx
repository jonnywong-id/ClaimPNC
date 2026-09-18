import { zodResolver } from '@hookform/resolvers/zod'
import { useEffect } from 'react'
import { useForm } from 'react-hook-form'
import { z } from 'zod'

import { GalatAPI, GalatJaringan } from '@/api/klien'
import { KodeGalat, type DetailGalat, type PosisiKlaim, type StatusProgres } from '@/api/tipe'
import { KolomIsian } from '@/components/KolomIsian'
import { KolomPilihan } from '@/components/KolomPilihan'
import { PesanGalat, type NadaGalat } from '@/components/PesanGalat'
import { Tombol } from '@/components/Tombol'

/**
 * Batas panjang nama harus sama dengan statusprogres.BatasPanjangNama di backend.
 *
 * Nilainya panjang kolom STS_PROGRESS1 yang ditetapkan Work Owner 2026-09-17 — bukan
 * tebakan. Diperiksa di dua tempat dengan sengaja: di sini supaya pengguna tahu sebelum
 * mengirim, dan di server karena API dapat ditembak tanpa melewati layar ini. Server
 * tetap yang berwenang — pemeriksaan di sini hanya kenyamanan.
 *
 * Bila angka ini berubah, `internal/statusprogres/statusprogres.go` harus ikut berubah.
 */
const BATAS_PANJANG_NAMA = 100

const skema = z.object({
  nama: z
    .string()
    .trim()
    .min(1, 'Nama status progres wajib diisi.')
    .max(BATAS_PANJANG_NAMA, `Nama status progres paling panjang ${BATAS_PANJANG_NAMA} karakter.`),
  kode_posisi: z.string().trim().min(1, 'Posisi klaim wajib dipilih.'),
})

export type IsianForm = z.infer<typeof skema>

type Props = {
  /** Baris yang disunting; null berarti penambahan baru. */
  disunting: StatusProgres | null
  posisi: PosisiKlaim[]
  sedangMenyimpan: boolean
  galat: unknown
  onSimpan: (isian: IsianForm) => void
  onBatal: () => void
}

type IsiPesan = { judul: string; keterangan: string; nada: NadaGalat }

/**
 * Mengubah galat penyimpanan menjadi pesan yang dapat ditindaklanjuti.
 *
 * Galat validasi TIDAK ditangani di sini — ia disorot per isian (lihat pelanggaranPer).
 * Yang ditampilkan sebagai kotak pesan hanyalah galat yang tidak menunjuk isian
 * tertentu, karena itulah yang tidak dapat diperbaiki pengguna dengan mengetik.
 */
function pesanUntuk(galat: unknown): IsiPesan | null {
  if (galat instanceof GalatJaringan) {
    return {
      judul: 'Server Claim PNC tidak dapat dihubungi',
      keterangan: 'Isian Anda belum tersimpan. Periksa koneksi jaringan, lalu simpan lagi.',
      nada: 'gangguan',
    }
  }
  if (galat instanceof GalatAPI) {
    switch (galat.kode) {
      case KodeGalat.validasiGagal:
        // Bila detailnya ada, isiannya sudah disorot satu per satu; kotak pesan hanya
        // akan mengulang hal yang sama. Bila detailnya TIDAK ada — misalnya nomor baru
        // bentrok dengan petugas lain — pesan servernya yang ditampilkan.
        return pelanggaranPer(galat).length > 0
          ? null
          : {
              judul: 'Belum dapat disimpan',
              keterangan: galat.message,
              nada: 'penolakan',
            }
      case KodeGalat.tidakDitemukan:
        return {
          judul: 'Baris ini sudah tidak ada',
          keterangan:
            'Mungkin sudah diubah petugas lain. Tutup form ini dan muat ulang daftarnya.',
          nada: 'penolakan',
        }
      case KodeGalat.portalTidakDisebut:
      case KodeGalat.portalTidakDikenal:
        return {
          judul: 'Portal entitas belum dipilih',
          keterangan: 'Pilih portal entitas di bagian atas halaman, lalu simpan lagi.',
          nada: 'penolakan',
        }
      case KodeGalat.portalBelumSiap:
        return {
          judul: 'Basis data entitas ini belum tersedia',
          keterangan:
            'Mengulang tidak akan menolong. Hubungi administrator Claim PNC untuk melengkapi kredensial basis datanya.',
          nada: 'gangguan',
        }
      default:
        return {
          judul: 'Terjadi kesalahan pada sistem',
          keterangan: 'Isian Anda belum tersimpan. Coba beberapa saat lagi.',
          nada: 'gangguan',
        }
    }
  }
  return null
}

/** Mengambil pelanggaran per isian dari galat validasi server. */
function pelanggaranPer(galat: unknown): DetailGalat[] {
  return galat instanceof GalatAPI ? galat.detail : []
}

/**
 * FormStatusProgres adalah satu form untuk DUA mode — tambah dan ubah.
 *
 * Satu form, bukan dua, mengikuti sistem lama: `UpdateStatusProgress1_act` memuat baris
 * ke modal yang sama lalu menandainya "Update" (`TempDcol.pyLabel`). Isian yang
 * dibandingkan pengguna karena itu berada di tempat yang sama pada kedua mode.
 *
 * ID tidak dapat disunting. Di Pega pun begitu: `UpdateStatusProgress1_sql` memakai
 * ID_PROGRESS hanya sebagai penyaring `WHERE`, tidak pernah sebagai kolom yang di-SET —
 * ia dirujuk `GCNM_PROGRESS_CLAIM.STATUS_PROGRESS1` dan `GCNM_MST_PROGRESS.ID_PROGRESS`.
 */
export function FormStatusProgres({
  disunting,
  posisi,
  sedangMenyimpan,
  galat,
  onSimpan,
  onBatal,
}: Props) {
  const modeUbah = disunting !== null

  const {
    register,
    handleSubmit,
    reset,
    setError,
    formState: { errors },
  } = useForm<IsianForm>({
    resolver: zodResolver(skema),
    defaultValues: {
      nama: disunting?.nama ?? '',
      kode_posisi: disunting?.kode_posisi ?? '',
    },
  })

  // Isian disesuaikan ketika baris yang disunting berganti tanpa form ditutup lebih
  // dulu — misalnya pengguna menekan "Ubah" pada baris lain.
  useEffect(() => {
    reset({ nama: disunting?.nama ?? '', kode_posisi: disunting?.kode_posisi ?? '' })
  }, [disunting, reset])

  // Pelanggaran yang dilaporkan server disorot pada isiannya masing-masing, bukan
  // hanya diringkas di satu kotak pesan. Server mengirim SELURUH pelanggaran sekaligus
  // (P-5), dan itu hanya berguna bila layar menyorotnya satu per satu.
  useEffect(() => {
    for (const p of pelanggaranPer(galat)) {
      if (p.kolom === 'nama' || p.kolom === 'kode_posisi') {
        setError(p.kolom, { type: 'server', message: p.pesan })
      }
    }
  }, [galat, setError])

  const pesan = pesanUntuk(galat)
  const judul = modeUbah ? 'Ubah Status Progres' : 'Tambah Status Progres'

  return (
    <form
      onSubmit={handleSubmit(onSimpan)}
      noValidate
      aria-label={judul}
      className="space-y-4 rounded-lg border border-slate-200 bg-white p-5 shadow-sm"
    >
      <h2 className="text-base font-semibold text-slate-900">{judul}</h2>

      {pesan && (
        <PesanGalat judul={pesan.judul} keterangan={pesan.keterangan} nada={pesan.nada} />
      )}

      {/* ID hanya ditampilkan saat menyunting, dan tidak dapat diubah. Pada penambahan
          ia belum ada — nomornya diterbitkan server dari isi tabel. */}
      {modeUbah && (
        <div>
          <span className="block text-sm font-medium text-slate-700">ID</span>
          <p className="mt-1 rounded border border-slate-200 bg-slate-50 px-3 py-2 text-slate-600">
            {disunting.id}
            <span className="ml-2 text-xs text-slate-500">(tidak dapat diubah)</span>
          </p>
        </div>
      )}

      <KolomIsian
        id="nama"
        label="Status Progres"
        type="text"
        autoFocus
        maxLength={BATAS_PANJANG_NAMA}
        galat={errors.nama?.message}
        {...register('nama')}
      />

      <KolomPilihan
        id="kode_posisi"
        label="Posisi"
        pilihan={posisi.map((p) => ({ nilai: p.kode, label: p.nama }))}
        galat={errors.kode_posisi?.message}
        {...register('kode_posisi')}
      />

      <div className="flex flex-wrap justify-end gap-2 pt-2">
        <Tombol peran="halus" onClick={onBatal} disabled={sedangMenyimpan}>
          Batal
        </Tombol>
        <Tombol
          type="submit"
          peran="utama"
          sedangJalan={sedangMenyimpan}
          teksSedangJalan="Menyimpan…"
        >
          Simpan
        </Tombol>
      </div>
    </form>
  )
}
