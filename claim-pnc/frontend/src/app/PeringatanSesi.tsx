import { useEffect, useState } from 'react'

import { Tombol } from '@/components/Tombol'
import { gunakanPerpanjangSesi } from '@/modules/masuk/api'
import { gunakanSesi } from '@/app/sesi'

/** Peringatan muncul ketika sisa waktu sesi tinggal di bawah ambang ini. */
const AMBANG_PERINGATAN_MS = 5 * 60 * 1000

/**
 * PeringatanSesi memberi tahu pengguna sebelum sesinya habis.
 *
 * Alasannya bukan kerapian: pengguna sistem klaim mengisi form panjang, dan sesi yang
 * habis di tengah pengisian tanpa peringatan berarti pekerjaan hilang (TKT-U1-002).
 * Isian yang belum tersimpan tidak dikirim ke mana pun oleh komponen ini — ia tetap di
 * memori peramban sampai pengguna menyimpannya sendiri.
 *
 * # Kenapa dua nada warna
 *
 * Sesi yang MASIH berjalan berwarna kuning: ada yang dapat dilakukan, dan tombol
 * Perpanjang ada di sebelahnya. Sesi yang SUDAH habis berwarna merah dan tanpa tombol —
 * memperpanjang tidak lagi mungkin, dan menampilkan tombol yang pasti gagal hanya
 * membuang waktu pengguna.
 */
export function PeringatanSesi() {
  const berlakuSampai = gunakanSesi((keadaan) => keadaan.berlakuSampai)
  const perpanjang = gunakanPerpanjangSesi()
  const [sisaMs, setSisaMs] = useState<number | null>(null)

  useEffect(() => {
    if (!berlakuSampai) {
      setSisaMs(null)
      return
    }
    const batas = new Date(berlakuSampai).getTime()
    const hitung = () => setSisaMs(batas - Date.now())
    hitung()
    const pengatur = window.setInterval(hitung, 1000)
    return () => window.clearInterval(pengatur)
  }, [berlakuSampai])

  if (sisaMs === null || sisaMs > AMBANG_PERINGATAN_MS) return null

  const habis = sisaMs <= 0
  const menit = Math.max(0, Math.floor(sisaMs / 60000))
  const detik = Math.max(0, Math.floor((sisaMs % 60000) / 1000))

  const gaya = habis
    ? 'border-red-200 bg-red-50 text-red-900'
    : 'border-amber-200 bg-amber-50 text-amber-900'

  return (
    <div role="status" className={`border-b px-4 py-2.5 text-sm ${gaya}`}>
      <div className="mx-auto flex max-w-6xl flex-wrap items-center justify-between gap-2">
        <span className="flex items-center gap-2">
          <svg
            viewBox="0 0 20 20"
            aria-hidden="true"
            className={`h-4 w-4 shrink-0 fill-current ${habis ? '' : 'animate-pulse'}`}
          >
            <path d="M10 2a8 8 0 1 0 0 16 8 8 0 0 0 0-16Zm.9 4.2v3.5l2.4 1.4a.9.9 0 1 1-.9 1.5l-2.8-1.6a.9.9 0 0 1-.5-.8V6.2a.9.9 0 1 1 1.8 0Z" />
          </svg>
          {habis
            ? 'Sesi Anda sudah berakhir. Simpan pekerjaan Anda sebelum masuk kembali.'
            : `Sesi Anda berakhir dalam ${menit}.${String(detik).padStart(2, '0')} menit.`}
        </span>
        {!habis && (
          <Tombol
            nada="kedua"
            onClick={() => perpanjang.mutate()}
            disabled={perpanjang.isPending}
            className="border-amber-300 !py-1.5 text-amber-900 hover:border-amber-400 hover:bg-amber-100"
          >
            {perpanjang.isPending ? 'Memperpanjang…' : 'Perpanjang sesi'}
          </Tombol>
        )}
      </div>
    </div>
  )
}
