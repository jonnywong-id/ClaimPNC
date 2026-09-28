import { Fragment, useMemo, useState } from 'react'

import type { AutoClaimCompanySummary } from '@/api/types'
import { Paginator } from '@/components/DataTable'
import { ErrorMessage } from '@/components/ErrorMessage'
import { ChevronIcon, EmptyBoxIcon, SearchIcon } from '@/components/Icon'

import { useAutoClaimSummary } from './api'
import { CompanyBatches } from './CompanyBatches'
import { loadMessage } from './messages'

/** Banyaknya perusahaan per halaman — sama dengan ukuran halaman grid batch. */
const PAGE_SIZE = 15

type Props = {
  /** Tab yang sedang terbuka. */
  source: string
  /** Nama tabel sumber tab itu, untuk keterangan di kepala tabel. */
  table: string
}

/**
 * Tabel perusahaan Inbox Auto Claim — satu baris per perusahaan, dan baris yang diklik
 * MELAR ke bawah menampilkan daftar batch perusahaan itu.
 *
 * # Apa yang digantikannya
 *
 * Donut di kiri dan tabel jumlah di kanan (keputusan 2026-09-19) beserta grid batch
 * panjang di bawahnya. Work Owner meminta bentuk ini pada 2026-09-27: tanpa grafik, tabel
 * selebar layar, kolom Nama Perusahaan & Kode paling kiri, pencarian di atas, paginasi di
 * bawah, dan rincian batch terbuka DI TEMPAT — bukan jendela sembulan dan bukan grid
 * terpisah yang harus dicari dengan menggulir.
 *
 * Alasan yang disebut: tab Asuransi Kredit dapat memuat lebih dari 20 perusahaan, dan
 * donut dengan dua puluh irisan tidak lagi dapat dibaca maupun diklik.
 *
 * # Kenapa pencarian dan paginasinya di PERAMBAN
 *
 * Ringkasan dikirim server UTUH — satu baris per perusahaan yang punya batch di tab ini,
 * jumlahnya puluhan, bukan jutaan. Seluruh barisnya sudah di tangan, sehingga menyaring
 * di sini jujur: yang tidak ditemukan memang tidak ada. Larangan menyaring di peramban
 * (`DataTable`, `serverSearch`) berlaku untuk data yang hanya dipegang satu halamannya.
 *
 * Grid batch di DALAM baris tetap dipaginasi server — di sanalah datanya dapat besar.
 *
 * # Kenapa tabelnya ditulis di sini, bukan lewat DataTable
 *
 * DataTable tidak mengenal baris yang melar, dan menambahkannya menyentuh komponen yang
 * dipakai belasan layar yang sudah selesai. Panel ringkasan sebelumnya pun tabel lokal
 * modul ini; yang dipakai ulang adalah `Paginator`, supaya kaki tabelnya sama persis
 * dengan layar lain.
 */
