import { useMemo, useState, type ReactNode } from 'react'

import { IkonCari, IkonKotakKosong } from './Ikon'

/**
 * Satu kolom tabel.
 *
 * `nilai` mengambil isi sel dari satu baris, dan `tampil` menggambarnya. Keduanya
 * dipisah dengan sengaja: yang dicari dan diurutkan adalah `nilai` (teks polos),
 * sedangkan yang dilihat pengguna adalah `tampil` (boleh berisi lencana, tautan, atau
 * tombol). Menyatukannya akan membuat pencarian ikut menelusuri markup.
 */
export type Kolom<T> = {
  kunci: string
  judul: string
  nilai: (baris: T) => string
  tampil?: (baris: T) => ReactNode

  /** Lebar kolom pada tampilan meja. Diabaikan pada tampilan kartu. */
  lebar?: string

  /** Kolom yang tidak layak diurutkan — misalnya kolom aksi. */
  tanpaUrut?: boolean

  /** Ratakan isi sel ke kanan pada tampilan meja. Dipakai kolom aksi. */
  keKanan?: boolean
}

type Props<T> = {
  kolom: Kolom<T>[]
  baris: T[]
  kunciBaris: (baris: T) => string

  /** Judul di atas tabel; boleh dikosongkan bila layar sudah punya judulnya sendiri. */
  judul?: string
  keterangan?: string

  /** Tombol-tombol di kanan judul — Tambah, Muat ulang, dan sejenisnya. */
  aksi?: ReactNode

  sedangMemuat?: boolean
  galat?: ReactNode

  labelCari?: string
  pesanKosong?: string
}

type Urutan = { kunci: string; arah: 'naik' | 'turun' }

