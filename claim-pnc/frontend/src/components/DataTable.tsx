import { useMemo, useState, type ReactNode } from 'react'

import { SearchIcon, EmptyBoxIcon, ChevronIcon } from './Icon'

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

  /**
   * Banyaknya baris per halaman. Tidak diisi berarti **tanpa paginasi** — seluruh baris
   * digambar sekaligus.
   *
   * # Kenapa opt-in, bukan bawaan
   *
   * Ukuran halaman di sistem lama **berbeda-beda per layar**, dan angkanya terbaca dari
   * `pyPageSize` (atau `pyPageSizeOther` bila nilainya `"Other"`) pada masing-masing
   * section:
   *
   *	Master Supplier · Master Bengkel                     20 baris
   *	Master Rekening · Status Klaim · Status Progres ·
   *	Pasal Kerugian · Penolakan Klaim                     15 baris
   *	Master Panel                                         15 baris tab Approve,
   *	                                                     50 baris tab Reject & Waiting
   *
   * Tidak ada satu angka yang benar untuk semuanya, sehingga ia milik layar — bukan milik
   * komponen ini.
   *
   * Master Panel bahkan berbeda ANTARTAB pada satu layar yang sama, sehingga angkanya
   * tidak dapat dititipkan ke layar sekali pun — ia milik tab. Itulah kenapa prop ini
   * menerima angka biasa, bukan dibaca komponen dari suatu tempat.
   *
   * Bawaannya **tanpa paginasi** supaya layar yang belum menyalakannya berperilaku persis
   * seperti sebelumnya. Menyalakannya untuk sebuah layar cukup satu prop.
   */
  pageSize?: number
}

type SortOrder = { key: string; direction: 'asc' | 'desc' }

/**
 * pageWindow memilih nomor halaman mana yang digambar sebagai tombol.
 *
 * Tujuh halaman atau kurang digambar seluruhnya. Lebih dari itu, yang digambar adalah
 * halaman pertama, terakhir, halaman sekarang beserta tetangganya, dan `sela` di antara
 * kelompok yang tidak bersambung.
 *
 * Tanpa pembatasan ini, daftar seribu baris menggambar lima puluh tombol nomor yang
 * membungkus beberapa baris — dan justru membuat halaman yang sedang dibuka sulit
 * ditemukan.
 */
