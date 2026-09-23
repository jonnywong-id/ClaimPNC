import { KomiteInboxKind, type KomiteInboxSummary } from '@/api/types'

type Props = {
  summary: KomiteInboxSummary | undefined
  active: KomiteInboxKind
  onSelect: (kind: KomiteInboxKind) => void
  loading?: boolean
}

/**
 * Urutan tab mengikuti ALUR KERJA, bukan abjad.
 *
 * Anggota komite membuka layar ini untuk mengerjakan sesuatu, bukan untuk membaca
 * riwayat — yang menunggu karena itu berada paling kiri.
 */
const TABS: { kind: KomiteInboxKind; label: string; hint: string }[] = [
  {
    kind: KomiteInboxKind.outstanding,
    label: 'Outstanding',
    hint: 'Kasus yang menunggu keputusan Anda',
  },
  {
    kind: KomiteInboxKind.accepted,
    label: 'Diterima',
    hint: 'Kasus yang sudah Anda setujui',
  },
  {
    kind: KomiteInboxKind.rejected,
    label: 'Ditolak',
    hint: 'Kasus yang Anda tolak atau kembalikan',
  },
]

/**
 * Tab kotak Inbox Komite beserta lencana jumlahnya.
 *
 * # Apa yang digantikan
 *
 * Tiga grid pada `Section/InboxKomite_section-Section.xml` — "Kotak Masuk Komite
 * Outstanding", "…Diterima", dan "…Ditolak" — yang di sana adalah tiga bagian layar
 * terpisah, masing-masing dengan kueri dan tombol carinya sendiri.
 *
 * # Kenapa tab, bukan tiga tabel bertumpuk seperti aslinya
 *
 * Ketiganya tidak pernah dibaca bersamaan. Yang dibutuhkan anggota komite saat membuka
 * layar adalah **berapa yang menunggu**, dan itu terbaca dari satu lencana. Tiga tabel
 * bertumpuk menuntutnya menggulir melewati riwayat untuk sampai ke pekerjaannya.
 *
 * # Lencana "Ditolak" menghitung DUA keputusan
 *
 * Tolak dan kembalikan sama-sama menghentikan komite, dan keduanya masuk kotak ini. Itu
 * disebutkan pada keterangan tab, bukan dibiarkan disimpulkan dari isinya.
 */
export function InboxTabs({ summary, active, onSelect, loading = false }: Props) {
  return (
    /*
      Digulir menyamping pada layar sempit, bukan dilipat menjadi dropdown. Melipatnya
      menyembunyikan justru angka yang paling ingin dilihat saat membuka layar.
    */
    <div className="overflow-x-auto" role="tablist" aria-label="Kotak Inbox Komite">
      <div className="flex min-w-max items-center gap-1.5 border-b border-slate-200 pb-px">
        {TABS.map((tab) => {
          const selected = tab.kind === active
          const count = summary?.[tab.kind]

          return (
            <button
              key={tab.kind}
              type="button"
              role="tab"
              aria-selected={selected}
              title={tab.hint}
              onClick={() => onSelect(tab.kind)}
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
              {tab.label}
              {/*
                Lencana dibedakan warnanya saat terpilih, TETAPI juga oleh warna teks dan
                garis bawah tabnya. Warna saja tidak terbaca oleh sebagian pengguna.

                Angkanya ditulis sebagai `—` selama belum ada, bukan sebagai 0: nol berarti
                "tidak ada pekerjaan", dan itu keterangan yang berbeda dari "belum tahu".
              */}
              <span
                className={[
                  'inline-flex min-w-[1.5rem] items-center justify-center rounded-full px-1.5 py-0.5',
                  'text-xs font-semibold tabular-nums',
                  'transition-[background-color,color] duration-150 ease-halus',
                  selected ? 'bg-blue-100 text-blue-800' : 'bg-slate-100 text-slate-600',
                  loading ? 'opacity-50' : '',
                ].join(' ')}
              >
                {count === undefined ? '—' : count}
              </span>
            </button>
          )
        })}
      </div>
    </div>
  )
}
