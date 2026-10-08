import { useState, type ReactNode } from 'react'

import { useSelectedPortal } from '@/app/portal'
import { type Column } from '@/components/DataTable'
import { formatDate } from '@/components/format'
import { ClaimNumberLink } from '@/components/inbox/ClaimNumberLink'
import { CreateClaimButton } from '@/components/inbox/CreateClaimButton'
import { InboxCheckbox } from '@/components/inbox/InboxCheckbox'
import { BlockedNotice, NoteList, screenGate } from '@/components/inbox/InboxNotices'
import { QueueTable } from '@/components/inbox/QueueTable'
import { apiMessageOf, isISODate } from '@/components/inbox/messages'
import { serverColumns } from '@/components/inbox/serverColumns'

import { TreatyViewSelect } from './TreatyViewSelect'
import { useClaimTreatyPropList, useClaimTreatyPropMetadata } from './api'
import {
  EMPTY_FILTER,
  type FilterForm,
  type Tab,
  type TabColumn,
  type WorkItem,
} from './types'

/**
 * Inbox Claim Treaty Prop — menu `MENU_ID 54`, pengganti harness
 * `InboxClaimTreaty_Harness`.
 *
 * Isinya daftar pekerjaan klaim treaty PROPORSIONAL: klaim yang masuk lewat jalur treaty
 * inward, tempat ASM menjadi penanggung ulang atas klaim milik perusahaan asuransi lain
 * (Ceding Co). Per `D-79` ia benar-benar Inbox: barisnya pekerjaan dari tabel penugasan
 * Pega, dan hilang begitu penugasannya selesai.
 *
 * Menu `MENU_ID 55` "Inbox Claim Treaty Non Prop" adalah layar SAUDARA yang berdiri
 * sendiri dan tidak dilayani berkas ini.
 *
 * # Susunan layar, dan dari mana bentuknya
 *
 * Mengikuti layar Pega PRODUKSI, yang dibandingkan berdampingan bersama Work Owner pada
 * 2026-09-29: judul "Claim Treatyin In Progress", tombol pembuat klaim, dropdown pemilih
 * antrean, lalu grid berjudul "Work List Treatyin Propotional". Nama kolom TIDAK
 * diterjemahkan — `D-13` menetapkan tampilan meniru Pega, dan itulah teks yang selama ini
 * dibaca pengguna.
 *
 * Bentuk itu berbeda dari `Section/InboxClaimTreaty_Section-Section.xml` di export pada dua
 * hal, dan keduanya karena Pega produksi sudah berubah sesudah export diambil (`R-09`):
 * pemilih antreannya DROPDOWN alih-alih bilah tab, dan gridnya punya SEPULUH kolom alih-alih
 * delapan — "Last update" dan "Status Claim ID" menyusul di ujung.
 *
 * # Nomor klaim adalah TAUTAN, bukan tombol di kolom terakhir
 *
 * Di Pega produksi, mengklik Claim ID menjalankan Flow Action `OutstandingClaim` dan
 * menggambar `Section/OutstandingClaim-Section.xml`. Layar itu dibangun sebagai modul
 * tersendiri (`modules/outstanding-claim`), dan tautan di sini menunjuk ke sana.
 *
 * Tombol "Lihat Detail Klaim" di kolom terakhir karena itu DIHAPUS: ia menunjuk penampung
 * `/view-claim/:referensi` yang memang belum punya isi, sementara tautan nomor klaim
 * menunjuk layar yang sudah ada. Membiarkan keduanya berarti dua jalan dari satu baris ke
 * dua layar yang berbeda, dan yang satu buntu.
 *
 * # Kenapa kolomnya datang dari server
 *
 * Karena ketiga tab punya kolom yang berbeda, dan daftar itu adalah hasil pembacaan export
 * Pega yang tercatat di backend. Menyalinnya ke sini berarti daftar yang sama hidup di dua
 * tempat.
 */
