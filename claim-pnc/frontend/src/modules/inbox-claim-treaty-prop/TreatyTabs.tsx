import type { Tab } from './types'

type Props = {
  tabs: Tab[]
  /** Kode tab yang sedang terbuka. */
  active: string
  onSelect: (code: string) => void
}

/**
 * Bilah tab Inbox Claim Treaty Prop.
 *
 * # Apa yang digantikan
 *
 * Tiga kontainer pada `Section/InboxClaimTreaty_Section-Section.xml`. Di sana ketiganya
 * BUKAN tab yang dapat diklik: yang memilih di antaranya adalah KEADAAN pemanggil —
 * `InputData.CARI13 == 'tampil'` bila ia anggota komite, dan
 * `SearchWorkbasket.CARI1 == 'TreatyinPNCTeknik'` bila ia memegang akun antrean teknik.
 * Pengguna tidak pernah dapat berpindah di antaranya.
 *
 * Di sini ketiganya menjadi tab yang dapat dipilih, dan itu perubahan yang disadari:
 * pemeriksaan peran belum ada (`TKT-F3-004`), sehingga tidak ada cara menentukan keadaan
 * pemanggil. Menyembunyikan antrean berdasarkan tebakan akan membuat petugas yang
 * seharusnya melihatnya justru tidak melihatnya sama sekali.
 *
 * Judul tabnya dipertahankan apa adanya — termasuk salah eja "Propotional" — karena `D-13`
 * menetapkan tampilan meniru Pega supaya pengguna tidak perlu belajar ulang.
 *
 * # Tab terhalang tetap dapat diklik
 *
 * Keputusan Work Owner 2026-09-21. Ia digambar dengan penanda, dan mengkliknya menampilkan
 * alasan beserta pemiliknya — bukan tabel kosong yang terbaca sebagai "tidak ada
 * pekerjaan". Menonaktifkannya akan membuat pengguna tidak pernah tahu kenapa.
 */
export function TreatyTabs({ tabs, active, onSelect }: Props) {
  return (
    /*
      Digulir menyamping pada layar sempit, bukan dilipat menjadi dropdown. Judul tab di
      layar ini panjang-panjang, dan melipatnya menyembunyikan antrean mana saja yang
      tersedia — hal pertama yang ingin dilihat petugas saat membuka layar.
    */
    <div className="overflow-x-auto" role="tablist" aria-label="Antrean klaim treaty proporsional">
      <div className="flex min-w-max items-center gap-1.5 border-b border-slate-200 pb-px">
        {tabs.map((tab) => {
          const selected = tab.kode === active
          return (
            <button
              key={tab.kode}
              type="button"
              role="tab"
              aria-selected={selected}
              title={tab.terhalang ? tab.alasan_terhalang : tab.keterangan}
              onClick={() => onSelect(tab.kode)}
              className={[
                'flex items-center gap-2 rounded-t-kontrol border-b-2 px-3.5 py-2.5',
                'text-sm font-medium whitespace-nowrap',
                'transition-[color,border-color,background-color] duration-150 ease-halus',
                'focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500/50',
                selected
                  ? 'border-blue-600 text-blue-700'
                  : 'border-transparent text-slate-600 hover:border-slate-300 hover:bg-slate-50 hover:text-slate-900',
              ].join(' ')}
            >
              {tab.nama}

              {/*
                Penanda dibaca pembaca layar pula, bukan hanya terlihat. Tab yang
                terhalang adalah keadaan yang harus diketahui SEBELUM diklik, bukan
                sesudahnya.
              */}
              {tab.terhalang && (
                <span className="rounded-full bg-amber-100 px-2 py-0.5 text-xs font-normal text-amber-900">
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
