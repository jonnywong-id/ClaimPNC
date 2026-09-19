import { useEffect, useState } from 'react'

import { Button } from '@/components/Button'
import { useExtendSession } from '@/modules/login/api'
import { useSession } from '@/app/session'

/** Peringatan muncul ketika sisa waktu sesi tinggal di bawah ambang ini. */
const WARNING_THRESHOLD_MS = 5 * 60 * 1000

/**
 * SessionWarning memberi tahu pengguna sebelum sesinya habis.
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
export function SessionWarning() {
  const validUntil = useSession((state) => state.validUntil)
  const extend = useExtendSession()
  const [remainingMs, setRemainingMs] = useState<number | null>(null)

  useEffect(() => {
    if (!validUntil) {
      setRemainingMs(null)
      return
    }
    const batas = new Date(validUntil).getTime()
    const count = () => setRemainingMs(batas - Date.now())
    count()
    const controller = window.setInterval(count, 1000)
    return () => window.clearInterval(controller)
  }, [validUntil])

  if (remainingMs === null || remainingMs > WARNING_THRESHOLD_MS) return null

  const expired = remainingMs <= 0
  const minutes = Math.max(0, Math.floor(remainingMs / 60000))
  const seconds = Math.max(0, Math.floor((remainingMs % 60000) / 1000))

  const style = expired
    ? 'border-red-200 bg-red-50 text-red-900'
    : 'border-amber-200 bg-amber-50 text-amber-900'

  return (
    <div role="status" className={`border-b px-4 py-2.5 text-sm ${style}`}>
      <div className="mx-auto flex max-w-6xl flex-wrap items-center justify-between gap-2">
        <span className="flex items-center gap-2">
          <svg
            viewBox="0 0 20 20"
            aria-hidden="true"
            className={`h-4 w-4 shrink-0 fill-current ${expired ? '' : 'animate-pulse'}`}
          >
            <path d="M10 2a8 8 0 1 0 0 16 8 8 0 0 0 0-16Zm.9 4.2v3.5l2.4 1.4a.9.9 0 1 1-.9 1.5l-2.8-1.6a.9.9 0 0 1-.5-.8V6.2a.9.9 0 1 1 1.8 0Z" />
          </svg>
          {expired
            ? 'Sesi Anda sudah berakhir. Simpan pekerjaan Anda sebelum masuk kembali.'
            : `Sesi Anda berakhir dalam ${minutes}.${String(seconds).padStart(2, '0')} menit.`}
        </span>
        {!expired && (
          <Button
            tone="kedua"
            onClick={() => extend.mutate()}
            disabled={extend.isPending}
            className="border-amber-300 !py-1.5 text-amber-900 hover:border-amber-400 hover:bg-amber-100"
          >
            {extend.isPending ? 'Memperpanjang…' : 'Perpanjang sesi'}
          </Button>
        )}
      </div>
    </div>
  )
}
