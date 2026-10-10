import { useEffect, useRef } from 'react'

import { Button } from '@/components/Button'

import { Isian_, type Isian } from './BagianRincian'

/**
 * Sub-popup yang terbuka dari satu baris grid di dalam popup rincian.
 *
 * # Kenapa ia ada
 *
 * Tujuh grid di dalam `ViewTempDetailClaim` memasang Flow Action sebagai `pyEditAction` —
 * aksi yang Pega jalankan saat baris dibuka:
 *
 *	ViewCoverageAdj · ViewCoverageGrid_fa · ViewInputReceiver · DetailHasilSurveyor ·
 *	ShowSurveyResults · ShowDetailProgress · KomiteCoverageGrid_ClaimHE
 *
 * Ketujuhnya membuka section tersendiri. Di sini satu komponen melayani semuanya, karena yang
 * berbeda hanya judul dan daftar isiannya — bukan perilakunya.
 *
 * # Barisnya dibawa, bukan diambil ulang
 *
 * Isi sub-popup seluruhnya berasal dari baris yang diklik, dan baris itu sudah ada di tangan.
 * Mengambilnya ulang dari server akan menambah perjalanan jaringan untuk data yang sudah
 * terbaca — dan membuka kemungkinan keduanya berbeda.
 */
export function SubPopupRincian({
  judul,
  baris,
  isian,
  onTutup,
}: {
  judul: string
  baris: Record<string, unknown>
  isian: Isian[]
  onTutup: () => void
}) {
  const tutupRef = useRef<HTMLButtonElement>(null)

  useEffect(() => {
    tutupRef.current?.focus()

    function onKey(event: KeyboardEvent) {
      // Escape menutup sub-popup SAJA, bukan popup di belakangnya.
      //
      // Tanpa stopPropagation, satu tekan Escape menutup keduanya sekaligus — dan pengguna
      // yang hanya ingin kembali ke daftar kehilangan seluruh rinciannya.
      if (event.key === 'Escape') {
        event.stopPropagation()
        onTutup()
      }
    }
    document.addEventListener('keydown', onKey, true)
    return () => document.removeEventListener('keydown', onKey, true)
  }, [onTutup])

  return (
    <div
      className="fixed inset-0 z-[60] flex items-center justify-center bg-slate-900/40 p-4"
      role="dialog"
      aria-modal="true"
      aria-label={judul}
    >
      <div className="max-h-[85vh] w-full max-w-2xl overflow-y-auto rounded-kartu bg-white p-6 shadow-terbang">
        <div className="mb-4 flex items-start justify-between gap-4">
          <h3 className="text-base font-semibold text-slate-900">{judul}</h3>
          <button
            ref={tutupRef}
            type="button"
            onClick={onTutup}
            aria-label="Tutup"
            className={[
              'rounded-kontrol px-3 py-1.5 text-sm text-slate-500',
              'transition ease-halus hover:bg-slate-100 hover:text-slate-700',
              'focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2',
              'focus-visible:outline-blue-600',
            ].join(' ')}
          >
            ✕
          </button>
        </div>

        <Isian_ dokumen={baris} isian={isian} />

        <div className="mt-6 flex justify-end">
          <Button tone="utama" onClick={onTutup}>
            Tutup
          </Button>
        </div>
      </div>
    </div>
  )
}
