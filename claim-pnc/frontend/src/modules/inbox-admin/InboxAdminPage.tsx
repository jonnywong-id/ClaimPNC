import { useEffect, useState, type ReactNode } from 'react'
import { useNavigate } from 'react-router-dom'

import { APIError } from '@/api/client'
import { useSelectedPortal } from '@/app/portal'
import { Button } from '@/components/Button'
import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage } from '@/components/ErrorMessage'
import { SelectField } from '@/components/SelectField'
import { formatDate } from '@/components/format'

import { InboxTabs } from './InboxTabs'
import { useInboxAdminList, useInboxAdminMetadata } from './api'
import {
  EMPTY_FILTER,
  type BusinessLine,
  type DisabledTab,
  type FilterForm,
  type PageInfo,
  type Tab,
  type TabColumn,
  type WorkItem,
} from './types'

/**
 * Inbox Admin — menu `MENU_ID 63`, pengganti harness `PNCInboxAdmin`.
 *
 * Isinya daftar pekerjaan petugas admin klaim, dipecah menjadi delapan antrean. Per `D-79`
 * ia benar-benar Inbox: barisnya pekerjaan, hilang begitu klaimnya selesai, dan punya
 * tenggat berupa kolom Aging.
 *
 * # Susunan layar, dan dari mana bentuknya
 *
 * Diambil dari `Section/PNCInboxAdmin-Section.xml` apa adanya: judul, bilah tab, penyaring,
 * lalu grid berhalaman. Nama kolom TIDAK diterjemahkan — `D-13` menetapkan tampilan meniru
 * Pega, dan itulah teks yang selama ini dibaca pengguna.
 *
 * # Kenapa kolomnya datang dari server
 *
 * Karena kedelapan tab punya kolom yang berbeda-beda, dan daftar itu adalah hasil pembacaan
 * export Pega yang tercatat di backend. Menyalinnya ke sini berarti daftar yang sama hidup
 * di dua tempat.
 *
 * # Tiga tab yang tidak ada, dan itu bukan kelalaian
 *
 * Tab Komunikasi — "Not Answered", "Not replied from Receiver", dan "Replied from ASM" —
 * dinyatakan Work Owner 2026-09-20 sudah tidak dipakai. Ketiadaannya dijelaskan di bawah
 * tabel, bukan dibiarkan sebagai tab yang hilang tanpa keterangan.
 */
