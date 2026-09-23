import { useMemo, useState, type ReactNode } from 'react'

import { SearchIcon, EmptyBoxIcon } from './Icon'

/**
 * Satu kolom tabel.
 *
 * `nilai` mengambil isi sel dari satu baris, dan `tampil` menggambarnya. Keduanya
 * dipisah dengan sengaja: yang dicari dan diurutkan adalah `nilai` (teks polos),
 * sedangkan yang dilihat pengguna adalah `tampil` (boleh berisi lencana, tautan, atau
 * tombol). Menyatukannya akan membuat pencarian ikut menelusuri markup.
 */
export type Column<T> = {
  key: string
  title: string
  value: (rows: T) => string
  render?: (rows: T) => ReactNode

  /** Lebar kolom pada tampilan meja. Diabaikan pada tampilan kartu. */
  width?: string

  /** Kolom yang tidak layak diurutkan — misalnya kolom aksi. */
  noSort?: boolean

  /** Ratakan isi sel ke kanan pada tampilan meja. Dipakai kolom aksi. */
  alignRight?: boolean
}

/**
 * Kendali halaman yang dikerjakan SERVER, bukan peramban.
 *
 * Kehadirannya mengubah tiga hal sekaligus, dan ketiganya harus berubah bersamaan:
 * penyaringan internal dimatikan, pengurutan internal dimatikan, dan kaki tabel
 * menggambar nomor halaman.
 *
 * # Kenapa ketiganya, bukan salah satunya
 *
 * Karena tabel hanya memegang SATU halaman. Menyaring atau mengurutkan halaman itu
 * menghasilkan jawaban yang tampak benar tetapi dihitung dari sepersekian data —
 * "3 dari 10 baris cocok" padahal yang cocok di seluruh tabel ada 84. Itu bukan sekadar
 * kurang berguna; itu menyesatkan.
 *
 * Layar berdata besar — inbox dan laporan (`D-10`: puluhan juta baris) — WAJIB memakai
 * ini. Layar master yang barisnya puluhan tidak, dan karena itu seluruh isian di sini
 * opsional: tiga layar master yang sudah selesai tidak berubah perilakunya sama sekali.
 */
export type ServerPaging = {
  /** Halaman yang sedang tampil, dihitung mulai 1. */
  page: number
  pageSize: number

  /** Banyaknya baris yang cocok di SELURUH tabel, bukan di halaman ini. */
  total: number
  totalPages: number

  onPageChange: (page: number) => void
}

type Props<T> = {
  columns: Column<T>[]
  rows: T[]
  rowKey: (rows: T) => string

  /** Judul di atas tabel; boleh dikosongkan bila layar sudah punya judulnya sendiri. */
  title?: string
  description?: string

  /** Tombol-tombol di kanan judul — Tambah, Muat ulang, dan sejenisnya. */
  actions?: ReactNode

  isLoading?: boolean
  error?: ReactNode

  searchLabel?: string
  emptyMessage?: string

  /** Bila diisi, halaman dikendalikan server. Lihat ServerPaging. */
  serverPaging?: ServerPaging

  /**
   * Menyembunyikan kotak cari bawaan.
   *
   * Dipakai layar yang sudah punya panel saringnya sendiri — dua kotak cari pada satu
   * layar membuat pengguna menebak mana yang berlaku.
   */
  hideSearch?: boolean
}

type SortOrder = { key: string; direction: 'asc' | 'desc' }

