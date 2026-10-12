import { useState, type ReactNode } from 'react'

import { APIError } from '@/api/client'
import { Button } from '@/components/Button'
import { DataTable, type Column as TableColumn } from '@/components/DataTable'
import { ErrorMessage } from '@/components/ErrorMessage'
import { SelectField } from '@/components/SelectField'
import { ChevronIcon } from '@/components/Icon'
import { formatDate } from '@/components/format'

import { useProgressClaimList } from './api'
import {
  EMPTY_FILTER,
  type BusinessLine,
  type ClaimRow,
  type Column,
  type FilterForm,
  type PageInfo,
  type Row,
  type Section,
} from './types'

type Props = {
  section: Section
  lines: BusinessLine[]

  /** Bagian ini terbuka saat layar pertama dibuka. */
  openByDefault: boolean

  /** Metadata sudah diterima; sebelum itu tidak ada yang layak diminta. */
  ready: boolean

  onOpenClaim: (claimNumber: string) => void
}

/**
 * Satu bagian layar Inbox Progress Claim.
 *
 * # Kenapa bagian, bukan tab
 *
 * Karena begitulah bentuknya di Pega. `Section/ProgressClaim_Section-Section.xml` menumpuk
 * kelima bagiannya dalam satu halaman sebagai kontainer berjudul, bukan sebagai bilah tab —
 * berbeda dari Inbox Admin, yang memang bertab. `D-13` menetapkan tampilan meniru Pega.
 *
 * # Kenapa isinya baru dimuat saat dibuka
 *
 * Karena sistem lama pun begitu: tiap kontainer memasang
 * `pyDeferLoadRetrievalActivity` sendiri, sehingga kuerinya berjalan saat bagiannya
 * ditampilkan. Alasannya bukan kesetiaan semata — rekap per PIC menghitung lima subkueri
 * agregat, dan membayarnya untuk bagian yang tidak dilihat adalah pemborosan yang terasa.
 */
export function ProgressSection({
  section,
  lines,
  openByDefault,
  ready,
  onOpenClaim,
}: Props) {
  const [open, setOpen] = useState(openByDefault)
  const [filter, setFilter] = useState<FilterForm>(EMPTY_FILTER)
  const [page, setPage] = useState(1)

  /**
   * Isi kotak cari yang sedang DIKETIK, terpisah dari kata kunci yang sudah dikirim.
   *
   * Keduanya dipisah karena setiap ketikan menembak basis data. Jedanya dipasang di
   * `commitSearch`, dipicu saat pengguna menekan tombol Cari atau Enter — bukan per huruf,
   * karena kotak cari di sistem lama pun baru menyaring setelah tombol "Cari Data"
   * ditekan.
   */
  const [draft, setDraft] = useState('')

  /**
   * Bagian yang memang kosong di sistem lama TIDAK meminta apa pun ke server.
   *
   * Backend memang menjawabnya tanpa menyentuh basis data, tetapi perjalanan jaringannya
   * tetap sia-sia: jawabannya sudah pasti kosong, dan bentuk layarnya sudah diketahui
   * dari metadata.
   */
  const empty = section.bentuk === 'kosong'

  const list = useProgressClaimList(section.kode, filter, page, ready && open && !empty)

  function commitSearch() {
    setFilter((previous) => ({ ...previous, cari: draft }))
    setPage(1)
  }

  function changeFilter(patch: Partial<FilterForm>) {
    setFilter((previous) => ({ ...previous, ...patch }))
    setPage(1)
  }

  function clearFilter() {
    setFilter(EMPTY_FILTER)
    setDraft('')
    setPage(1)
  }

  return (
    <section className="mt-6 rounded-kartu border border-slate-200">
      <SectionHeader
        section={section}
        open={open}
        onToggle={() => setOpen((previous) => !previous)}
      />

      {open && (
        <div className="border-t border-slate-200 px-4 py-4">
          {empty ? (
            <EmptyRegionNote />
          ) : (
            <>
              <FilterBar
                section={section}
                lines={lines}
                filter={filter}
                search={draft}
                onSearch={setDraft}
                onCommitSearch={commitSearch}
                onChange={changeFilter}
                onClear={clearFilter}
              />

              <DeadControlNotes section={section} />

              <div className="mt-4">
                <DataTable<Row>
                  columns={columnsFor(section, onOpenClaim)}
                  rows={list.data?.baris ?? []}
                  rowKey={rowKeyFor(section)}
                  // Kotak cari bawaan disembunyikan: bagian ini punya penyaringnya
                  // sendiri di atas, dan yang kedua hanya akan menyaring halaman yang
                  // sedang terbuka — hasilnya menyesatkan pada data berhalaman.
                  hideSearch
                  isLoading={list.isPending}
                  error={
                    list.isError ? (
                      <ErrorMessage
                        title={`Bagian ${section.nama} tidak dapat dimuat`}
                        description={messageOf(list.error)}
                        tone="gangguan"
                      />
                    ) : undefined
                  }
                  emptyMessage={emptyMessageFor(section, filter)}
                />

                {section.pakai_paginasi &&
                  list.data &&
                  list.data.paginasi.total > 0 && (
                    <Pagination
                      info={list.data.paginasi}
                      visible={list.data.baris.length}
                      onMove={setPage}
                      loading={list.isFetching}
                    />
                  )}
              </div>
            </>
          )}
        </div>
      )}
    </section>
  )
}

