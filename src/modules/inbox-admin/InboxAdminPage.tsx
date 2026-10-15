import { useEffect, useState, type ReactNode } from 'react'
import { useNavigate } from 'react-router-dom'

import { APIError } from '@/api/client'
import { useSelectedPortal } from '@/app/portal'
import { Button } from '@/components/Button'
import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage } from '@/components/ErrorMessage'

import { StatusRegister } from './StatusRegister'
import {
  useInboxAdminCounts,
  useInboxAdminExport,
  useInboxAdminList,
  useInboxAdminMetadata,
  useInboxAdminViewer,
  type ExportKind,
} from './api'
import {
  EMPTY_FILTER,

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
  const counts = useInboxAdminCounts(filter.bisnis, filter.kanwil, meta.isSuccess)
  const viewer = useInboxAdminViewer(meta.isSuccess)

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
    // Lini bisnis DIPERTAHANKAN: di Pega dropdown Bisnis adalah penyaring tingkat layar,
    // di atas daftar Status Register, bukan milik satu antrean.
    setFilter((previous) => ({ ...EMPTY_FILTER, tab: code, bisnis: previous.bisnis, kanwil: previous.kanwil }))
    setDraft('')
    setPage(1)
  }

  function changeBusiness(code: string) {
    setFilter((previous) => ({ ...previous, bisnis: code }))
    setPage(1)
  }

  function changeRegion(code: string) {
    setFilter((previous) => ({ ...previous, kanwil: code }))
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
      <StatusRegister
        tabs={tabs}
        counts={counts.data?.jumlah}
        countsLoading={counts.isPending}
        countsFailed={counts.isError}
        active={active}
        onSelect={selectTab}
        lines={meta.data?.lini_bisnis ?? []}
        business={filter.bisnis}
        onBusiness={changeBusiness}
        viewer={viewer.data}
        region={filter.kanwil}
        onRegion={changeRegion}
      />

      {tab && (
        <>
          <p className="mt-3 text-sm text-slate-600">{tab.keterangan}</p>

          <FilterBar
            tab={tab}
            search={draft}
            onSearch={setDraft}
          />

          <ExportBar tabCode={tab.kode} filter={filter} />

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
              // Mode rapat: antrean berkolom belasan harus muat selebar layar tanpa gulir
              // menyamping, dengan tombol Lihat Detail Klaim di ujung kanan tetap terlihat.
              dense
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
  search,
  onSearch,
}: {
  tab: Tab
  search: string
  onSearch: (text: string) => void
}) {
  if (!tab.pakai_pencarian && !tab.hanya_milik_saya) {
    return null
  }

  return (
    <div className="mt-4 flex flex-wrap items-end gap-4">
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
 * # Dua tujuan, menurut asal klaimnya
 *
 * - Klaim PNCN — lahir di aplikasi ini — dibuka LANGSUNG di halaman kerja klaimnya,
 *   `/registrasi/klaim/<nomor>`, sama seperti Pega membuka case-nya. Nomor klaim dipakai
 *   sebagai kunci: halaman itu mencari klaim menurut nomornya bila ID tidak cocok, dan
 *   nomor itulah PYID barisnya di T_CLAIMLIST_ADMIN. Aturan yang sama dipakai Inbox
 *   Outstanding (`isOpenableHere`); ia disalin, bukan diimpor, karena modul tidak boleh
 *   saling mengimpor.
 * - Klaim Pega menuju `MENU_ID 75` "View Claim" (`PNCViewClaim`) — modul tersendiri yang
 *   belum dibangun — dengan `referensi`, kunci teknis Pega yang dipakai
 *   `setDataViewKlaim_Act` di sistem lama.
 */
export function detailPath(item: Pick<WorkItem, 'case_id' | 'referensi'>): string | null {
  const number = item.case_id.trim()
  if (number.startsWith('PNCN.')) return `/registrasi/klaim/${encodeURIComponent(number)}`
  const key = item.referensi || item.case_id
  return key ? `/view-claim/${encodeURIComponent(key)}` : null
}

function DetailButton({ item }: { item: WorkItem }) {
  const navigate = useNavigate()
  const path = detailPath(item)

  return (
    // Nada `utama` dan tidak terlipat: ini satu-satunya tindakan pada baris antrean, dan
    // tombol berwarna redup di ujung tabel lebar mudah terlewat.
    <Button
      tone="utama"
      className="whitespace-nowrap px-3 py-1.5 text-xs"
      disabled={path === null}
      onClick={() => path && navigate(path)}
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
 * Kolom aksi ditambahkan di sini, bukan disebut server: ia bukan DATA melainkan kontrol,
 * dan backend tidak tahu apa pun tentang rute antarmuka.
 *
 * Ia diletakkan di ujung KANAN (permintaan Work Owner 2026-10-07). Supaya tetap terlihat
 * tanpa gulir menyamping, tabelnya memakai mode rapat `DataTable` dan tanggal diringkas —
 * seluruh kolom muat selebar layar meja.
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
    title: 'Aksi',
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

  if (value == null || value === '') return '—'

  if (isAging(column.kunci)) return `${value} hari`

  const text = String(value)
  return isDate(text) ? shortDate(text) : text
}

/**
 * shortDate menulis `YYYY-MM-DD` sebagai `dd/mm/yyyy`. Antrean ini memuat tiga kolom tanggal
 * berdampingan; bentuk panjang ("7 Oktober 2026") melipat setiap sel menjadi tiga baris dan
 * melebarkan tabel. Nilainya tanggal murni, sehingga tidak ada pergeseran zona waktu.
 */
function shortDate(iso: string): string {
  const [year, month, day] = iso.split('-')
  return `${day}/${month}/${year}`
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

/** Kode tab Branch Claim — satu-satunya tab tempat tombol Export LOD tampil. */
const TAB_BRANCH_CLAIM = '12'

const EXPORT_LABEL: Record<ExportKind, string> = {
  lod: 'Export LOD',
  'hasil-auto-claim': 'Export Hasil Auto Claim',
  'klaim-gagal': 'Export Klaim Gagal',
}

/**
 * ExportBar memuat tombol ekspor CSV layar lama.
 *
 * Export LOD hanya tampil di tab Branch Claim — kondisi `TempView.CityID==12` pada
 * wadahnya di section Pega. Dua tombol Auto Claim tidak punya kondisi tampil pada selnya,
 * sehingga keduanya tampil di setiap tab.
 */
function ExportBar({ tabCode, filter }: { tabCode: string; filter: FilterForm }) {
  const download = useInboxAdminExport()
  const [busy, setBusy] = useState<ExportKind | null>(null)
  const [error, setError] = useState<string | null>(null)

  const kinds: ExportKind[] = [
    ...(tabCode === TAB_BRANCH_CLAIM ? (['lod'] as const) : []),
    'hasil-auto-claim',
    'klaim-gagal',
  ]

  async function run(kind: ExportKind) {
    setBusy(kind)
    setError(null)
    try {
      await download(kind, filter)
    } catch (e) {
      setError(messageOf(e))
    } finally {
      setBusy(null)
    }
  }

  return (
    <div className="mt-3">
      <div className="flex flex-wrap gap-2">
        {kinds.map((kind) => (
          <Button key={kind} tone="kedua" disabled={busy !== null} onClick={() => run(kind)}>
            {busy === kind ? 'Menyiapkan berkas…' : EXPORT_LABEL[kind]}
          </Button>
        ))}
      </div>
      {error && (
        <div className="mt-2">
          <ErrorMessage title="Berkas tidak dapat diunduh" description={error} tone="gangguan" />
        </div>
      )}
    </div>
  )
}