export function InboxAdminPage() {
  const [filter, setFilter] = useState<FilterForm>(EMPTY_FILTER)
  const [page, setPage] = useState(1)

  /**
   * Isi kotak cari yang sedang DIKETIK, terpisah dari kata kunci yang sudah dikirim.
   *
   * Keduanya dipisah karena permintaan ini mahal: server menarik SELURUH baris yang cocok
   * sebelum memotong halamannya (paginasi direplikasi apa adanya, keputusan Work Owner
   * 2026-09-20). Mengirim satu permintaan per huruf berarti delapan kali penarikan penuh
   * untuk satu kata — terhadap tabel berpuluh juta baris itu bukan sekadar boros.
   */
  const [draft, setDraft] = useState('')

  // Ketikan menunggu jeda sebelum dikirim. Jedanya cukup panjang untuk menelan satu kata
  // yang diketik cepat, dan cukup pendek untuk tidak terasa seperti layar yang menggantung.
  useEffect(() => {
    if (draft === filter.cari) return

    const timer = setTimeout(() => {
      setFilter((previous) => ({ ...previous, cari: draft }))
      setPage(1)
    }, 350)
    return () => clearTimeout(timer)
  }, [draft, filter.cari])

  const portal = useSelectedPortal((state) => state.alias)
  const meta = useInboxAdminMetadata()
  const list = useInboxAdminList(filter, page, meta.isSuccess)

  const tabs: Tab[] = meta.data?.tab ?? []
  const active = filter.tab || meta.data?.tab_bawaan || ''
  const tab = tabs.find((candidate) => candidate.kode === active)

  /**
   * Berpindah tab MEMBERSIHKAN penyaring dan mengembalikan ke halaman pertama.
   *
   * Membawa kata kunci lama ke tab baru akan menampilkan antrean yang tampak kosong
   * padahal isinya ada — dan pengguna tidak punya cara melihat bahwa penyebabnya kotak
   * cari yang masih terisi dari tab sebelumnya.
   */
  function selectTab(code: string) {
    setFilter({ ...EMPTY_FILTER, tab: code })
    setDraft('')
    setPage(1)
  }

  function changeBusiness(code: string) {
    setFilter((previous) => ({ ...previous, bisnis: code }))
    setPage(1)
  }

  if (portal === null) {
    return (
      <PageFrame>
        <ErrorMessage
          title="Pilih entitas lebih dulu"
          description={
            'Antrean kerja milik satu badan hukum, dan aplikasi ini melayani empat. ' +
            'Pilih portal di bilah atas untuk membukanya.'
          }
          tone="gangguan"
        />
      </PageFrame>
    )
  }

  if (meta.isError) {
    return (
      <PageFrame>
        <ErrorMessage
          title="Layar tidak dapat dibuka"
          description={messageOf(meta.error)}
          tone="gangguan"
        />
      </PageFrame>
    )
  }

  return (
    <PageFrame>
      <div className="mt-4">
        <InboxTabs tabs={tabs} active={active} onSelect={selectTab} />
      </div>

      {tab && (
        <>
          <p className="mt-3 text-sm text-slate-600">{tab.keterangan}</p>

          <FilterBar
            tab={tab}
            lines={meta.data?.lini_bisnis ?? []}
            business={filter.bisnis}
            search={draft}
            onBusiness={changeBusiness}
            onSearch={setDraft}
          />

          <div className="mt-4">
            <DataTable<WorkItem>
              columns={columnsFor(tab, (row) => <DetailButton item={row} />)}
              rows={list.data?.baris ?? []}
              rowKey={(row) => `${row.referensi}|${row.case_id}`}
              title={tab.nama}
              // Kotak cari bawaan disembunyikan: layar ini punya penyaringnya sendiri di
              // atas, dan yang kedua hanya akan menyaring halaman yang sedang terbuka —
              // hasilnya menyesatkan pada data berhalaman.
              hideSearch
              isLoading={list.isPending}
              error={
                list.isError ? (
                  <ErrorMessage
                    title="Antrean tidak dapat dimuat"
                    description={messageOf(list.error)}
                    tone="gangguan"
                  />
                ) : undefined
              }
              emptyMessage={emptyMessageFor(tab, filter)}
            />

            {list.data && list.data.paginasi.total > 0 && (
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

      <Notes
        limitations={meta.data?.keterbatasan ?? []}
        disabled={meta.data?.tab_dinonaktifkan ?? []}
      />
    </PageFrame>
  )
}

function PageFrame({ children }: { children: ReactNode }) {
  return (
    <div className="mx-auto max-w-[96rem] px-4 py-8">
      <header className="border-b border-slate-200 pb-4">
        <h1 className="text-xl font-semibold text-slate-900">Inbox Admin</h1>
        <p className="mt-1 text-sm text-slate-600">
          Antrean kerja admin klaim: klaim berjalan, dokumen yang belum diregistrasi,
          permintaan survei, dan pengingat RCL/PUCL.
        </p>
      </header>
      {children}
    </div>
  )
}

/**
 * Penyaring di atas tabel.
 *
 * Kontrolnya digambar HANYA bila tab yang terbuka memang menyaringnya — dan itu dinyatakan
 * server, bukan ditebak layar. Kotak cari yang tidak menyaring apa pun lebih buruk daripada
 * kotak cari yang tidak ada: pengguna akan menyimpulkan antreannya kosong.
 */
function FilterBar({
  tab,
  lines,
  business,
  search,
  onBusiness,
  onSearch,
}: {
  tab: Tab
  lines: BusinessLine[]
  business: string
  search: string
  onBusiness: (code: string) => void
  onSearch: (text: string) => void
}) {
  if (!tab.pakai_pencarian && !tab.pakai_lini_bisnis && !tab.hanya_milik_saya) {
    return null
  }

  return (
    <div className="mt-4 flex flex-wrap items-end gap-4">
      {tab.pakai_lini_bisnis && (
        <div className="w-full sm:w-64">
          <SelectField
            id="inbox-admin-bisnis"
            label="Business"
            options={lines.map((line) => ({ value: line.kode, label: line.label }))}
            emptyText="Semua Lini Bisnis"
            value={business}
            onChange={(event) => onBusiness(event.target.value)}
          />
        </div>
      )}

      {tab.pakai_pencarian && (
        <div className="w-full sm:w-72">
          <label
            htmlFor="inbox-admin-cari"
            className="block text-sm font-medium text-slate-700"
          >
            Cari
          </label>
          {/*
            Kotak ini sengaja TIDAK dinonaktifkan saat permintaan sedang berjalan.
            Menonaktifkannya berarti huruf yang diketik selama permintaan itu HILANG —
            dan karena setiap ketikan memicu permintaan, kotak yang menonaktifkan diri
            akan menelan sebagian besar kata yang diketik cepat.
          */}
          <input
            id="inbox-admin-cari"
            type="search"
            value={search}
            placeholder="Case ID atau No Polis"
            onChange={(event) => onSearch(event.target.value)}
            className={[
              'mt-1 block w-full rounded-kontrol border border-slate-300 px-3 py-2 text-sm',
              'focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500/30',
            ].join(' ')}
          />
          {/*
            Jangkauan kotak cari dinyatakan di tempatnya, bukan hanya di catatan bawah.
            Di sistem lama ia memang hanya menyentuh dua kolom, dan pengguna yang mencari
            nama tertanggung akan mengira antreannya kosong.
          */}
          <p className="mt-1 text-xs text-slate-500">
            Hanya menelusuri Case ID dan No Polis.
          </p>
        </div>
      )}

      {tab.hanya_milik_saya && (
        <p className="pb-2 text-xs text-slate-500">
          Antrean ini hanya berisi pekerjaan milik Anda.
        </p>
      )}
    </div>
  )
}

/**
 * Tombol "Lihat Detail Klaim".
 *
 * Layar tujuannya adalah `MENU_ID 75` "View Claim" (`PNCViewClaim`) — modul tersendiri yang
 * belum dibangun. Tombolnya tetap dibangun atas keputusan Work Owner 2026-09-20, dan
 * tujuannya diarahkan ke rute yang sudah ada tempat keadaan itu dinyatakan apa adanya.
 *
 * Yang dikirim adalah `referensi`, kunci teknis Pega yang memang dipakai
 * `setDataViewKlaim_Act` di sistem lama. Dengan begitu menyalakan layar rincian kelak tidak
 * menuntut perubahan kontrak API modul ini.
 */
function DetailButton({ item }: { item: WorkItem }) {
  const navigate = useNavigate()
  const key = item.referensi || item.case_id

  return (
    <Button
      tone="halus"
      disabled={key === ''}
      onClick={() => navigate(`/view-claim/${encodeURIComponent(key)}`)}
    >
      Lihat Detail Klaim
    </Button>
  )
}

/**
 * Paginasi "sebelumnya / berikutnya", bukan nomor halaman.
 *
 * Bentuknya sama dengan layar Pelaporan Klaim dan View History Claim supaya ketiganya tidak
 * terasa dirakit dari tiga aplikasi berbeda. Ia hidup di sini, bukan di dalam `DataTable`,
 * karena komponen tabel baku belum mengenal paginasi server — itu lingkup `TKT-U2-001`.
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
 * Catatan di bawah tabel: keterbatasan yang berlaku dan tab yang tidak dibangun.
 *
 * Keduanya datang dari SERVER, bukan ditulis tetap di sini, supaya hilang dengan sendirinya
 * begitu penghalangnya hilang. Tanpa catatan ini, Aging yang lebih besar daripada di Pega
 * dan tab Komunikasi yang tidak ada akan dilaporkan berulang kali sebagai kerusakan.
 */
function Notes({
  limitations,
  disabled,
}: {
  limitations: string[]
  disabled: DisabledTab[]
}) {
  if (limitations.length === 0 && disabled.length === 0) return null

  return (
    <section className="mt-6 rounded-kartu border border-slate-200 bg-slate-50 px-4 py-3">
      <h2 className="text-sm font-medium text-slate-800">Yang perlu diketahui</h2>

      {limitations.length > 0 && (
        <ul className="mt-2 list-disc space-y-1 pl-5 text-xs text-slate-600">
          {limitations.map((line) => (
            <li key={line}>{line}</li>
          ))}
        </ul>
      )}

      {disabled.length > 0 && (
        <p className="mt-2 text-xs text-slate-600">
          Tab {disabled.map((tab) => `"${tab.nama}"`).join(', ')} tidak dibawa ke sistem
          baru — {disabled[0]?.alasan ?? 'tidak dipakai'}.
        </p>
      )}
    </section>
  )
}

/**
 * columnsFor menyusun kolom tabel dari bentuk yang ditetapkan server.
 *
 * Kolom aksi ditambahkan di ujung, bukan disebut server: ia bukan DATA melainkan kontrol,
 * dan backend tidak tahu apa pun tentang rute antarmuka.
 */
function columnsFor(tab: Tab, action: (row: WorkItem) => ReactNode): Column<WorkItem>[] {
  const columns: Column<WorkItem>[] = tab.kolom.map((column) => ({
    key: column.kunci,
    title: column.judul,
    value: (row) => cellText(row, column),
    // Kolom Aging diratakan ke kanan karena isinya angka, sama seperti kolom nilai uang
    // di layar lain.
    alignRight: isAging(column.kunci),
  }))

  columns.push({
    key: 'aksi',
    title: '',
    value: () => '',
    render: action,
    noSort: true,
    alignRight: true,
  })

  return columns
}

/** isAging menyatakan sebuah kolom berisi jumlah hari. */
function isAging(key: TabColumn['kunci']): boolean {
  return key === 'aging_lapor' || key === 'aging_total' || key === 'aging_lod' || key === 'aging_request'
}

/**
 * cellText menyusun teks satu sel.
 *
 * Tiga bentuk ditangani berbeda: tanggal diformat ke `1 Juni 2026`, Aging diberi satuan
 * hari, dan sisanya ditampilkan apa adanya. Nilai kosong menjadi tanda pisah — bukan sel
 * kosong yang tidak dapat dibedakan dari kolom yang gagal dimuat.
 */
function cellText(row: WorkItem, column: TabColumn): string {
  const value = row[column.kunci]

  if (value === null || value === undefined || value === '') return '—'

  if (isAging(column.kunci)) return `${value} hari`

  const text = String(value)
  return isDate(text) ? formatDate(text) : text
}

/** isDate mengenali bentuk `YYYY-MM-DD` yang dikirim server untuk seluruh kolom tanggal. */
function isDate(text: string): boolean {
  return /^\d{4}-\d{2}-\d{2}$/.test(text)
}

/**
 * emptyMessageFor menjelaskan antrean kosong menurut sebabnya.
 *
 * "Tidak ada data" tidak cukup: antrean yang kosong karena penyaring berbeda jauh dari
 * antrean yang memang tidak punya pekerjaan, dan tindakannya pun berbeda.
 */
function emptyMessageFor(tab: Tab, filter: FilterForm): string {
  if (filter.cari.trim() !== '') {
    return 'Tidak ada baris yang cocok dengan pencarian ini. Kotak cari hanya menelusuri Case ID dan No Polis.'
  }
  if (filter.bisnis !== '') {
    return 'Tidak ada baris pada lini bisnis yang dipilih.'
  }
  if (tab.hanya_milik_saya) {
    return 'Tidak ada pekerjaan milik Anda di antrean ini.'
  }
  return 'Antrean ini sedang kosong.'
}

/** messageOf mengambil pesan yang layak dibaca pengguna dari sebuah galat. */
function messageOf(error: unknown): string {
  if (error instanceof APIError) return error.message
  return 'Coba lagi beberapa saat lagi. Bila terus berulang, hubungi tim teknis.'
}