/** Kepala bagian yang dapat dibuka dan ditutup. */
function SectionHeader({
  section,
  open,
  onToggle,
}: {
  section: Section
  open: boolean
  onToggle: () => void
}) {
  return (
    <button
      type="button"
      onClick={onToggle}
      aria-expanded={open}
      className={[
        'flex w-full items-start justify-between gap-4 px-4 py-3 text-left',
        'transition-colors duration-150 ease-halus hover:bg-slate-50',
        'focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500/50',
      ].join(' ')}
    >
      <span>
        <span className="block text-sm font-semibold text-slate-900">{section.nama}</span>
        <span className="mt-0.5 block text-xs text-slate-600">{section.keterangan}</span>
      </span>
      <ChevronIcon
        className={[
          'mt-1 size-4 shrink-0 text-slate-500 transition-transform duration-150',
          open ? 'rotate-180' : '',
        ].join(' ')}
      />
    </button>
  )
}

/**
 * Keterangan bagian yang memang kosong di sistem lama.
 *
 * Bagian "Evaluasi Progress Klaim" punya judul dan kerangka tabel di Pega, tetapi nol
 * properti terikat dan nol activity pengisi. Mengarang isinya berarti mengarang layar yang
 * tidak pernah ada; menghilangkannya begitu saja akan membuat orang mengira modulnya belum
 * selesai.
 */
function EmptyRegionNote() {
  return (
    <p className="text-sm text-slate-600">
      Bagian ini kosong di sistem lama: judulnya ada, tetapi tidak ada satu pun kolom
      maupun sumber data di baliknya. Tidak ada yang dapat dipindahkan, dan isinya menunggu
      keterangan dari pemilik proses.
    </p>
  )
}

/**
 * Penyaring di atas tabel.
 *
 * Kontrolnya digambar HANYA bila bagian yang terbuka memang menyaringnya — dan itu
 * dinyatakan server, bukan ditebak layar.
 */