export function pageWindow(current: number, total: number): (number | 'sela')[] {
  if (total <= 7) {
    return Array.from({ length: total }, (_, index) => index + 1)
  }

  const result: (number | 'sela')[] = [1]
  const start = Math.max(2, current - 1)
  const stop = Math.min(total - 1, current + 1)

  if (start > 2) result.push('sela')
  for (let nomor = start; nomor <= stop; nomor++) result.push(nomor)
  if (stop < total - 1) result.push('sela')

  result.push(total)
  return result
}

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
  pageSize,
}: Props<T>) {
  const [query, setQuery] = useState('')
  const [sort, setSort] = useState<SortOrder | null>(null)
  const [page, setPage] = useState(1)

  const visible = useMemo(() => {
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
  }, [rows, columns, query, sort])

  function toggleSort(key: string) {
    setSort((previous) => {
      if (previous?.key !== key) return { key, direction: 'asc' }
      if (previous.direction === 'asc') return { key, direction: 'desc' }
      // Klik ketiga mengembalikan urutan asli dari server. Tanpa ini, pengguna tidak
      // punya cara kembali ke urutan semula selain memuat ulang halaman.
      return null
    })
    // Mengurutkan ulang menyusun ulang seluruh daftar, sehingga halaman ketujuh yang
    // sedang dibuka tidak lagi memuat baris yang sama. Kembali ke halaman pertama adalah
    // satu-satunya posisi yang artinya tidak berubah.
    setPage(1)
  }

  const hasSearch = query.trim() !== ''

  /*
    Paginasi dikerjakan SESUDAH pencarian dan pengurutan, bukan sebelumnya.

    Itu yang ditiru dari sistem lama: grid Pega terikat pada page list klipboard
    (`pyPageListProperty`), sehingga paginatornya memotong daftar yang SUDAH tersaring —
    bukan meminta halaman berikutnya ke server. Memotong lebih dulu akan membuat pencarian
    hanya menemukan baris yang kebetulan ada di halaman yang sedang dibuka.

    Ia juga bukan paginasi keyset sisi server. Itu `TKT-U2-001`, dan Steering menyebutnya
    **perubahan perilaku, bukan pemeliharaan** — layar berpuluh juta baris menuntutnya,
    layar master yang berbaris puluhan tidak.
  */
  const paginated = pageSize !== undefined && pageSize > 0
  const totalPages = paginated ? Math.max(1, Math.ceil(visible.length / pageSize)) : 1

  // Halaman dijepit saat menggambar, bukan disetel lewat efek. Baris dapat berkurang di
  // luar kendali komponen ini — penyaring dipersempit, atau daftarnya dimuat ulang setelah
  // sebuah baris berpindah — dan halaman yang sudah tidak ada harus menampilkan halaman
  // terakhir, bukan tabel kosong tanpa penjelasan.
  const currentPage = Math.min(page, totalPages)
  const firstIndex = paginated ? (currentPage - 1) * pageSize : 0
  const shown = paginated ? visible.slice(firstIndex, firstIndex + pageSize) : visible

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
            onChange={(e) => {
              setQuery(e.target.value)
              // Kata kunci baru menghasilkan daftar yang berbeda, dan halaman ketujuh
              // daftar lama hampir pasti tidak ada pada daftar baru.
              setPage(1)
            }}
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
                    {k.noSort ? (
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
              {shown.map((b) => (
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

      {/*
        Paginator digambar hanya bila paginasinya menyala DAN ada baris yang terlihat.
        Saat memuat, saat gagal, dan saat kosong ia tidak berarti apa-apa — dan tiga
        keadaan itu sudah punya tampilannya sendiri di atas.
      */}
      {paginated && !error && !isLoading && visible.length > 0 && (
        <Paginator
          firstRow={firstIndex + 1}
          lastRow={firstIndex + shown.length}
          totalRows={visible.length}
          currentPage={currentPage}
          totalPages={totalPages}
          onPick={setPage}
        />
      )}
    </section>
  )
}

/**
 * Paginator adalah pemilih halaman di kaki tabel.
 *
 * # Bentuknya mengikuti paginator Pega, bukan dikarang
 *
 * `Section/InboxMasterSupplier-Section.xml` menyisipkan `pyGridPaginator` di kaki grid
 * dengan `pyPageMode = Numeric` dan `pyPaginationButtonsFormat = Standard` — **nomor
 * halaman**, bukan tombol "muat lebih banyak". Gaya selnya
 * (`dataLabelRead gridActionAlignRight`) menaruhnya **rata kanan**, dan itu yang ditiru.
 *
 * # Ringkasan barisnya DITAMBAHKAN
 *
 * Pega hanya menggambar nomor halamannya. "Menampilkan 1–20 dari 57 baris" tidak ada di
 * sana, dan ia ditambahkan karena tanpa itu nomor halaman tidak memberi tahu apa pun
 * tentang seberapa banyak yang belum dilihat — pada tabel yang baru saja disaring,
 * itulah justru yang ingin diketahui.
 *
 * Pencacah "Total Data :" milik `Section/DataCountMasterSupllier` tetap ada di kepala
 * layar dan menghitung hal yang berbeda: seluruh baris yang dimuat, bukan yang tersaring.
 *
 * # Yang dijaga untuk pembaca layar
 *
 * Ringkasannya `role="status"`, sehingga berpindah halaman diumumkan tanpa memindahkan
 * fokus. Nomor halaman yang sedang dibuka memakai `aria-current="page"` — bukan hanya
 * warna, yang tidak terbaca pembaca layar dan tidak terbedakan oleh sekitar satu dari dua
 * belas laki-laki yang mengalami buta warna merah-hijau.
 */
function Paginator({
  firstRow,
  lastRow,
  totalRows,
  currentPage,
  totalPages,
  onPick,
}: {
  firstRow: number
  lastRow: number
  totalRows: number
  currentPage: number
  totalPages: number
  onPick: (page: number) => void
}) {
  const step =
    'inline-flex h-8 min-w-8 items-center justify-center rounded-kontrol border px-2 text-sm ' +
    'transition-colors duration-150 ease-halus ' +
    'focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500/50 ' +
    'disabled:cursor-not-allowed disabled:opacity-40'

  return (
    <div className="flex flex-col gap-3 border-t border-slate-200 bg-slate-50/60 px-5 py-3 sm:flex-row sm:items-center sm:justify-between">
      <p className="text-xs text-slate-600" role="status">
        Menampilkan{' '}
        <span className="font-medium text-slate-800">
          {firstRow}–{lastRow}
        </span>{' '}
        dari {totalRows} baris.
      </p>

      {/*
        Tombolnya disembunyikan saat halamannya hanya satu. Paginator berisi satu tombol
        yang tidak dapat ditekan tidak memberi tahu apa pun; ringkasan barisnya tetap
        berguna dan tetap tampil.
      */}
      {totalPages > 1 && (
        <nav aria-label="Halaman tabel" className="flex flex-wrap items-center gap-1">
          <button
            type="button"
            onClick={() => onPick(currentPage - 1)}
            disabled={currentPage === 1}
            aria-label="Halaman sebelumnya"
            className={`${step} border-slate-300 bg-white text-slate-700 hover:enabled:border-slate-400 hover:enabled:bg-slate-100`}
          >
            <ChevronIcon className="h-4 w-4 rotate-180" />
          </button>

          {pageWindow(currentPage, totalPages).map((item, index) =>
            item === 'sela' ? (
              <span
                // Sela tidak punya nilai yang dapat dijadikan kunci, dan dua di antaranya
                // dapat muncul bersamaan — indeksnya yang membedakan.
                key={`sela-${index}`}
                aria-hidden="true"
                className="px-1 text-sm text-slate-400"
              >
                …
              </span>
            ) : (
              <button
                key={item}
                type="button"
                onClick={() => onPick(item)}
                aria-label={`Halaman ${item}`}
                aria-current={item === currentPage ? 'page' : undefined}
                className={[
                  step,
                  item === currentPage
                    ? 'border-blue-600 bg-blue-600 font-medium text-white'
                    : 'border-slate-300 bg-white text-slate-700 hover:border-slate-400 hover:bg-slate-100',
                ].join(' ')}
              >
                {item}
              </button>
            ),
          )}

          <button
            type="button"
            onClick={() => onPick(currentPage + 1)}
            disabled={currentPage === totalPages}
            aria-label="Halaman berikutnya"
            className={`${step} border-slate-300 bg-white text-slate-700 hover:enabled:border-slate-400 hover:enabled:bg-slate-100`}
          >
            <ChevronIcon className="h-4 w-4" />
          </button>
        </nav>
      )}
    </div>
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
