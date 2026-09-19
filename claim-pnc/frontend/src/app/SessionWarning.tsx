import { useEffect, useState } from 'react'

import { useExtendSession } from '@/modules/login/api'
import { useSession } from '@/app/session'

/** Peringatan muncul ketika sisa waktu sesi tinggal di bawah ambang ini. */
const AMBANG_PERINGATAN_MS = 5 * 60 * 1000

/**
 * SessionWarning memberi tahu pengguna sebelum sesinya habis.
 *
 * Alasannya bukan kerapian: pengguna sistem klaim mengisi form panjang, dan sesi yang
 * habis di tengah pengisian tanpa peringatan berarti pekerjaan hilang (TKT-U1-002).
 * Isian yang belum tersimpan tidak dikirim ke mana pun oleh komponen ini — ia tetap di
 * memori peramban sampai pengguna menyimpannya sendiri.
 */
export function SessionWarning() {
  const expiresAt = useSession((state) => state.expiresAt)
  const extend = useExtendSession()
  const [remainingMs, setRemainingMs] = useState<number | null>(null)

  useEffect(() => {
    if (!expiresAt) {
      setRemainingMs(null)
      return
    }
    const limit = new Date(expiresAt).getTime()
    const compute = () => setRemainingMs(limit - Date.now())
    compute()
    const timer = window.setInterval(compute, 1000)
    return () => window.clearInterval(timer)
  }, [expiresAt])

  if (remainingMs === null || remainingMs > AMBANG_PERINGATAN_MS) return null

  const expired = remainingMs <= 0
  const minutes = Math.max(0, Math.floor(remainingMs / 60000))
  const seconds = Math.max(0, Math.floor((remainingMs % 60000) / 1000))

  return (
    <div role="status" className="border-b border-amber-300 bg-amber-50 px-4 py-2 text-sm text-amber-900">
      <div className="mx-auto flex max-w-5xl flex-wrap items-center justify-between gap-2">
        <span>
          {expired
            ? 'Sesi Anda sudah berakhir. Simpan pekerjaan Anda sebelum masuk kembali.'
            : `Sesi Anda berakhir dalam ${minutes}.${String(seconds).padStart(2, '0')} menit.`}
        </span>
        {!expired && (
          <button
            type="button"
            onClick={() => extend.mutate()}
            disabled={extend.isPending}
            className="rounded border border-amber-400 bg-white px-3 py-1 font-medium hover:bg-amber-100 disabled:cursor-not-allowed disabled:opacity-60"
          >
            {extend.isPending ? 'Memperpanjang…' : 'Perpanjang sesi'}
          </button>
        )}
      </div>
    </div>
  )
}