function FilterBar({
  section,
  lines,
  filter,
  search,
  onSearch,
  onCommitSearch,
  onChange,
  onClear,
}: {
  section: Section
  lines: BusinessLine[]
  filter: FilterForm
  search: string
  onSearch: (text: string) => void
  onCommitSearch: () => void
  onChange: (patch: Partial<FilterForm>) => void
  onClear: () => void
}) {
  const anyControl =
    section.pakai_pencarian || section.pakai_lini_bisnis || section.pakai_rentang_tanggal

  if (!anyControl) return null

  return (
    <div className="flex flex-wrap items-end gap-4">
      {section.pakai_pencarian && (
        <div className="w-full sm:w-72">
          <label
            htmlFor={`cari-${section.kode}`}
            className="block text-sm font-medium text-slate-700"
          >
            Cari
          </label>
          <input
            id={`cari-${section.kode}`}
            type="search"
            value={search}
            placeholder="No Klaim, No Polis, atau PIC"
            onChange={(event) => onSearch(event.target.value)}
            onKeyDown={(event) => {
              if (event.key === 'Enter') onCommitSearch()
            }}
            className={[
              'mt-1 block w-full rounded-kontrol border border-slate-300 px-3 py-2 text-sm',
              'focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500/30',
            ].join(' ')}
          />
          <p className="mt-1 text-xs text-slate-500">
            Menelusuri No Klaim, No Polis, dan nama PIC sekaligus.
          </p>
        </div>
      )}

      {section.pakai_lini_bisnis && (
        <div className="w-full sm:w-64">
          <SelectField
            id={`bisnis-${section.kode}`}
            label="Lini Bisnis"
            options={lines.map((line) => ({ value: line.kode, label: line.label }))}
            emptyText="Pilih lini bisnis"
            value={filter.bisnis}
            onChange={(event) => onChange({ bisnis: event.target.value })}
          />
          {/*
            Wajib, dan itu bukan pilihan kami: nilai yang sama dicocokkan ke
            MST_USER_TEKNIK.TYPE_BUSINESS, sehingga tanpa lini bisnis tidak ada satu pun
            petugas yang cocok.
          */}
          <p className="mt-1 text-xs text-slate-500">Wajib dipilih.</p>
        </div>
      )}

      {section.pakai_rentang_tanggal && (
        <>
          <DateField
            id={`dari-${section.kode}`}
            label="Tanggal Registrasi dari"
            value={filter.dari}
            onChange={(value) => onChange({ dari: value })}
          />
          <DateField
            id={`sampai-${section.kode}`}
            label="sampai"
            value={filter.sampai}
            onChange={(value) => onChange({ sampai: value })}
          />
        </>
      )}

      <div className="flex gap-2 pb-0.5">
        {section.pakai_pencarian && (
          <Button tone="utama" onClick={onCommitSearch}>
            Cari Data
          </Button>
        )}
        <Button tone="kedua" onClick={onClear}>
          Clear Filter
        </Button>
      </div>
    </div>
  )
}

/** Satu isian tanggal. */
function DateField({
  id,
  label,
  value,
  onChange,
}: {
  id: string
  label: string
  value: string
  onChange: (value: string) => void
}) {
  return (
    <div className="w-full sm:w-44">
      <label htmlFor={id} className="block text-sm font-medium text-slate-700">
        {label}
      </label>
      <input
        id={id}
        type="date"
        value={value}
        onChange={(event) => onChange(event.target.value)}
        className={[
          'mt-1 block w-full rounded-kontrol border border-slate-300 px-3 py-2 text-sm',
          'focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500/30',
        ].join(' ')}
      />
    </div>
  )
}

/**
 * Keterangan kontrol yang tampil di sistem lama tetapi tidak menyaring apa pun.
 *
 * Keduanya sengaja tidak digambar sebagai kontrol mati yang mengundang klik. Yang digambar
 * adalah alasannya — supaya pengguna yang mencari penyaring Tanggal Kejadian memperoleh
 * jawaban alih-alih menduga modulnya belum selesai.
 */
function DeadControlNotes({ section }: { section: Section }) {
  if (section.kontrol_mati.length === 0) return null

  return (
    <ul className="mt-3 list-disc space-y-1 pl-5 text-xs text-slate-500">
      {section.kontrol_mati.map((control) => (
        <li key={control.nama}>
          Penyaring <span className="font-medium">{control.nama}</span> tidak dibawa —{' '}
          {control.alasan}.
        </li>
      ))}
    </ul>
  )
}