export function CompanyTable({ source, table }: Props) {
  const summary = useAutoClaimSummary(source)

  const [query, setQuery] = useState('')
  const [page, setPage] = useState(1)
  // Satu baris terbuka sekaligus (accordion). Membuka beberapa sekaligus berarti beberapa
  // grid batch bertumpuk dengan kepala kolom yang sama — petugas kehilangan jejak batch
  // mana milik perusahaan mana.
  const [expanded, setExpanded] = useState<string | null>(null)

  const company = useMemo(() => summary.data?.perusahaan ?? [], [summary.data])

  const visible = useMemo(() => {
    const word = query.trim().toLowerCase()
    if (word === '') return company
    // Kode ikut dicari: petugas sering hanya ingat kode dari berkas unggahan, dan baris
    // tanpa master memang hanya punya kode.
    return company.filter(
      (c) => c.nama.toLowerCase().includes(word) || c.kode.toLowerCase().includes(word),
    )
  }, [company, query])

  const totalPages = Math.max(1, Math.ceil(visible.length / PAGE_SIZE))
  // Dijepit saat menggambar, sama seperti DataTable: daftar dapat menyusut setelah dimuat
  // ulang, dan halaman yang sudah tidak ada harus menampilkan halaman terakhir.
  const currentPage = Math.min(page, totalPages)
  const firstIndex = (currentPage - 1) * PAGE_SIZE
  const shown = visible.slice(firstIndex, firstIndex + PAGE_SIZE)

  function toggle(code: string) {
    setExpanded((previous) => (previous === code ? null : code))
  }

  return (
    <section className="mt-5 overflow-hidden rounded-kartu border border-slate-200 bg-white shadow-lembut">
      <header className="border-b border-slate-200 p-5">
        <h2 className="text-base font-semibold text-slate-900">Daftar Perusahaan</h2>
        {/* Nama tabelnya datang bersama tabnya: ketiga tab membaca tabel yang berbeda. */}
        <p className="mt-1 text-sm text-slate-600">Sumber: {table === '' ? '…' : table}</p>
      </header>

      <div className="border-b border-slate-200 bg-slate-50/60 px-5 py-4">
        <label htmlFor="cari-perusahaan" className="sr-only">
          Cari nama perusahaan
        </label>
        <div className="relative sm:max-w-sm">
          <span
            aria-hidden="true"
            className="pointer-events-none absolute inset-y-0 left-0 flex w-10 items-center justify-center text-slate-400"
          >
            <SearchIcon className="h-4 w-4" />
          </span>
          <input
            id="cari-perusahaan"
            type="search"
            value={query}
            onChange={(event) => {
              setQuery(event.target.value)
              // Kata kunci baru menghasilkan daftar lain; halaman lama hampir pasti tidak
              // ada di sana, dan baris yang terbuka mungkin ikut tersaring keluar.
              setPage(1)
              setExpanded(null)
            }}
            placeholder="Cari nama perusahaan atau kode…"
            className={[
              'w-full rounded-kontrol border border-slate-300 bg-white py-2.5 pl-10 pr-3',
              'text-sm text-slate-900 placeholder:text-slate-400',
              'transition-[border-color,box-shadow] duration-150 ease-halus hover:border-slate-400',
              'focus:border-blue-500 focus:outline-none focus:ring-4 focus:ring-blue-500/15',
            ].join(' ')}
          />
        </div>
        {query.trim() !== '' && summary.isSuccess && (
          <p className="mt-2 text-xs text-slate-600" role="status">
            {visible.length} dari {company.length} perusahaan cocok.
          </p>
        )}
      </div>

      {summary.isError ? (
        <div className="p-5">
          {(() => {
            const message = loadMessage(summary.error)
            return (
              <ErrorMessage
                title={message.title}
                description={message.description}
                tone={message.tone}
              />
            )
          })()}
        </div>
      ) : summary.isPending ? (
        <div className="space-y-3 p-5" role="status" aria-label="Memuat daftar perusahaan">
          {[0, 1, 2].map((n) => (
            <div key={n} className="h-10 animate-pulse rounded bg-slate-100" />
          ))}
        </div>
      ) : visible.length === 0 ? (
        <div className="flex flex-col items-center gap-3 px-5 py-14 text-center">
          <span className="flex h-12 w-12 items-center justify-center rounded-full bg-slate-100 text-slate-400">
            <EmptyBoxIcon className="h-6 w-6" />
          </span>
          <p className="text-sm font-medium text-slate-700">
            {query.trim() === ''
              ? 'Belum ada batch klaim pada tab ini.'
              : `Tidak ada perusahaan yang cocok dengan “${query.trim()}”.`}
          </p>
        </div>
      ) : (
        <div className="overflow-x-auto">
          <table aria-label="Daftar perusahaan" className="w-full border-collapse text-sm">
            <thead>
              <tr className="border-b border-slate-200 bg-slate-50/80 text-left">
                <th
                  scope="col"
                  className="px-5 py-3 text-xs font-semibold uppercase tracking-wide text-slate-600"
                >
                  Nama Perusahaan &amp; Kode
                </th>
                <th
                  scope="col"
                  className="w-32 px-5 py-3 text-right text-xs font-semibold uppercase tracking-wide text-slate-600"
                >
                  Jumlah Batch
                </th>
                <th
                  scope="col"
                  className="w-24 px-5 py-3 text-right text-xs font-semibold uppercase tracking-wide text-slate-600"
                >
                  Aksi
                </th>
              </tr>
            </thead>
            <tbody>
              {shown.map((c) => (
                <CompanyRow
                  key={c.kode}
                  company={c}
                  source={source}
                  open={expanded === c.kode}
                  onToggle={() => toggle(c.kode)}
                />
              ))}
            </tbody>
          </table>
        </div>
      )}

      {summary.isSuccess && visible.length > 0 && (
        <Paginator
          firstRow={firstIndex + 1}
          lastRow={firstIndex + shown.length}
          totalRows={visible.length}
          currentPage={currentPage}
          totalPages={totalPages}
          onPick={(next) => {
            setPage(next)
            // Baris yang terbuka tidak ada di halaman lain; membiarkannya "terbuka" di luar
            // pandangan membuat kembali ke halaman ini menampilkan grid yang tidak diminta.
            setExpanded(null)
          }}
        />
      )}
    </section>
  )
}