/**
 * TabelData adalah satu-satunya tabel di seluruh aplikasi.
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
export function TabelData<T>({
  kolom,
  baris,
  kunciBaris,
  judul,
  keterangan,
  aksi,
  sedangMemuat = false,
  galat,
  labelCari = 'Cari',
  pesanKosong = 'Belum ada data.',
}: Props<T>) {
  const [cari, setCari] = useState('')
  const [urutan, setUrutan] = useState<Urutan | null>(null)

  const terlihat = useMemo(() => {
    const kata = cari.trim().toLowerCase()
    const disaring = kata
      ? baris.filter((b) => kolom.some((k) => k.nilai(b).toLowerCase().includes(kata)))
      : baris

    if (!urutan) return disaring

    const kolomUrut = kolom.find((k) => k.kunci === urutan.kunci)
    if (!kolomUrut) return disaring

    // Salinan dibuat lebih dulu: sort mengubah senarai di tempat, dan mengurutkan props
    // secara langsung akan mengubah data milik pemanggil.
    return [...disaring].sort((a, b) => {
      const perbandingan = kolomUrut.nilai(a).localeCompare(kolomUrut.nilai(b), 'id', {
        numeric: true,
        sensitivity: 'base',
      })
      return urutan.arah === 'naik' ? perbandingan : -perbandingan
    })
  }, [baris, kolom, cari, urutan])

  function gantiUrutan(kunci: string) {
    setUrutan((sebelumnya) => {
      if (sebelumnya?.kunci !== kunci) return { kunci, arah: 'naik' }
      if (sebelumnya.arah === 'naik') return { kunci, arah: 'turun' }
      // Klik ketiga mengembalikan urutan asli dari server. Tanpa ini, pengguna tidak
      // punya cara kembali ke urutan semula selain memuat ulang halaman.
      return null
    })
  }

  const adaPencarian = cari.trim() !== ''

  return (
    <section className="overflow-hidden rounded-kartu border border-slate-200 bg-white shadow-lembut">
      {(judul || aksi) && (
        <header className="flex flex-col gap-4 border-b border-slate-200 bg-white p-5 sm:flex-row sm:items-start sm:justify-between">
          <div className="min-w-0">
            {judul && <h2 className="text-base font-semibold text-slate-900">{judul}</h2>}
            {keterangan && <p className="mt-1 text-sm text-slate-600">{keterangan}</p>}
          </div>
          {aksi && <div className="flex flex-wrap items-center gap-2">{aksi}</div>}
        </header>
      )}

      <div className="border-b border-slate-200 bg-slate-50/60 px-5 py-4">
        <label htmlFor="tabel-cari" className="sr-only">
          {labelCari}
        </label>
        <div className="relative sm:max-w-sm">
          <span
            aria-hidden="true"
            className="pointer-events-none absolute inset-y-0 left-0 flex w-10 items-center justify-center text-slate-400"
          >
            <IkonCari className="h-4 w-4" />
          </span>
          <input
            id="tabel-cari"
            type="search"
            value={cari}
            onChange={(e) => setCari(e.target.value)}
            placeholder={labelCari}
            className={[
              'w-full rounded-kontrol border border-slate-300 bg-white py-2.5 pl-10 pr-3',
              'text-sm text-slate-900 placeholder:text-slate-400',
              'transition-[border-color,box-shadow] duration-150 ease-halus',
              'hover:border-slate-400',
              'focus:border-blue-500 focus:outline-none focus:ring-4 focus:ring-blue-500/15',
            ].join(' ')}
          />
        </div>
        {adaPencarian && (
          <p className="mt-2 text-xs text-slate-600" role="status">
            {terlihat.length} dari {baris.length} baris cocok.
          </p>
        )}
      </div>

      {galat ? (
        <div className="p-5">{galat}</div>
      ) : sedangMemuat ? (
        <KeadaanMemuat />
      ) : terlihat.length === 0 ? (
        <KeadaanKosong
          pesan={
            adaPencarian
              ? `Tidak ada baris yang cocok dengan “${cari.trim()}”.`
              : pesanKosong
          }
          saran={adaPencarian ? 'Coba kata kunci yang lebih pendek.' : undefined}
        />
      ) : (
        <div className="md:overflow-x-auto">
          <table className="block w-full border-collapse text-sm md:table">
            <thead className="hidden md:table-header-group">
              <tr className="border-b border-slate-200 bg-slate-50/80 text-left">
                {kolom.map((k) => (
                  <th
                    key={k.kunci}
                    scope="col"
                    style={k.lebar ? { width: k.lebar } : undefined}
                    aria-sort={
                      urutan?.kunci === k.kunci
                        ? urutan.arah === 'naik'
                          ? 'ascending'
                          : 'descending'
                        : 'none'
                    }
                    className={[
                      'px-5 py-3 text-xs font-semibold uppercase tracking-wide text-slate-600',
                      k.keKanan ? 'text-right' : '',
                    ].join(' ')}
                  >
                    {k.tanpaUrut ? (
                      k.judul
                    ) : (
                      <button
                        type="button"
                        onClick={() => gantiUrutan(k.kunci)}
                        className={[
                          'group -mx-1.5 inline-flex items-center gap-1.5 rounded px-1.5 py-1',
                          'transition-colors duration-150 ease-halus',
                          'hover:bg-slate-200/70 hover:text-slate-900',
                          'focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500/50',
                        ].join(' ')}
                      >
                        {k.judul}
                        <PenandaUrutan
                          aktif={urutan?.kunci === k.kunci}
                          arah={urutan?.arah ?? 'naik'}
                        />
                      </button>
                    )}
                  </th>
                ))}
              </tr>
            </thead>

            <tbody className="block md:table-row-group">
              {terlihat.map((b) => (
                <tr
                  key={kunciBaris(b)}
                  className={[
                    'block border-b border-slate-200 py-2.5 last:border-0',
                    'md:table-row md:border-slate-100 md:py-0',
                    'transition-colors duration-150 ease-halus',
                    'hover:bg-blue-50/50',
                  ].join(' ')}
                >
                  {kolom.map((k) => (
                    <td
                      key={k.kunci}
                      className={[
                        'flex items-baseline gap-3 px-5 py-1.5',
                        'md:table-cell md:py-3.5 md:align-middle',
                        k.keKanan ? 'md:text-right' : '',
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
                        {k.judul}
                      </span>
                      <span className="min-w-0 flex-1 break-words text-slate-900">
                        {k.tampil ? k.tampil(b) : k.nilai(b) || '—'}
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
 * Keadaan memuat digambar sebagai kerangka baris, bukan tulisan "Memuat…" saja.
 *
 * Bentuknya meniru tabel yang akan muncul, sehingga tata letak tidak melompat saat data
 * tiba — dan pengguna melihat "sesuatu sedang datang", bukan layar yang tampak kosong.
 *
 * Teksnya tetap ada untuk pembaca layar lewat `role="status"`; denyutnya murni visual.
 */
function KeadaanMemuat() {
  return (
    <div className="divide-y divide-slate-100" role="status" aria-live="polite">
      <span className="sr-only">Memuat data…</span>
      {[0, 1, 2, 3].map((baris) => (
        <div key={baris} className="flex items-center gap-4 px-5 py-4">
          <div className="h-3.5 w-16 animate-pulse rounded bg-slate-200" />
          <div
            className="h-3.5 flex-1 animate-pulse rounded bg-slate-200"
            style={{ animationDelay: `${baris * 90}ms`, maxWidth: `${60 - baris * 6}%` }}
          />
          <div className="hidden h-3.5 w-14 animate-pulse rounded bg-slate-200 sm:block" />
        </div>
      ))}
    </div>
  )
}

function KeadaanKosong({ pesan, saran }: { pesan: string; saran?: string | undefined }) {
  return (
    <div className="flex flex-col items-center gap-3 px-5 py-14 text-center">
      <span className="flex h-12 w-12 items-center justify-center rounded-full bg-slate-100 text-slate-400">
        <IkonKotakKosong className="h-6 w-6" />
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
function PenandaUrutan({ aktif, arah }: { aktif: boolean; arah: 'naik' | 'turun' }) {
  if (!aktif) {
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
      {arah === 'naik' ? (
        <path d="m6 2 3.2 4H2.8L6 2Z" fill="currentColor" />
      ) : (
        <path d="M6 10 2.8 6h6.4L6 10Z" fill="currentColor" />
      )}
    </svg>
  )
}
