import type { Tab } from './types'

type Props = {
  tabs: Tab[]
  /** Kode tab yang sedang terbuka. */
  active: string
  onSelect: (code: string) => void
}

/**
 * Bilah tab Inbox Compliance.
 *
 * # Apa yang digantikan
 *
 * Dua kontainer `TABBED` pada `Section/InputCompliance_Section-Section.xml` — "Compliance"
 * (`:985`) dan "Post Audit" (`:2315`). Judulnya dipertahankan apa adanya, termasuk yang
 * berbahasa Inggris, karena `D-13` menetapkan tampilan meniru Pega supaya pengguna tidak
 * perlu belajar ulang.
 *
 * # Kenapa tab yang belum dapat dilayani tetap digambar
 *
 * Karena menyembunyikannya membuat pengguna yang mencari "Post Audit" menduga modulnya
 * belum selesai — sementara yang sebenarnya kurang adalah satu artefak yang pemiliknya
 * jelas. Tab itu tetap dapat diklik, dan yang tampil adalah penjelasan apa yang ditunggu,
 * bukan tabel kosong yang tidak menerangkan apa pun.
 *
 * Ia dibedakan secara visual DAN lewat `aria-disabled`, bukan lewat warna saja: perbedaan
 * yang hanya berupa warna tidak sampai kepada pembaca layar maupun pengguna yang sulit
 * membedakan warna.
 *
 * # Kenapa tanpa lencana jumlah
 *
 * Sistem lama tidak menampilkannya, dan menghadirkannya berarti menjalankan kueri kedua tab
 * setiap kali layar dibuka — termasuk tab yang belum punya kueri.
 */
export function ComplianceTabs({ tabs, active, onSelect }: Props) {
  return (
    /*
      Digulir menyamping pada layar sempit, bukan dilipat menjadi dropdown. Dua tab
      sebenarnya muat di ponsel, tetapi bentuknya dibuat sama dengan Inbox Admin supaya
      kedua layar antrean kerja tidak terasa dirakit dari dua aplikasi berbeda.
    */
    <div className="overflow-x-auto" role="tablist" aria-label="Antrean Inbox Compliance">
      <div className="flex min-w-max items-center gap-1.5 border-b border-slate-200 pb-px">
        {tabs.map((tab) => {
          const selected = tab.kode === active
          const pending = !tab.tersedia

          return (
            <button
              key={tab.kode}
              type="button"
              role="tab"
              aria-selected={selected}
              aria-disabled={pending}
              title={pending ? tab.penghalang : tab.keterangan}
              onClick={() => onSelect(tab.kode)}
              className={[
                'rounded-t-kontrol border-b-2 px-3.5 py-2.5',
                'text-sm font-medium whitespace-nowrap',
                'transition-[color,border-color,background-color] duration-150 ease-halus',
                'focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500/50',
                selected
                  ? 'border-blue-600 text-blue-700'
                  : 'border-transparent text-slate-600 hover:border-slate-300 hover:bg-slate-50 hover:text-slate-900',
                pending && !selected ? 'text-slate-400' : '',
              ].join(' ')}
            >
              {tab.nama}
              {pending && (
                <span
                  className="ml-2 rounded-full bg-amber-100 px-2 py-0.5 text-xs font-normal text-amber-800"
                  // Teksnya pendek supaya bilah tab tidak melebar, sedangkan alasan
                  // lengkapnya ada di panel penjelas begitu tabnya dibuka.
                >
                  belum tersedia
                </span>
              )}
            </button>
          )
        })}
      </div>
    </div>
  )
}