function CompanyRow({
  company,
  source,
  open,
  onToggle,
}: {
  company: AutoClaimCompanySummary
  source: string
  open: boolean
  onToggle: () => void
}) {
  const registered = company.nama !== ''
  const name = registered ? company.nama : company.kode
  const panelId = `batch-perusahaan-${company.kode}`

  return (
    <Fragment>
      {/* SELURUH BARIS dapat diklik — sorotan hover menjanjikannya, dan versi panel
          ringkasan yang hanya memasang penangan pada nama pernah dilaporkan sebagai cacat.
          Tombol di kolom Aksi tetap satu-satunya yang dapat difokus: pengguna papan ketik
          dan pembaca layar perlu target bernama beserta keadaan terbuka/tertutupnya. */}
      <tr
        onClick={onToggle}
        className={[
          'cursor-pointer border-b border-slate-100 transition-colors duration-150 ease-halus',
          open ? 'bg-blue-50/70' : 'hover:bg-slate-50',
        ].join(' ')}
      >
        <td className="px-5 py-3">
          <span
            className={['block', open ? 'font-semibold text-blue-900' : 'text-slate-900'].join(' ')}
          >
            {name}
          </span>
          <span className="block text-xs text-slate-500">
            {registered ? company.kode : 'tidak terdaftar di Master Auto Claim'}
          </span>
        </td>
        <td className="px-5 py-3 text-right">
          <span className="inline-flex min-w-8 justify-center rounded-full bg-slate-100 px-2.5 py-0.5 text-xs font-medium tabular-nums text-slate-800 ring-1 ring-slate-200 ring-inset">
            {company.jumlah_batch}
          </span>
        </td>
        <td className="px-5 py-3 text-right">
          <button
            type="button"
            onClick={(event) => {
              // Tanpa ini klik pada tombol ikut menjalankan penangan baris, dan barisnya
              // terbuka lalu langsung tertutup kembali.
              event.stopPropagation()
              onToggle()
            }}
            aria-expanded={open}
            aria-controls={open ? panelId : undefined}
            aria-label={`${open ? 'Tutup' : 'Buka'} batch ${name}`}
            className={[
              'inline-flex h-8 w-8 items-center justify-center rounded-kontrol text-slate-600',
              'transition-colors duration-150 ease-halus hover:bg-slate-200/70 hover:text-slate-900',
              'focus:outline-none focus-visible:ring-4 focus-visible:ring-blue-500/20',
            ].join(' ')}
          >
            <ChevronIcon
              className={[
                'h-4 w-4 transition-transform duration-150 ease-halus',
                open ? '-rotate-90' : 'rotate-90',
              ].join(' ')}
            />
          </button>
        </td>
      </tr>

      {open && (
        <tr className="border-b border-slate-200">
          <td id={panelId} colSpan={3} className="bg-slate-50/70 px-5 py-4">
            <CompanyBatches source={source} company={company.kode} companyName={company.nama} />
          </td>
        </tr>
      )}
    </Fragment>
  )
}
