import type { AutoClaimTab } from '@/api/types'

type Props = {
  tab: AutoClaimTab[]
  selected: string
  onSelect: (source: string) => void
}

/**
 * Tab ANEKA / Asuransi Kredit / Travel.
 *
 * # Ketiganya satu harness, tiga tabel
 *
 * `InboxAutoClaim/InboxAutoClaim-Harness.xml` memuat ketiga tab ini (`pyCaption`), dan
 * masing-masing membaca TABEL yang berbeda:
 *
 *	ANEKA            TMP_BATCH_AUTO_CLAIM    kolom INISIALID
 *	Asuransi Kredit  TMP_BATCH_CLAIM_KREDIT  kolom AGENID
 *	Travel           TMP_BATCH_AUTO_TRAVEL   kolom INISIALID
 *
 * Label yang dipakai di sini disalin dari harness, bukan diturunkan dari nama rule:
 * rule-nya bernama `*_AutoClaim` sementara yang dilihat pengguna adalah **ANEKA**.
 *
 * # Kenapa `role="tablist"`, bukan sekadar deretan tombol
 *
 * Dengan peran yang benar, pembaca layar mengumumkan "tab 2 dari 3" dan panah kiri/kanan
 * berpindah tab — perilaku yang diharapkan pengguna papan ketik dari sesuatu yang terlihat
 * seperti tab. Deretan tombol biasa terbaca sebagai tiga tombol lepas tanpa hubungan.
 */
export function SourceTab({ tab, selected, onSelect }: Props) {
  if (tab.length === 0) return null

  function pindah(arah: -1 | 1) {
    const sekarang = tab.findIndex((t) => t.kode === selected)
    if (sekarang < 0) return
    // Berputar di ujungnya, mengikuti perilaku tab yang lazim.
    const tujuan = (sekarang + arah + tab.length) % tab.length
    const berikut = tab[tujuan]
    if (berikut !== undefined) onSelect(berikut.kode)
  }

  return (
    <div role="tablist" aria-label="Jenis klaim" className="flex gap-1 border-b border-slate-200">
      {tab.map((t) => {
        const aktif = t.kode === selected
        return (
          <button
            key={t.kode}
            role="tab"
            type="button"
            aria-selected={aktif}
            // Hanya tab yang aktif yang dapat di-Tab-kan; panah memindahkan yang lain.
            // Itu pola papan ketik baku untuk tablist, dan tanpa itu pengguna papan ketik
            // harus menekan Tab melewati setiap tab sebelum sampai ke isinya.
            tabIndex={aktif ? 0 : -1}
            onClick={() => onSelect(t.kode)}
            onKeyDown={(event) => {
              if (event.key === 'ArrowRight') {
                event.preventDefault()
                pindah(1)
              }
              if (event.key === 'ArrowLeft') {
                event.preventDefault()
                pindah(-1)
              }
            }}
            className={[
              '-mb-px border-b-2 px-4 py-2.5 text-sm font-medium whitespace-nowrap',
              'transition-[color,border-color] duration-150 ease-halus',
              'focus:outline-none focus-visible:ring-4 focus-visible:ring-blue-500/20',
              aktif
                ? 'border-blue-600 text-blue-700'
                : 'border-transparent text-slate-600 hover:border-slate-300 hover:text-slate-900',
            ].join(' ')}
          >
            {t.label}
          </button>
        )
      })}
    </div>
  )
}
