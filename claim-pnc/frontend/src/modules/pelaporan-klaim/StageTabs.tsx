import type { StageSummary } from '@/api/types'

type Props = {
  summary: StageSummary[]
  /** Tahap yang sedang terbuka. Teks kosong berarti tab "Semua". */
  active: string
  onSelect: (stage: string) => void
  /** Jumlah seluruh laporan, untuk lencana tab "Semua". */
  total: number
  loading?: boolean
}

/**
 * Tab tahap beserta lencana jumlahnya.
 *
 * # Apa yang digantikan
 *
 * Bilah tab pada `Section/ViewStatusReceiveDocument-Section.xml`, yang di sana berjudul
 * campur dua bahasa — "Data hasn't been transferred" bersebelahan dengan "Claim yang sudah
 * Registrasi". Judulnya di sini diseragamkan ke bahasa Indonesia.
 *
 * Angka lencananya menggantikan `RDB List/BrowseClaimRCV_Aksep-SQL.xml`, satu kueri berisi
 * enam `SUM(CASE WHEN ...)` yang menghitung seluruh lencana sekaligus. Bentuk itu
 * dipertahankan — angkanya datang dari satu permintaan yang sama dengan isi tabelnya,
 * sehingga lencana tidak dapat berselisih dengan tabel di bawahnya.
 *
 * # Kenapa tab, bukan penyaring dropdown
 *
 * Tahap adalah antrean kerja, bukan atribut. Petugas penerimaan bekerja dari kiri ke
 * kanan: mencatat, mentransfer, lalu menunggu registrasi. Tab membuat urutan itu terlihat
 * dan membuat jumlah pekerjaan yang menunggu terbaca tanpa satu klik pun — dropdown
 * menyembunyikan keduanya.
 *
 * # Tab yang kosong TETAP ditampilkan
 *
 * Termasuk dua tahap yang hari ini belum dapat terjadi, karena modul klaim (`B-5`,
 * `B-10`) yang mengisinya belum ada. Tab yang menghilang saat kosong membuat pengguna
 * mengira tabnya tidak ada — dan bagi tahap yang memang belum pernah terjadi, perbedaan
 * antara "kosong" dan "tidak ada" itulah yang menentukan apakah ia melaporkan cacat.
 */
export function StageTabs({ summary, active, onSelect, total, loading = false }: Props) {
  const entries = [{ tahap: '', label: 'Semua', jumlah: total }, ...summary]

  return (
    /*
      Digulir menyamping pada layar sempit, bukan dilipat menjadi dropdown. Enam tab tidak
      muat berjajar di ponsel, tetapi melipatnya menyembunyikan justru angka yang paling
      ingin dilihat petugas saat membuka layar.
    */
    <div className="overflow-x-auto" role="tablist" aria-label="Tahap laporan">
      <div className="flex min-w-max items-center gap-1.5 border-b border-slate-200 pb-px">
        {entries.map((entry) => {
          const selected = entry.tahap === active
          return (
            <button
              key={entry.tahap || 'semua'}
              type="button"
              role="tab"
              aria-selected={selected}
              onClick={() => onSelect(entry.tahap)}
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
              {entry.label}
              {/*
                Lencana angka dibedakan warnanya saat tabnya terpilih, TETAPI juga dibedakan
                oleh tebal tepinya. Warna saja tidak terbaca oleh sebagian pengguna, dan
                yang membedakan tab aktif di sini sudah ada tiga: warna teks, garis bawah,
                dan aria-selected untuk pembaca layar.
              */}
              <span
                className={[
                  'inline-flex min-w-[1.5rem] items-center justify-center rounded-full px-1.5 py-0.5',
                  'text-xs font-semibold tabular-nums',
                  'transition-[background-color,color] duration-150 ease-halus',
                  selected ? 'bg-blue-100 text-blue-800' : 'bg-slate-100 text-slate-600',
                  // Saat memuat, angkanya diredupkan alih-alih diganti pemutar: pemutar
                  // yang muncul-hilang di enam tempat sekaligus membuat bilahnya berkedip.
                  loading ? 'opacity-50' : '',
                ].join(' ')}
              >
                {entry.jumlah}
              </span>
            </button>
          )
        })}
      </div>
    </div>
  )
}