/**
 * DataTable adalah satu-satunya tabel di seluruh aplikasi.
 *
 * # Kenapa ia ada
 *
 * 268 dari 269 section sistem lama memakai pola grid yang sama. Membuat tabelnya
 * sendiri-sendiri di tiap layar berarti 268 tafsir berbeda tentang bagaimana sebuah
 * tabel berperilaku saat kosong, saat memuat, dan saat gagal — dan itu persis mode
 * kegagalan yang dikhawatirkan `D-09` untuk tim yang sedang belajar React.
 *
 * # Kenapa tanpa pustaka tabel
 *
 * Pilihan antara TanStack Table dan AG Grid MASIH TERBUKA (`ADR-0002`, `TKT-U2-005`),
 * dan koreksi ukuran — median grid ternyata **6 kolom**, bukan 18–27 — melemahkan alasan
 * memilih pustaka kelas berat. Menarik salah satunya sekarang berarti mendahului
 * keputusan yang sengaja ditinggalkan terbuka untuk diambil dengan angka.
 *
 * Komponen ini karena itu dibuat sesempit mungkin dan hanya menyediakan yang benar-benar
 * dipakai layar master: pencarian, pengurutan, dan tiga keadaan tampilan. Bila kelak
 * pustaka dipilih, yang diganti adalah isi berkas ini — bukan setiap layar yang
 * memakainya.
 *
 * # Satu DOM, dua tampilan
 *
 * Pada layar sempit, tiap baris berubah menjadi kartu — tetapi markup-nya TETAP SATU
 * `<table>`. Menggambar meja dan kartu sebagai dua pohon terpisah lebih mudah ditulis,
 * dan itu yang mula-mula saya lakukan; akibatnya setiap isi sel ada dua kali di DOM,
 * pembaca layar membacanya dua kali, dan setiap pencarian di pengujian menemukan dua
 * elemen untuk satu nilai.
 *
 * Yang dipakai sebagai gantinya: elemen tabel diubah menjadi blok lewat CSS, dan nama
 * kolom digambar ulang di dalam sel sebagai label kecil yang hilang pada layar lebar.
 * Label itu `aria-hidden` karena `<th scope="col">` sudah menjelaskan selnya.
 *
 * # Kenapa pencarian dan pengurutan dikerjakan di peramban
 *
 * Bukan karena paginasi server tidak penting — `D-10` justru menetapkan sebaliknya untuk
 * data berpuluh juta baris. Master status berisi **33 baris** dan bertambah beberapa
 * baris per tahun. Menyaring 33 baris di server berarti satu perjalanan jaringan untuk
 * setiap huruf yang diketik, tanpa satu pun manfaat.
 *
 * Layar yang datanya besar — inbox dan laporan — TIDAK boleh memakai penyaringan ini;
 * mereka menuntut paginasi keyset dari server, dan itu lingkup `TKT-U2-001`.
 */
