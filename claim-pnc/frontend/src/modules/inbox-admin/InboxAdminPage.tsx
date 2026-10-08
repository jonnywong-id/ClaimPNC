import { useState, type ReactNode } from 'react'

import { useSelectedPortal } from '@/app/portal'
import { type Column } from '@/components/DataTable'
import { SelectField } from '@/components/SelectField'
import { formatDate } from '@/components/format'
import { screenGate } from '@/components/inbox/InboxNotices'
import { QueueTable } from '@/components/inbox/QueueTable'
import { ViewClaimButton } from '@/components/inbox/ViewClaimButton'
import { apiMessageOf, isISODate } from '@/components/inbox/messages'
import { columnsWithAction } from '@/components/inbox/serverColumns'
import { useDebouncedCommit } from '@/components/inbox/useDebouncedCommit'
import { InboxPageFrame } from '@/components/inbox/InboxPageFrame'

import { InboxTabs } from './InboxTabs'
import { useInboxAdminList, useInboxAdminMetadata } from './api'
import {
  EMPTY_FILTER,
  type BusinessLine,
  type DisabledTab,
  type FilterForm,
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

  // Ketikan menunggu jeda sebelum dikirim (`useDebouncedCommit`).
  useDebouncedCommit(draft, filter.cari, (value) => {
    setFilter((previous) => ({ ...previous, cari: value }))
    setPage(1)
  })

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

  const gate = screenGate({
    portal,
    subject: 'Antrean kerja',
    failed: meta.isError,
    error: meta.error,
    describe: apiMessageOf,
  })
  if (gate) return <PageFrame>{gate}</PageFrame>

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
            <QueueTable<WorkItem>
              query={list}
              columns={columnsFor(tab, renderDetailButton)}
              rowKey={(row) => `${row.referensi}|${row.case_id}`}
              title={tab.nama}
              emptyMessage={emptyMessageFor(tab, filter)}
              onMove={setPage}
              describe={apiMessageOf}
            />
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

function PageFrame({ children }: Readonly<{ children: ReactNode }>) {
  return (
    <InboxPageFrame
      title="Inbox Admin"
      intro={
        'Antrean kerja admin klaim: klaim berjalan, dokumen yang belum diregistrasi, ' +
        'permintaan survei, dan pengingat RCL/PUCL.'
      }
    >
      {children}
    </InboxPageFrame>
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
}: Readonly<{
  tab: Tab
  lines: BusinessLine[]
  business: string
  search: string
  onBusiness: (code: string) => void
  onSearch: (text: string) => void
}>) {
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
 * Isi kolom aksi tiap baris. Didefinisikan di tingkat modul — bukan sebagai fungsi panah di
 * dalam render layar — supaya tidak dibuat ulang setiap render (temuan SonarQube S6478).
 */
function renderDetailButton(row: WorkItem): ReactNode {
  return <ViewClaimButton reference={row.referensi || row.case_id} />
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
}: Readonly<{
  limitations: string[]
  disabled: DisabledTab[]
}>) {
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
  // Kolom Aging diratakan ke kanan karena isinya angka, sama seperti kolom nilai uang di
  // layar lain.
  return columnsWithAction<WorkItem, TabColumn>(tab.kolom, cellText, action, (column) => ({
    alignRight: isAging(column.kunci),
  }))
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
  return isISODate(text) ? formatDate(text) : text
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