/**
 * Paginasi "sebelumnya / berikutnya", bukan nomor halaman.
 *
 * Bentuknya sama dengan layar Inbox Admin, Pelaporan Klaim, dan View History Claim supaya
 * keempatnya tidak terasa dirakit dari empat aplikasi berbeda. Ia hidup di sini, bukan di
 * dalam `DataTable`, karena komponen tabel baku belum mengenal paginasi server — itu
 * lingkup `TKT-U2-001`.
 */
function Pagination({
  info,
  visible,
  onMove,
  loading,
}: {
  info: PageInfo
  visible: number
  onMove: (page: number) => void
  loading: boolean
}) {
  const first = visible === 0 ? 0 : (info.halaman - 1) * info.ukuran + 1
  const last = (info.halaman - 1) * info.ukuran + visible

  return (
    <div className="mt-4 flex flex-wrap items-center justify-between gap-3">
      <p className="text-sm text-slate-600" role="status">
        Menampilkan {first}–{last} dari {info.total} baris.
      </p>
      <div className="flex gap-2">
        <Button
          tone="kedua"
          onClick={() => onMove(Math.max(1, info.halaman - 1))}
          disabled={info.halaman <= 1 || loading}
        >
          Sebelumnya
        </Button>
        <Button
          tone="kedua"
          onClick={() => onMove(info.halaman + 1)}
          disabled={info.halaman >= info.total_halaman || loading}
        >
          Berikutnya
        </Button>
      </div>
    </div>
  )
}

/**
 * columnsFor menyusun kolom tabel dari bentuk yang ditetapkan server.
 *
 * Kolom aksi hanya ditambahkan pada bagian berbentuk klaim: rekap per PIC tidak punya
 * klaim untuk dibuka. Ia tidak disebut server karena ia bukan DATA melainkan kontrol, dan
 * backend tidak tahu apa pun tentang rute antarmuka.
 */
export function columnsFor(
  section: Section,
  onOpenClaim: (claimNumber: string) => void,
): TableColumn<Row>[] {
  const columns: TableColumn<Row>[] = section.kolom.map((column) => ({
    key: column.kunci,
    title: column.judul,
    value: (row) => cellText(row, column),
    // Judulnya memakai alias Pega yang sebagian menyesatkan; artinya dititipkan ke
    // tooltip kolom supaya "District" yang berisi nama tertanggung tetap dapat dipahami.
    render: renderFor(column),
    alignRight: isCount(column.isian),
  }))

  if (section.bentuk === 'klaim') {
    columns.push({
      key: 'aksi',
      title: '',
      value: () => '',
      render: (row) => <DetailButton row={row as ClaimRow} onOpen={onOpenClaim} />,
      noSort: true,
      alignRight: true,
    })
  }

  return columns
}

/**
 * renderFor menggambar sel.
 *
 * Dua kolom digambar khusus, sisanya teks biasa:
 *
 *   - `no_polis` membawa nomor perpanjangan di bawahnya. Di Pega keduanya berada di baris
 *     rincian yang terbuka saat baris diklik; komponen tabel baku belum punya baris
 *     rincian, dan menempelkannya di sini menjaga keterangannya tetap terbaca alih-alih
 *     hilang.
 *   - `posisi` membawa jumlah posisi. Satu klaim dapat berada di beberapa posisi sekaligus,
 *     dan tanpa angkanya pengguna harus menghitung koma.
 */
function renderFor(column: Column): (row: Row) => ReactNode {
  if (column.isian === 'no_polis') {
    return (row) => {
      const claim = row as ClaimRow
      return (
        <span title={column.keterangan}>
          {claim.no_polis || '—'}
          {claim.prod_ke !== '' && (
            <span className="mt-0.5 block text-xs text-slate-500">
              Prod ke-{claim.prod_ke}
            </span>
          )}
        </span>
      )
    }
  }

  if (column.isian === 'posisi') {
    return (row) => {
      const claim = row as ClaimRow
      return (
        <span title={column.keterangan}>
          {claim.posisi || '—'}
          {claim.jumlah_posisi > 1 && (
            <span className="mt-0.5 block text-xs text-slate-500">
              {claim.jumlah_posisi} posisi berjalan
            </span>
          )}
        </span>
      )
    }
  }

  return (row) => <span title={column.keterangan}>{cellText(row, column)}</span>
}

