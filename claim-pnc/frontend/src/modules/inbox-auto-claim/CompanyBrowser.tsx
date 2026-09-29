import { useMemo, useState } from 'react'

import type { AutoClaimCompanySummary } from '@/api/types'
import { ErrorMessage } from '@/components/ErrorMessage'
import { EmptyBoxIcon, SearchIcon } from '@/components/Icon'

import { useAutoClaimSummary } from './api'
import { CompanyBatches } from './CompanyBatches'
import { loadMessage } from './messages'

type Props = {
  /** Tab yang sedang terbuka. */
  source: string
  /** Nama tabel sumber tab itu, untuk keterangan di kepala panel kanan. */
  table: string
}

/**
 * Daftar perusahaan di KIRI, batch perusahaan terpilih di KANAN (master–detail).
 *
 * # Apa yang digantikannya
 *
 * Tabel perusahaan dengan baris melar (accordion, keputusan 2026-09-27). Work Owner
 * meminta bentuk ini pada 2026-09-28 dengan contoh tampilan "kategori di kiri, daftar di
 * kanan": daftar perusahaan tetap terlihat selama batch diperiksa, sehingga berpindah
 * perusahaan cukup satu klik — pada accordion, membuka satu baris mendorong perusahaan
 * berikutnya keluar layar.
 *
 * # Tiga keputusan kecil
 *
 * - **Perusahaan pertama langsung terpilih** saat tab dibuka, seperti contohnya. Panel
 *   kanan yang kosong sampai sesuatu diklik tidak memberi tahu apa pun.
 * - **Daftar kiri digulir, tidak dipaginasi**, seperti contohnya. Isinya puluhan baris dan
 *   dikirim server utuh; pencarian di atasnya yang mempersempit. Grid batch di kanan tetap
 *   dipaginasi server — di sanalah datanya dapat besar.
 * - **Pilihan dipertahankan saat mencari.** Kata kunci hanya menyaring daftar kiri; batch
 *   yang sedang diperiksa di kanan tidak ikut hilang hanya karena namanya tersaring keluar.
 */