export function ClaimTreatyPropPage() {
  const [filter, setFilter] = useState<FilterForm>(EMPTY_FILTER)
  const [page, setPage] = useState(1)

  const portal = useSelectedPortal((state) => state.alias)
  const meta = useClaimTreatyPropMetadata()

  const tabs: Tab[] = meta.data?.tab ?? []
  const active = filter.tab || meta.data?.tab_bawaan || ''
  const tab = tabs.find((candidate) => candidate.kode === active)

  // Tab terhalang TIDAK meminta isinya. Permintaannya pasti ditolak server dengan 422,
  // dan galat yang sudah diketahui pasti terjadi bukan galat yang layak ditampilkan —
  // yang layak ditampilkan adalah alasan terhalangnya.
  const blocked = tab?.terhalang === true
  const list = useClaimTreatyPropList(filter, page, meta.isSuccess && !blocked)

  /**
   * Berpindah tab MEMBERSIHKAN penyaring dan mengembalikan ke halaman pertama.
   *
   * Membawa centang "See All Claim" ke tab yang tidak mengenalnya akan membuat centang
   * yang tampak aktif padahal tidak mengubah apa pun.
   */
  function selectTab(code: string) {
    setFilter({ ...EMPTY_FILTER, tab: code })
    setPage(1)
  }

  function toggleSeeAll(value: boolean) {
    setFilter((previous) => ({ ...previous, lihatSemua: value }))
    setPage(1)
  }

  const gate = screenGate({
    portal,
    subject: 'Antrean klaim treaty',
    failed: meta.isError,
    error: meta.error,
    describe: apiMessageOf,
  })
  if (gate) return <PageFrame>{gate}</PageFrame>

  return (
    <PageFrame>
      <div className="mt-4">
        <TreatyViewSelect tabs={tabs} active={active} onSelect={selectTab} />
      </div>

      {tab && (
        <>
          <p className="mt-3 text-sm text-slate-600">{tab.keterangan}</p>

          {tab.terhalang ? (
            <BlockedNotice tab={tab} />
          ) : (
            <>
              <FilterBar
                tab={tab}
                seeAll={filter.lihatSemua}
                onSeeAll={toggleSeeAll}
              />

              <div className="mt-4">
                <QueueTable<WorkItem>
                  query={list}
                  columns={columnsFor(tab)}
                  rowKey={(row) => `${row.referensi}|${row.claim_id}`}
                  // Judul GRID, bukan teks pilihan dropdown. Keduanya tampil bersamaan
                  // dan berbunyi berbeda: dropdown "Prop Treaty-in Admin", grid "Work
                  // List Treatyin Propotional".
                  title={tab.judul_grid ?? tab.nama}
                  emptyMessage={emptyMessageFor(tab, filter)}
                  onMove={setPage}
                  describe={apiMessageOf}
                />
              </div>
            </>
          )}
        </>
      )}

      <NoteList lines={meta.data?.selisih_terencana ?? []} />
    </PageFrame>
  )
}

function PageFrame({ children }: Readonly<{ children: ReactNode }>) {
  return (
    <div className="mx-auto max-w-[96rem] px-4 py-8">
      <header className="flex flex-wrap items-start justify-between gap-4 border-b border-slate-200 pb-4">
        <div>
          {/*
            Judulnya "Claim Treatyin In Progress", bukan nama menunya. Itulah judul
            kontainer di `Section/InboxClaimTreaty_Section-Section.xml` dan itu pula yang
            terbaca di Pega produksi (`D-13`). Nama menunya — "Inbox Claim Treaty Prop" —
            tetap terlihat di menu kiri, tempat pengguna memilihnya.
          */}
          <h1 className="text-xl font-semibold text-slate-900">
            Claim Treatyin In Progress
          </h1>
          <p className="mt-1 text-sm text-slate-600">
            Antrean klaim treaty proporsional — klaim yang dialihkan perusahaan asuransi
            lain (Ceding Co) kepada ASM sebagai penanggung ulang.
          </p>
        </div>
        {/*
          Tombol "Create Claim Treaty Prop" DIGAMBAR meski belum membuat apa pun —
          keputusan Work Owner 2026-09-21. Di sistem lama ia memanggil
          `CreateInputKlaimTreaty`.
        */}
        <CreateClaimButton
          path="/api/inbox-claim-treaty-prop/klaim"
          label="Create Claim Treaty Prop"
          readyNotice="Pembuatan klaim treaty sudah tersedia. Muat ulang layar ini."
          describe={apiMessageOf}
          gapClassName="gap-2"
        />
      </header>
      {children}
    </div>
  )
}