export function DataTable<T>({
  columns,
  rows,
  rowKey,
  title,
  description,
  actions,
  isLoading = false,
  error,
  searchLabel = 'Cari',
  emptyMessage = 'Belum ada data.',
  serverPaging,
  hideSearch = false,
}: Props<T>) {
  const [query, setQuery] = useState('')
  const [sort, setSort] = useState<SortOrder | null>(null)

  // Saat halaman dikendalikan server, tabel hanya memegang SATU halaman — menyaring dan
  // mengurutkannya di sini akan menjawab pertanyaan yang salah. Lihat ServerPaging.
  const local = serverPaging === undefined

  const visible = useMemo(() => {
    if (!local) return rows

    const word = query.trim().toLowerCase()
    const filtered = word
      ? rows.filter((b) => columns.some((k) => k.value(b).toLowerCase().includes(word)))
      : rows

    if (!sort) return filtered

    const sortColumn = columns.find((k) => k.key === sort.key)
    if (!sortColumn) return filtered

    // Salinan dibuat lebih dulu: sort mengubah senarai di tempat, dan mengurutkan props
    // secara langsung akan mengubah data milik pemanggil.
    return [...filtered].sort((a, b) => {
      const comparison = sortColumn.value(a).localeCompare(sortColumn.value(b), 'id', {
        numeric: true,
        sensitivity: 'base',
      })
      return sort.direction === 'asc' ? comparison : -comparison
    })
  }, [rows, columns, query, sort, local])

  function toggleSort(key: string) {
    setSort((previous) => {
      if (previous?.key !== key) return { key, direction: 'asc' }
      if (previous.direction === 'asc') return { key, direction: 'desc' }
      // Klik ketiga mengembalikan urutan asli dari server. Tanpa ini, pengguna tidak
      // punya cara kembali ke urutan semula selain memuat ulang halaman.
      return null
    })
  }

  const hasSearch = local && query.trim() !== ''

  return (
    <section className="overflow-hidden rounded-kartu border border-slate-200 bg-white shadow-lembut">
      {(title || actions) && (
        <header className="flex flex-col gap-4 border-b border-slate-200 bg-white p-5 sm:flex-row sm:items-start sm:justify-between">
          <div className="min-w-0">
            {title && <h2 className="text-base font-semibold text-slate-900">{title}</h2>}
            {description && <p className="mt-1 text-sm text-slate-600">{description}</p>}
          </div>
          {actions && <div className="flex flex-wrap items-center gap-2">{actions}</div>}
        </header>
      )}

      {!hideSearch && (
        <div className="border-b border-slate-200 bg-slate-50/60 px-5 py-4">
          <label htmlFor="tabel-cari" className="sr-only">
            {searchLabel}
          </label>
          <div className="relative sm:max-w-sm">
            <span
              aria-hidden="true"
              className="pointer-events-none absolute inset-y-0 left-0 flex w-10 items-center justify-center text-slate-400"
            >
              <SearchIcon className="h-4 w-4" />
            </span>
            <input
              id="tabel-cari"
              type="search"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder={searchLabel}
              className={[
                'w-full rounded-kontrol border border-slate-300 bg-white py-2.5 pl-10 pr-3',
                'text-sm text-slate-900 placeholder:text-slate-400',
                'transition-[border-color,box-shadow] duration-150 ease-halus',
                'hover:border-slate-400',
                'focus:border-blue-500 focus:outline-none focus:ring-4 focus:ring-blue-500/15',
              ].join(' ')}
            />
          </div>
          {hasSearch && (
            <p className="mt-2 text-xs text-slate-600" role="status">
              {visible.length} dari {rows.length} baris cocok.
            </p>
          )}
        </div>
      )}

      {/*
        Bilah halaman digambar DI ATAS tabel, seperti layar lama: jumlah seluruh baris dan
        nomor halaman berada di atas kepala kolom, bukan di bawah baris terakhir. Pada
        tabel berisi ribuan baris, menaruhnya di bawah berarti pengguna harus menggulung
        seluruh halaman hanya untuk berpindah halaman.
      */}
      {serverPaging && !error && !isLoading && (
        <PagingBar paging={serverPaging} shown={visible.length} />
      )}

      {error ? (
        <div className="p-5">{error}</div>
      ) : isLoading ? (
        <LoadingState />
      ) : visible.length === 0 ? (
        <EmptyState
          pesan={
            hasSearch
              ? `Tidak ada baris yang cocok dengan “${query.trim()}”.`
              : emptyMessage
          }
          saran={hasSearch ? 'Coba kata kunci yang lebih pendek.' : undefined}
        />
      ) : (
        <div className="md:overflow-x-auto">
          <table className="block w-full border-collapse text-sm md:table">
            <thead className="hidden md:table-header-group">
              <tr className="border-b border-slate-200 bg-slate-50/80 text-left">
                {columns.map((k) => (
                  <th
                    key={k.key}
                    scope="col"
                    style={k.width ? { width: k.width } : undefined}
                    aria-sort={
                      sort?.key === k.key
                        ? sort.direction === 'asc'
                          ? 'ascending'
                          : 'descending'
                        : 'none'
                    }
                    className={[
                      'px-5 py-3 text-xs font-semibold uppercase tracking-wide text-slate-600',
                      k.alignRight ? 'text-right' : '',
                    ].join(' ')}
                  >
                    {k.noSort || !local ? (
                      k.title
                    ) : (
                      <button
                        type="button"
                        onClick={() => toggleSort(k.key)}
                        className={[
                          'group -mx-1.5 inline-flex items-center gap-1.5 rounded px-1.5 py-1',
                          'transition-colors duration-150 ease-halus',
                          'hover:bg-slate-200/70 hover:text-slate-900',
                          'focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500/50',
                        ].join(' ')}
                      >
                        {k.title}
                        <SortMarker
                          active={sort?.key === k.key}
                          direction={sort?.direction ?? 'asc'}
                        />
                      </button>
                    )}
                  </th>
                ))}
              </tr>
            </thead>

            <tbody className="block md:table-row-group">
              {visible.map((b) => (
                <tr
                  key={rowKey(b)}
                  className={[
                    'block border-b border-slate-200 py-2.5 last:border-0',
                    'md:table-row md:border-slate-100 md:py-0',
                    'transition-colors duration-150 ease-halus',
                    'hover:bg-blue-50/50',
                  ].join(' ')}
                >
                  {columns.map((k) => (
                    <td
                      key={k.key}
                      className={[
                        'flex items-baseline gap-3 px-5 py-1.5',
                        'md:table-cell md:py-3.5 md:align-middle',
                        k.alignRight ? 'md:text-right' : '',
                      ].join(' ')}
                    >
                      {/*
                        Nama kolom digambar ulang di dalam sel untuk tampilan kartu.
                        aria-hidden karena <th scope="col"> sudah menjelaskan sel ini —
                        tanpa itu pembaca layar menyebut nama kolom dua kali.
                      */}
                      <span
                        aria-hidden="true"
                        className="w-28 shrink-0 text-xs font-medium uppercase tracking-wide text-slate-500 md:hidden"
                      >
                        {k.title}
                      </span>
                      <span className="min-w-0 flex-1 break-words text-slate-900">
                        {k.render ? k.render(b) : k.value(b) || '—'}
                      </span>
                    </td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

    </section>
  )
}

/**
 * Kaki tabel untuk halaman yang dikendalikan server.
 *
 * Yang ditulis bukan sekadar nomor halaman melainkan letaknya di dalam keseluruhan —
 * "11–20 dari 57". Pada data berpuluh juta baris, nomor halaman saja tidak memberi tahu
 * pengguna apakah yang ia cari masih jauh atau sudah terlewat.
 *
 * Tombolnya hanya Sebelumnya dan Berikutnya, bukan deretan nomor halaman. Deretan nomor
 * menuntut jumlah halaman yang bermakna, dan pada tabel yang barisnya terus bertambah
 * sementara pengguna membacanya, halaman ke-4.132 tidak menunjuk apa pun yang tetap.
 */
/**
 * Bilah halaman bergaya layar lama: `Total Data : 23767` lalu nomor-nomor halaman.
 *
 * # Kenapa nomor, bukan Sebelumnya/Berikutnya
 *
 * Layar lama menggambar nomor halaman, dan `D-13` menetapkan tata letak mengikutinya.
 * Pada daftar berisi ribuan halaman, nomor juga lebih berguna: satu klik memindahkan
 * lebih jauh daripada satu langkah.
 *
 * Jendela nomornya LIMA, bergeser mengikuti halaman yang sedang dibuka, sehingga halaman
 * mana pun tetap terjangkau dengan berjalan satu jendela setiap kali.
 */
function PagingBar({ paging }: { paging: ServerPaging; shown: number }) {
  const { page, total, totalPages, onPageChange } = paging

  const window = pageWindow(page, totalPages)

  return (
    <nav
      aria-label="Navigasi halaman"
      className="flex flex-col gap-3 border-b border-slate-200 bg-white px-5 py-3 sm:flex-row sm:items-center sm:justify-end"
    >
      {/*
        role="status" supaya pembaca layar mengumumkan perpindahan halaman. Tanpa itu,
        menekan nomor tidak menghasilkan satu pun umpan balik yang terdengar — isi tabel
        berganti di tempat, dan fokus tetap tinggal di tombol.
      */}
      <p className="text-xs text-slate-600 sm:mr-4" role="status">
        Total Data : <span className="font-medium text-slate-900">{total}</span>
        <span className="sr-only">
          , halaman {page} dari {totalPages}
        </span>
      </p>

      {totalPages > 1 && (
        <div className="flex flex-wrap items-center gap-1">
          {window.first > 1 && <PagingEllipsis />}
          {range(window.first, window.last).map((n) => (
            <PagingButton
              key={n}
              label={String(n)}
              current={n === page}
              onClick={() => onPageChange(n)}
            />
          ))}
          {window.last < totalPages && <PagingEllipsis />}
        </div>
      )}
    </nav>
  )
}

/** range mengembalikan deret bilangan dari first sampai last, inklusif. */
function range(first: number, last: number): number[] {
  const result: number[] = []
  for (let n = first; n <= last; n++) result.push(n)
  return result
}

/**
 * pageWindow memilih lima nomor halaman yang digambar.
 *
 * Ia bergeser mengikuti halaman yang sedang dibuka dan dijepit pada kedua ujungnya,
 * sehingga jendelanya selalu penuh selama halamannya cukup — tanpa itu, halaman pertama
 * dan terakhir menampilkan nomor lebih sedikit daripada halaman tengah.
 */
function pageWindow(page: number, totalPages: number): { first: number; last: number } {
  const width = 5
  if (totalPages <= width) return { first: 1, last: totalPages }

  const first = Math.min(Math.max(1, page - Math.floor(width / 2)), totalPages - width + 1)
  return { first, last: first + width - 1 }
}

/**
 * Penanda bahwa masih ada halaman di luar jendela.
 *
 * Ia sengaja BUKAN tombol. Layar lama menggambarnya, tetapi apa yang terjadi saat ditekan
 * tidak terlihat dari tangkapan layar — dan menebaknya berarti membuat perilaku yang
 * tidak dapat dirujuk ke mana pun. Halaman di luar jendela tetap terjangkau dengan
 * menekan nomor terjauh, yang menggeser jendelanya.
 */
function PagingEllipsis() {
  return (
    <span aria-hidden="true" className="px-1 text-xs text-slate-400">
      …
    </span>
  )
}

function PagingButton({
  label,
  disabled,
  current,
  onClick,
}: {
  label: string
  disabled?: boolean
  current?: boolean
  onClick: () => void
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={disabled}
      // aria-current menyatakan halaman yang sedang dibuka kepada pembaca layar. Warna
      // saja tidak menyampaikannya, dan pembedaan yang hanya warna dilarang sistem desain.
      aria-current={current ? 'page' : undefined}
      className={[
        'min-w-[2rem] rounded-kontrol border px-2.5 py-1 text-xs font-medium',
        'transition-[background-color,border-color,box-shadow] duration-150 ease-halus',
        'focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500/50',
        current
          ? 'border-blue-500 bg-blue-600 text-white'
          : 'border-slate-300 bg-white text-blue-700 hover:border-slate-400 hover:bg-slate-50',
        'disabled:cursor-not-allowed disabled:opacity-50',
      ].join(' ')}
    >
      {label}
    </button>
  )
}

/**
 * Keadaan memuat digambar sebagai kerangka baris, bukan tulisan "Memuat…" saja.
 *
 * Bentuknya meniru tabel yang akan muncul, sehingga tata letak tidak melompat saat data
 * tiba — dan pengguna melihat "sesuatu sedang datang", bukan layar yang tampak kosong.
 *
 * Teksnya tetap ada untuk pembaca layar lewat `role="status"`; denyutnya murni visual.
 */
function LoadingState() {
  return (
    <div className="divide-y divide-slate-100" role="status" aria-live="polite">
      <span className="sr-only">Memuat data…</span>
      {[0, 1, 2, 3].map((rows) => (
        <div key={rows} className="flex items-center gap-4 px-5 py-4">
          <div className="h-3.5 w-16 animate-pulse rounded bg-slate-200" />
          <div
            className="h-3.5 flex-1 animate-pulse rounded bg-slate-200"
            style={{ animationDelay: `${rows * 90}ms`, maxWidth: `${60 - rows * 6}%` }}
          />
          <div className="hidden h-3.5 w-14 animate-pulse rounded bg-slate-200 sm:block" />
        </div>
      ))}
    </div>
  )
}

function EmptyState({ pesan, saran }: { pesan: string; saran?: string | undefined }) {
  return (
    <div className="flex flex-col items-center gap-3 px-5 py-14 text-center">
      <span className="flex h-12 w-12 items-center justify-center rounded-full bg-slate-100 text-slate-400">
        <EmptyBoxIcon className="h-6 w-6" />
      </span>
      <p className="text-sm font-medium text-slate-700">{pesan}</p>
      {saran && <p className="text-sm text-slate-500">{saran}</p>}
    </div>
  )
}

/**
 * Penanda urutan punya tiga keadaan, dan ketiganya harus dapat dibedakan.
 *
 * Saat kolom tidak diurutkan, panah dua arah muncul samar dan menegas saat kolomnya
 * disentuh tetikus — itu yang memberi tahu bahwa judulnya dapat ditekan. Tanpa petunjuk
 * itu, pengguna tidak punya cara menduga tabelnya dapat diurutkan.
 */
function SortMarker({ active, direction }: { active: boolean; direction: 'asc' | 'desc' }) {
  if (!active) {
    return (
      <svg
        viewBox="0 0 12 12"
        aria-hidden="true"
        className="h-3 w-3 text-slate-300 transition-colors duration-150 ease-halus group-hover:text-slate-500"
      >
        <path d="m6 1.5 2.6 3H3.4L6 1.5Zm0 9L3.4 7.5h5.2L6 10.5Z" fill="currentColor" />
      </svg>
    )
  }
  return (
    <svg viewBox="0 0 12 12" aria-hidden="true" className="h-3 w-3 text-blue-600">
      {direction === 'asc' ? (
        <path d="m6 2 3.2 4H2.8L6 2Z" fill="currentColor" />
      ) : (
        <path d="M6 10 2.8 6h6.4L6 10Z" fill="currentColor" />
      )}
    </svg>
  )
}
