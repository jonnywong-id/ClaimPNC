import type { ReactNode } from 'react'

import { formatAdviceDate } from '@/api/inboxMessages'

/**
 * adviceCell menggambar satu sel grid dokumen pemberitahuan reasuransi.
 *
 * Kolom "Terkirim" digambar sebagai lencana, bukan sebagai teks mentah: nilainya di basis
 * data adalah kosong, `"0"`, atau `"1"` — tidak satu pun terbaca manusia, dan dua yang
 * pertama berarti hal yang SAMA bagi pengguna. Kolom tanggal diformat; teks kosong menjadi
 * tanda pisah.
 */
export function adviceCell(isi: string, kunci: string, tanggal: boolean): ReactNode {
  if (kunci === 'terkirim') {
    const sudah = isi === '1'
    return (
      <span
        className={[
          'inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium',
          sudah ? 'bg-emerald-50 text-emerald-700' : 'bg-amber-50 text-amber-800',
        ].join(' ')}
      >
        {sudah ? 'Terkirim' : 'Belum'}
      </span>
    )
  }

  if (tanggal) return formatAdviceDate(isi)
  if (isi === '') return <span className="text-slate-400">—</span>
  return isi
}