/**
 * Penyaring di atas tabel.
 *
 * Kontrolnya digambar HANYA bila tab yang terbuka memang mengenalnya — dan itu dinyatakan
 * server, bukan ditebak layar. Checkbox yang tidak mengubah apa pun lebih buruk daripada
 * checkbox yang tidak ada.
 */
function FilterBar({
  tab,
  seeAll,
  onSeeAll,
}: Readonly<{
  tab: Tab
  seeAll: boolean
  onSeeAll: (value: boolean) => void
}>) {
  if (!tab.pakai_lihat_semua && !tab.hanya_milik_saya) return null

  return (
    <div className="mt-4 flex flex-wrap items-center gap-4">
      {tab.pakai_lihat_semua && (
        <InboxCheckbox label="See All Claim" checked={seeAll} onChange={onSeeAll} />
      )}

      {tab.hanya_milik_saya && !seeAll && (
        <p className="text-xs text-slate-500">
          Antrean ini hanya berisi pekerjaan milik Anda.
        </p>
      )}

      {seeAll && (
        <p className="text-xs text-amber-800">
          Menampilkan penugasan seluruh petugas, termasuk yang bukan milik Anda.
        </p>
      )}
    </div>
  )
}

/**
 * columnsFor menyusun kolom tabel dari bentuk yang ditetapkan server.
 *
 * SELURUH kolom datang dari server, tanpa kolom aksi tambahan di ujung. Yang membawa
 * pengguna ke rincian klaim adalah nomor klaim di kolom pertama: di Pega produksi,
 * mengklik Claim ID menjalankan Flow Action `OutstandingClaim` dan menggambar
 * `Section/OutstandingClaim-Section.xml`, yang di sini dilayani modul `outstanding-claim`.
 */
function columnsFor(tab: Tab): Column<WorkItem>[] {
  return serverColumns<WorkItem, TabColumn>(tab.kolom, cellText, {
    claim_id: (row) => <ClaimNumberLink value={row.claim_id} basePath="/outstanding-claim" />,
  })
}

/**
 * cellText menyusun teks satu sel.
 *
 * Tanggal Kejadian diformat HANYA bila bentuknya memang `YYYY-MM-DD`. Ia dibaca dari blob
 * JSON yang bentuknya tidak dapat diperiksa (`R-08`), sehingga memaksakan pemformatan akan
 * mengubah nilai yang tidak dikenali menjadi teks yang salah — lebih buruk daripada
 * menampilkannya apa adanya.
 *
 * Nilai kosong menjadi tanda pisah, bukan sel kosong yang tidak dapat dibedakan dari
 * kolom yang gagal dimuat.
 */
function cellText(row: WorkItem, column: TabColumn): string {
  const value = row[column.kunci]
  if (value == null || value === '') return '—'

  const text = String(value)
  return isISODate(text) ? formatDate(text) : text
}

/**
 * emptyMessageFor menjelaskan antrean kosong menurut sebabnya.
 *
 * "Tidak ada data" tidak cukup: antrean yang kosong karena pekerjaannya milik orang lain
 * jauh berbeda dari antrean yang memang tidak punya pekerjaan, dan tindakannya pun
 * berbeda.
 */
function emptyMessageFor(tab: Tab, filter: FilterForm): string {
  if (tab.hanya_milik_saya && !filter.lihatSemua) {
    return (
      'Tidak ada pekerjaan klaim treaty milik Anda. Centang "See All Claim" untuk ' +
      'melihat penugasan seluruh petugas.'
    )
  }
  return 'Antrean ini sedang kosong.'
}