export function CompanyBrowser({ source, table }: Props) {
  const summary = useAutoClaimSummary(source)

  const [query, setQuery] = useState('')
  const [picked, setPicked] = useState<string | null>(null)

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

  // Yang terpilih: pilihan pengguna bila masih ada di data, selain itu perusahaan pertama.
  // Dihitung saat menggambar, bukan disetel lewat efek, supaya tidak ada satu gambar pun
  // dengan panel kanan kosong di antara data tiba dan efek berjalan.
  const selected =
    company.find((c) => c.kode === picked) ?? (picked === null ? company[0] : undefined)

  if (summary.isError) {
    const message = loadMessage(summary.error)
    return (
      <div className="mt-5">
        <ErrorMessage title={message.title} description={message.description} tone={message.tone} />
      </div>
    )
  }

  return (
    <div className="mt-5 grid gap-4 lg:grid-cols-[18rem_minmax(0,1fr)]">
      <aside className="flex flex-col overflow-hidden rounded-kartu border border-slate-200 bg-white shadow-lembut lg:max-h-[calc(100vh-12rem)]">
        <div className="flex items-center justify-between px-4 pt-4">
          <h2 className="text-xs font-semibold uppercase tracking-wide text-slate-600">
            Perusahaan
          </h2>
          <span className="text-sm font-semibold tabular-nums text-slate-700">
            {summary.isSuccess ? company.length : '…'}
          </span>
        </div>

        <div className="px-4 py-3">
          <label htmlFor="cari-perusahaan" className="sr-only">
            Cari nama perusahaan
          </label>
          <div className="relative">
            <span
              aria-hidden="true"
              className="pointer-events-none absolute inset-y-0 left-0 flex w-9 items-center justify-center text-slate-400"
            >
              <SearchIcon className="h-4 w-4" />
            </span>
            <input
              id="cari-perusahaan"
              type="search"
              value={query}
              onChange={(event) => setQuery(event.target.value)}
              placeholder="Cari perusahaan atau kode…"
              className={[
                'w-full rounded-kontrol border border-slate-300 bg-white py-2 pl-9 pr-3',
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

        {summary.isPending ? (
          <div className="space-y-2 px-4 pb-4" role="status" aria-label="Memuat daftar perusahaan">
            {[0, 1, 2, 3].map((n) => (
              <div key={n} className="h-9 animate-pulse rounded bg-slate-100" />
            ))}
          </div>
        ) : visible.length === 0 ? (
          <p className="px-4 pb-4 text-sm text-slate-600">
            {query.trim() === ''
              ? 'Belum ada batch klaim pada tab ini.'
              : `Tidak ada perusahaan yang cocok dengan “${query.trim()}”.`}
          </p>
        ) : (
          <nav
            aria-label="Daftar perusahaan"
            className="max-h-80 overflow-y-auto pb-2 lg:max-h-none"
          >
            <ul>
              {visible.map((c) => (
                <li key={c.kode}>
                  <CompanyItem
                    company={c}
                    active={selected?.kode === c.kode}
                    onPick={() => setPicked(c.kode)}
                  />
                </li>
              ))}
            </ul>
          </nav>
        )}
      </aside>

      <section
        aria-label="Batch perusahaan terpilih"
        className="min-w-0 rounded-kartu border border-slate-200 bg-white p-4 shadow-lembut sm:p-5"
      >
        {selected === undefined ? (
          <div className="flex flex-col items-center gap-3 py-14 text-center">
            <span className="flex h-12 w-12 items-center justify-center rounded-full bg-slate-100 text-slate-400">
              <EmptyBoxIcon className="h-6 w-6" />
            </span>
            <p className="text-sm font-medium text-slate-700">
              {summary.isPending ? 'Memuat…' : 'Pilih perusahaan di sebelah kiri.'}
            </p>
          </div>
        ) : (
          <>
            <header className="mb-4 flex flex-wrap items-baseline gap-x-3 gap-y-1">
              <h2 className="text-lg font-semibold text-slate-900">
                {selected.nama === '' ? selected.kode : selected.nama}
              </h2>
              <span className="text-sm text-slate-600">
                {selected.nama === '' ? 'tidak terdaftar di Master Auto Claim' : selected.kode}
                {' · '}
                <span className="tabular-nums">{selected.jumlah_batch}</span> batch
              </span>
              {/* Nama tabelnya datang bersama tabnya: ketiga tab membaca tabel berbeda. */}
              <span className="w-full text-xs text-slate-500">
                Sumber: {table === '' ? '…' : table}
              </span>
            </header>
            {/* `key` memasang ulang grid saat perusahaan berganti: halaman dan rincian
                yang terbuka milik perusahaan sebelumnya tidak boleh terbawa. */}
            <CompanyBatches
              key={selected.kode}
              source={source}
              company={selected.kode}
              companyName={selected.nama}
            />
          </>
        )}
      </section>
    </div>
  )
}

/**
 * Satu baris daftar perusahaan.
 *
 * Tombol, bukan tautan: memilih perusahaan tidak berpindah halaman. Yang terpilih ditandai
 * GARIS di tepi kiri dan `aria-current`, bukan warna latar saja — pembedaan penting tidak
 * pernah hanya warna, dan pembaca layar tidak melihat warna.
 */
function CompanyItem({
  company,
  active,
  onPick,
}: {
  company: AutoClaimCompanySummary
  active: boolean
  onPick: () => void
}) {
  const registered = company.nama !== ''
  return (
    <button
      type="button"
      onClick={onPick}
      aria-current={active ? 'true' : undefined}
      className={[
        'flex w-full items-center gap-3 border-l-[3px] py-2 pl-[13px] pr-4 text-left',
        'transition-colors duration-150 ease-halus',
        'focus:outline-none focus-visible:bg-blue-50 focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-blue-500/40',
        active ? 'border-blue-600 bg-blue-50' : 'border-transparent hover:bg-slate-50',
      ].join(' ')}
    >
      <span className="min-w-0 flex-1">
        <span
          className={[
            'block truncate text-sm',
            active ? 'font-semibold text-slate-900' : 'text-slate-800',
          ].join(' ')}
        >
          {registered ? company.nama : company.kode}
        </span>
        <span className="block truncate text-xs text-slate-500">
          {registered ? company.kode : 'tidak terdaftar di Master Auto Claim'}
        </span>
      </span>
      <span
        className={[
          'shrink-0 text-sm tabular-nums',
          active ? 'font-semibold text-slate-900' : 'text-slate-700',
        ].join(' ')}
      >
        {company.jumlah_batch}
      </span>
    </button>
  )
}
