import { useEffect, useState } from 'react'

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

  return (
    <div role="status" className="border-b border-amber-300 bg-amber-50 px-4 py-2 text-sm text-amber-900">
      <div className="mx-auto flex max-w-5xl flex-wrap items-center justify-between gap-2">
        <span>
          {habis
            ? 'Sesi Anda sudah berakhir. Simpan pekerjaan Anda sebelum masuk kembali.'
            : `Sesi Anda berakhir dalam ${menit}.${String(detik).padStart(2, '0')} menit.`}
        </span>
        {!habis && (
          <button
            type="button"
            onClick={() => perpanjang.mutate()}
            disabled={perpanjang.isPending}
            className="rounded border border-amber-400 bg-white px-3 py-1 font-medium hover:bg-amber-100 disabled:cursor-not-allowed disabled:opacity-60"
          >
            {perpanjang.isPending ? 'Memperpanjang…' : 'Perpanjang sesi'}
          </button>
        )}
      </div>
    </div>
  )
}
