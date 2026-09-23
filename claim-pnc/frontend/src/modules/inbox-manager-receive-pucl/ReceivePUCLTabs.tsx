import type { Tab } from './types'

type Props = {
  tabs: Tab[]
  /** Kode tab yang sedang terbuka. */
  active: string
  onSelect: (code: string) => void
}

/**
 * Bilah tab Inbox Manager Receive / PUCL.
 *
 * # Apa yang digantikan
 *
 * Layout TABBED pada `Section/InboxManagerReceive_Section-Section.xml`, yang di sana punya
 * DUA judul tab — `<pyTitle>Receive</pyTitle>` dan `<pyTitle>RCL/PUCL</pyTitle>` — tetapi
 * TIGA grid.
 *
 * Tab "Receive" di sana memuat dua grid bertumpuk: `ManagementRecieveView` dijalankan dua
 * kali, yang pertama dengan `<Position1>"PA"</Position1>` dan yang kedua dengan
 * `"NONMBU"`. Keduanya ber-`pyVisible` `ALWAYS`, sehingga keduanya tampil bersamaan — dan
 * keduanya TIDAK punya judul apa pun. Pengguna melihat dua tabel yang kolomnya identik,
 * berurutan ke bawah, tanpa satu pun tanda mana yang mana.
 *
 * Di sini keduanya menjadi tab tersendiri dan diberi judul. Itu perubahan yang disadari,
 * dan pola yang sama sudah ditempuh modul Inbox Claim Treaty Non Prop. Alasannya di
 * `internal/inboxmanagerreceivepucl/tab.go`; ringkasnya dua — masing-masing grid punya
 * halamannya sendiri, dan tabel tanpa judul yang isinya berbeda adalah cacat tampilan yang
 * tidak perlu dibawa.
 *
 * # Judul yang ADA di Pega dipertahankan apa adanya
 *
 * "RCL/PUCL" ditulis persis seperti di section, termasuk garis miringnya. Kedua judul tab
 * Receive tidak ada di sana sama sekali, dan yang dipakai adalah judul tab aslinya
 * ditambah lini bisnis yang disaringnya, dalam ejaan yang sama dengan nilai parameter
 * Report Definition-nya ("PA" dan "NONMBU") — bukan ejaan yang lebih rapi.
 *
 * # Tab terhalang tetap dapat diklik
 *
 * Tidak ada yang terhalang di layar ini hari ini. Penandanya tetap digambar supaya
 * penambahan tab terhalang kelak cukup dilakukan di backend — dan bila itu terjadi,
 * mengkliknya menampilkan alasan beserta pemiliknya, bukan tabel kosong yang terbaca
 * sebagai "tidak ada pekerjaan".
 */
export function ReceivePUCLTabs({ tabs, active, onSelect }: Props) {
  return (
    /*
      Digulir menyamping pada layar sempit, bukan dilipat menjadi dropdown. Melipatnya
      menyembunyikan antrean mana saja yang tersedia — hal pertama yang ingin dilihat
      penyelia saat membuka layar.
    */
    <div
      className="overflow-x-auto"
      role="tablist"
      aria-label="Antrean penerimaan dokumen dan RCL/PUCL"
    >
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
                Penanda dibaca pembaca layar pula, bukan hanya terlihat. Tab yang terhalang
                adalah keadaan yang harus diketahui SEBELUM diklik, bukan sesudahnya.
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