/** Tombol "Lihat Detail Klaim". */
function DetailButton({
  row,
  onOpen,
}: {
  row: ClaimRow
  onOpen: (claimNumber: string) => void
}) {
  return (
    <Button
      tone="halus"
      disabled={row.no_klaim === ''}
      onClick={() => onOpen(row.no_klaim)}
    >
      Lihat Detail Klaim
    </Button>
  )
}

/** isCount menyatakan sebuah kolom berisi angka pencacah. */
function isCount(field: Column['isian']): boolean {
  return (
    field === 'jumlah_klaim' ||
    field === 'jumlah_pembaruan' ||
    field === 'jatuh_tempo_hari_ini' ||
    field === 'tepat_waktu' ||
    field === 'terlambat'
  )
}

/**
 * cellText menyusun teks satu sel.
 *
 * Tanggal diformat ke `1 Juni 2026`, angka ditampilkan apa adanya, dan sisanya teks. Nilai
 * kosong menjadi tanda pisah — bukan sel kosong yang tidak dapat dibedakan dari kolom yang
 * gagal dimuat.
 *
 * Kolom gabungan — posisi, kedua status progres, dan tenggatnya — memuat BEBERAPA nilai
 * yang dipisah koma, dan tanggal di dalamnya tidak diformat ulang. Memformatnya menuntut
 * memecah teksnya lagi, dan pemecahan itu akan salah begitu sebuah nilai memuat koma.
 */
export function cellText(row: Row, column: Column): string {
  const value = (row as Record<string, unknown>)[column.isian]

  if (value === null || value === undefined || value === '') return '—'
  if (typeof value === 'number') return String(value)

  const text = String(value)
  return isDate(text) ? formatDate(text) : text
}

/** isDate mengenali bentuk `YYYY-MM-DD` yang dikirim server untuk kolom tanggal tunggal. */
function isDate(text: string): boolean {
  return /^\d{4}-\d{2}-\d{2}$/.test(text)
}

/**
 * rowKeyFor menyusun kunci baris.
 *
 * Bagian klaim memakai nomor klaim; rekap per PIC memakai nama petugas. Keduanya unik di
 * dalam hasilnya sendiri.
 */
function rowKeyFor(section: Section): (row: Row) => string {
  if (section.bentuk === 'pic') {
    return (row) => (row as { pic: string }).pic
  }
  return (row) => (row as ClaimRow).no_klaim
}

/**
 * emptyMessageFor menjelaskan hasil kosong menurut sebabnya.
 *
 * "Tidak ada data" tidak cukup: daftar yang kosong karena penyaring berbeda jauh dari
 * daftar yang memang tidak punya pekerjaan, dan tindakannya pun berbeda.
 */
export function emptyMessageFor(section: Section, filter: FilterForm): string {
  if (section.bentuk === 'pic' && filter.bisnis === '') {
    return 'Pilih lini bisnis lebih dulu untuk melihat rekapnya.'
  }
  if (filter.cari.trim() !== '') {
    return 'Tidak ada baris yang cocok dengan pencarian ini.'
  }
  if (filter.dari !== '' || filter.sampai !== '') {
    return 'Tidak ada baris pada rentang tanggal registrasi yang dipilih.'
  }
  if (section.bentuk === 'pic') {
    return 'Belum ada rekap untuk Anda pada lini bisnis ini. Rekap hanya muncul bila Anda terdaftar sebagai PIC teknis lini bisnis tersebut.'
  }
  if (section.kode === 'next-fu') {
    return 'Tidak ada klaim yang harus ditindaklanjuti hari ini.'
  }
  return 'Tidak ada klaim yang sedang berjalan.'
}

/** messageOf mengambil pesan yang layak dibaca pengguna dari sebuah galat. */
function messageOf(error: unknown): string {
  if (error instanceof APIError) return error.message
  return 'Coba lagi beberapa saat lagi. Bila terus berulang, hubungi tim teknis.'
}
