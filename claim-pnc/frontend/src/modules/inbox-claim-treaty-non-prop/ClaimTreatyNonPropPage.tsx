import { useState, type ReactNode } from 'react'

import { useSelectedPortal } from '@/app/portal'
import { type Column } from '@/components/DataTable'
import { formatDate } from '@/components/format'
import { ClaimNumberLink } from '@/components/inbox/ClaimNumberLink'
import { CreateClaimButton } from '@/components/inbox/CreateClaimButton'
import { ExportDataButton } from '@/components/inbox/ExportDataButton'
import { InboxCheckbox } from '@/components/inbox/InboxCheckbox'
import { BlockedNotice, NoteList, screenGate } from '@/components/inbox/InboxNotices'
import { QueueTable } from '@/components/inbox/QueueTable'
import { errorMessageOf, isISODate } from '@/components/inbox/messages'
import { serverColumns } from '@/components/inbox/serverColumns'

import { NonPropViewSelect } from './NonPropViewSelect'
import {
  useClaimTreatyNonPropList,
  useClaimTreatyNonPropMetadata,
  useExportClaimTreatyNonProp,
} from './api'
import {
  EMPTY_FILTER,
  type FilterForm,
  type Tab,
  type TabColumn,
  type WorkItem,
} from './types'

/**
 * Inbox Claim Treaty Non Prop — menu `MENU_ID 55`, pengganti harness
 * `InboxClaimNonProp_Harness`.
 *
 * Isinya daftar pekerjaan klaim treaty NON-proporsional: klaim yang masuk lewat jalur
 * treaty inward non-proporsional, tempat ASM menanggung kerugian di atas batas tertentu
 * atas klaim milik perusahaan asuransi lain (Ceding Co). Per `D-79` ia benar-benar Inbox:
 * barisnya pekerjaan dari tabel penugasan Pega, dan hilang begitu penugasannya selesai.
 *
 * Menu `MENU_ID 54` "Inbox Claim Treaty Prop" adalah layar SAUDARA yang berdiri sendiri.
 * Keduanya mirip di layar tetapi membaca tabel, kolom, dan penanda objek kerja yang
 * berbeda — lihat `internal/inboxclaimtreatynonprop`.
 *
 * # Susunan layar, dan dari mana bentuknya
 *
 * Diambil dari `Section/InboxClaimNonProp_Harness-Section.xml` apa adanya: judul "Claims In
 * Progress", tombol pembuat klaim, tombol ekspor, dropdown pemilih antrean, DUA checkbox,
 * lalu grid berhalaman yang punya judulnya sendiri. Nama kolom TIDAK diterjemahkan —
 * `D-13` menetapkan tampilan meniru Pega, dan itulah teks yang selama ini dibaca pengguna.
 *
 * Bentuk itu berbeda dari versi sebelumnya di sini pada tiga hal, dan ketiganya karena
 * section-nya dibaca ulang sel demi sel:
 *
 *   - pemilih antreannya DROPDOWN, bukan bilah tab;
 *   - urutan kolomnya mengikuti urutan SEL, bukan urutan kolom kuerinya — nomor polis
 *     berada sesudah Ceding Co, bukan di tengah;
 *   - kolom berjudul "Status" berisi WAKTU objek kerja dibuat, bukan teks tetap per
 *     antrean, dan hanya ada di grid Teknik.
 *
 * # Nomor klaim adalah TAUTAN, bukan tombol di kolom terakhir
 *
 * Sel Claim.ID di kedua grid ber-`pyAction openAssignment`: mengkliknya membuka objek
 * kerjanya. Tombol "Lihat Detail Klaim" di kolom tambahan karena itu DIHAPUS — ia kolom
 * ke-13 yang tidak ada di Pega, sementara tautannya menempati kolom yang memang sudah ada.
 *
 * # Kenapa kolomnya datang dari server
 *
 * Karena kedua antrean yang terisi punya kolom yang berbeda, dan daftar itu adalah hasil
 * pembacaan export Pega yang tercatat di backend. Menyalinnya ke sini berarti daftar yang
 * sama hidup di dua tempat.
 */
export function ClaimTreatyNonPropPage() {
  const [filter, setFilter] = useState<FilterForm>(EMPTY_FILTER)
  const [page, setPage] = useState(1)

  const portal = useSelectedPortal((state) => state.alias)
  const meta = useClaimTreatyNonPropMetadata()

  const tabs: Tab[] = meta.data?.tab ?? []
  const active = filter.tab || meta.data?.tab_bawaan || ''
  const tab = tabs.find((candidate) => candidate.kode === active)

  // Tab terhalang TIDAK meminta isinya. Permintaannya pasti ditolak server dengan 422,
  // dan galat yang sudah diketahui pasti terjadi bukan galat yang layak ditampilkan —
  // yang layak ditampilkan adalah alasan terhalangnya.
  const blocked = tab?.terhalang === true
  const list = useClaimTreatyNonPropList(filter, page, meta.isSuccess && !blocked)

  /**
   * Berpindah tab MEMBERSIHKAN kedua checkbox dan mengembalikan ke halaman pertama.
   *
   * Membawa centang ke tab yang tidak mengenalnya akan membuat centang yang tampak aktif
   * padahal tidak mengubah apa pun.
   */
  function selectTab(code: string) {
    setFilter({ ...EMPTY_FILTER, tab: code })
    setPage(1)
  }

  function toggleSeeAll(value: boolean) {
    setFilter((previous) => ({ ...previous, lihatSemua: value }))
    setPage(1)
  }

  function toggleTBA(value: boolean) {
    setFilter((previous) => ({ ...previous, lihatTBA: value }))
    setPage(1)
  }

  const gate = screenGate({
    portal,
    subject: 'Antrean klaim treaty',
    failed: meta.isError,
    error: meta.error,
    describe: errorMessageOf,
  })
  if (gate) {
    return (
      <PageFrame filter={filter} exportable={false}>
        {gate}
      </PageFrame>
    )
  }

  return (
    <PageFrame filter={filter} exportable={!blocked && (list.data?.paginasi.total ?? 0) > 0}>
      <div className="mt-4">
        <NonPropViewSelect tabs={tabs} active={active} onSelect={selectTab} />
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
                filter={filter}
                onSeeAll={toggleSeeAll}
                onTBA={toggleTBA}
              />

              <div className="mt-4">
                <QueueTable<WorkItem>
                  query={list}
                  columns={columnsFor(tab)}
                  rowKey={(row) => `${row.referensi}|${row.no_klaim}`}
                  // Judul GRID, bukan teks pilihan dropdown. Keduanya tampil bersamaan dan
                  // berbunyi berbeda: dropdown "Treaty-In Teknik", grid "Work Treatyin Non
                  // Propotional Teknik".
                  title={tab.judul_grid ?? tab.nama}
                  emptyMessage={emptyMessageFor(tab, filter)}
                  onMove={setPage}
                  describe={errorMessageOf}
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

function PageFrame({
  filter,
  exportable,
  children,
}: Readonly<{
  filter: FilterForm
  /** Tombol ekspor hanya berguna bila ada yang dapat diekspor. */
  exportable: boolean
  children: ReactNode
}>) {
  return (
    <div className="mx-auto max-w-[96rem] px-4 py-8">
      <header className="flex flex-wrap items-start justify-between gap-4 border-b border-slate-200 pb-4">
        <div>
          {/*
            Judulnya "Claims In Progress", bukan nama menunya. Itulah judul kontainer di
            `Section/InboxClaimNonProp_Harness-Section.xml` dan itu pula yang terbaca di
            Pega (`D-13`). Nama menunya — "Inbox Claim Treaty Non Prop" — tetap terlihat di
            menu kiri, tempat pengguna memilihnya.
          */}
          <h1 className="text-xl font-semibold text-slate-900">Claims In Progress</h1>
          <p className="mt-1 text-sm text-slate-600">
            Antrean klaim treaty non-proporsional — klaim yang dialihkan perusahaan
            asuransi lain (Ceding Co) kepada ASM sebagai penanggung kerugian di atas batas
            tertentu.
          </p>
        </div>
        <div className="flex flex-col items-end gap-2">
          <div className="flex flex-wrap items-center justify-end gap-2">
            <ExportButton filter={filter} enabled={exportable} />
            {/*
              "Create Claim Treaty Non Prop" DIGAMBAR meski belum membuat apa pun, mengikuti
              perlakuan layar Treaty Prop. Di sistem lama ia memanggil
              `CreateClaimTNonProp_Act`.
            */}
            <CreateClaimButton
              path="/api/inbox-claim-treaty-non-prop/klaim"
              label="Create Claim Treaty Non Prop"
              readyNotice="Pembuatan klaim treaty non-prop sudah tersedia. Muat ulang layar ini."
              describe={errorMessageOf}
              gapClassName="gap-1"
            />
          </div>
        </div>
      </header>
      {children}
    </div>
  )
}

/**
 * Tombol ekspor.
 *
 * Asalnya `Activity/GenerateClaimNonPropCSV-Act.xml`. Ia BERFUNGSI penuh — berbeda dari
 * tombol buat klaim di sebelahnya — karena mengekspor hanya membaca, dan membaca tidak
 * melanggar kepemilikan tabel (`P-1`).
 *
 * Tombolnya dimatikan saat tidak ada yang dapat diekspor. Berkas kosong yang tetap
 * terunduh adalah jawaban yang membingungkan: pengguna tidak dapat membedakannya dari
 * ekspor yang gagal diam-diam.
 */
function ExportButton({ filter, enabled }: Readonly<{ filter: FilterForm; enabled: boolean }>) {
  const ekspor = useExportClaimTreatyNonProp()

  return (
    <ExportDataButton
      state={ekspor}
      onExport={() => ekspor.mutate(filter)}
      enabled={enabled}
      describe={errorMessageOf}
    />
  )
}

/**
 * Penyaring di atas tabel.
 *
 * Kontrolnya digambar HANYA bila tab yang terbuka memang mengenalnya — dan itu dinyatakan
 * server, bukan ditebak layar. Checkbox yang tidak mengubah apa pun lebih buruk daripada
 * checkbox yang tidak ada.
 *
 * # Kedua checkbox saling bebas, dan itu selisih terencana
 *
 * Di sistem lama "See TBA Claim" hanya berpengaruh bila "See All Claim" ikut dicentang.
 * Keterangan di bawah karena itu menyebutkan apa yang sedang berlaku — bukan hanya
 * menandai centangnya — supaya pengguna yang terbiasa dengan perilaku lama tidak mengira
 * layar ini keliru.
 */
function FilterBar({
  tab,
  filter,
  onSeeAll,
  onTBA,
}: Readonly<{
  tab: Tab
  filter: FilterForm
  onSeeAll: (value: boolean) => void
  onTBA: (value: boolean) => void
}>) {
  if (!tab.pakai_lihat_semua && !tab.pakai_lihat_tba && !tab.hanya_milik_saya) return null

  return (
    <div className="mt-4 flex flex-col gap-2">
      <div className="flex flex-wrap items-center gap-4">
        {tab.pakai_lihat_semua && (
          <InboxCheckbox label="See All Claim" checked={filter.lihatSemua} onChange={onSeeAll} />
        )}

        {tab.pakai_lihat_tba && (
          <InboxCheckbox label="See TBA Claim" checked={filter.lihatTBA} onChange={onTBA} />
        )}
      </div>

      <output className="block text-xs text-slate-500">
        {filterExplanation(tab, filter)}
      </output>
    </div>
  )
}

/**
 * filterExplanation menyusun satu kalimat tentang apa yang sedang ditampilkan.
 *
 * Empat kombinasi checkbox menghasilkan empat isi yang berbeda, dan tanpa kalimat ini
 * pengguna harus menyimpulkannya sendiri dari dua centang. Pada layar yang salah bacanya
 * berarti mengira antrean kosong, itu terlalu mahal untuk dibiarkan disimpulkan.
 */
function filterExplanation(tab: Tab, filter: FilterForm): string {
  const milik = filter.lihatSemua ? 'seluruh petugas' : 'Anda'

  if (filter.lihatTBA) {
    return (
      `Menampilkan pekerjaan milik ${milik} yang nomor polisnya BELUM terbit. ` +
      'Di layar Pega, "See TBA Claim" hanya berpengaruh bila "See All Claim" ikut ' +
      'dicentang; di sini keduanya berdiri sendiri.'
    )
  }

  if (filter.lihatSemua) {
    return 'Menampilkan penugasan seluruh petugas, termasuk yang bukan milik Anda.'
  }

  if (tab.hanya_milik_saya) {
    return 'Antrean ini hanya berisi pekerjaan milik Anda.'
  }

  return 'Antrean bersama — belum diambil siapa pun.'
}

/**
 * Nomor klaim sebagai tautan ke layar Acceptation Claim.
 *
 * Sel Claim.ID di kedua grid Pega ber-`pyAction openAssignment` — mengkliknya membuka objek
 * kerjanya, dan Flow Action yang digambar untuk objek kerja itu adalah `InputAcceptation`:
 * satu-satunya Flow Action yang terdaftar pada kelas
 * `ASM-FW-GCNMFW-Work-ClaimTreatyNonProp`. Padanannya di sini adalah modul
 * `input-acceptation`.
 *
 * Sebelumnya tautan ini menunjuk `/view-claim/:referensi`, penampung layar "View Claim" yang
 * belum dibangun — sehingga setiap klik berakhir di pemberitahuan "layar belum dibangun".
 * Tujuannya dibetulkan setelah Flow Action-nya ditelusuri (2026-09-30).
 *
 * # Yang dikirim adalah NOMOR KLAIM, bukan kunci teknis Pega
 *
 * Alamatnya terbaca orang (`/input-acceptation/CLMNP-232`), dapat disalin ke percakapan, dan
 * tidak membocorkan bentuk kunci internal Pega ke bilah alamat. Itu mengikuti layar saudaranya
 * `outstanding-claim`, yang memakai nomor klaim dengan alasan yang sama. Kunci teknisnya tetap
 * dikirim server pada setiap baris, sehingga beralih memakainya kelak tidak menuntut perubahan
 * kontrak.
 *
 * # Baris tanpa nomor klaim tidak menjadi tautan
 *
 * Tautan yang alamatnya kosong tetap dapat diklik dan membawa pengguna ke layar yang pasti
 * gagal. Yang digambar untuk baris seperti itu adalah tanda pisah.
 */
function ClaimIDLink({ item }: Readonly<{ item: WorkItem }>) {
  return <ClaimNumberLink value={item.no_klaim} basePath="/input-acceptation" />
}

/**
 * columnsFor menyusun kolom tabel dari bentuk yang ditetapkan server.
 *
 * SELURUH kolom datang dari server, tanpa kolom aksi tambahan di ujung. Yang membawa
 * pengguna ke rincian klaim adalah nomor klaim di kolom pertama — lihat ClaimIDLink.
 *
 * Kolom pertama itu tetap punya `value` berupa teks polos. `Column.value` adalah yang
 * dicari dan diurutkan, sedangkan `render` yang dilihat; menyatukannya akan membuat
 * pengurutan menelusuri markup tautannya, bukan nomornya.
 */
function columnsFor(tab: Tab): Column<WorkItem>[] {
  return serverColumns<WorkItem, TabColumn>(tab.kolom, cellText, {
    no_klaim: (row) => <ClaimIDLink item={row} />,
  })
}

/**
 * cellText menyusun teks satu sel.
 *
 * Tiga perlakuan yang berbeda, dan masing-masing punya alasannya:
 *
 *  - Aging adalah ANGKA, digambar apa adanya tanpa satuan — persis seperti di Pega. Nol
 *    hari adalah nilai yang sah dan berarti "masuk hari ini", sehingga ia TIDAK boleh
 *    berubah menjadi tanda pisah seperti isian teks yang kosong. Satuannya disebutkan di
 *    keterangan antrean dan di daftar selisih terencana, bukan diulang pada setiap baris.
 *  - Tanggal Kejadian diformat HANYA bila bentuknya memang `YYYY-MM-DD`. Ia dibaca dari
 *    blob JSON yang bentuknya tidak dapat diperiksa (`R-08`), sehingga memaksakan
 *    pemformatan akan mengubah nilai yang tidak dikenali menjadi teks yang salah — lebih
 *    buruk daripada menampilkannya apa adanya. Kolom "Status" lolos syarat itu dengan
 *    sendirinya: `20240201T095612.955 GMT` bukan `YYYY-MM-DD`, jadi ia tampil apa adanya.
 *  - Teks kosong menjadi tanda pisah, bukan sel kosong yang tidak dapat dibedakan dari
 *    kolom yang gagal dimuat.
 */
function cellText(row: WorkItem, column: TabColumn): string {
  const value = row[column.kunci]

  if (typeof value === 'number') {
    return String(value)
  }

  if (value == null || value === '') return '—'

  const text = String(value)
  return isISODate(text) ? formatDate(text) : text
}

/**
 * emptyMessageFor menjelaskan antrean kosong menurut sebabnya.
 *
 * "Tidak ada data" tidak cukup: antrean yang kosong karena pekerjaannya milik orang lain
 * jauh berbeda dari antrean yang memang tidak punya pekerjaan, dan tindakannya pun
 * berbeda. Pada layar berisi dua checkbox, sebabnya bahkan ada empat.
 */
function emptyMessageFor(tab: Tab, filter: FilterForm): string {
  if (filter.lihatTBA) {
    const cakupan = filter.lihatSemua ? '' : ' Centang "See All Claim" untuk memperluas.'
    return (
      'Tidak ada klaim yang nomor polisnya belum terbit pada penyaring ini.' + cakupan
    )
  }

  if (tab.hanya_milik_saya && !filter.lihatSemua) {
    return (
      'Tidak ada pekerjaan klaim treaty non-prop milik Anda. Centang "See All Claim" ' +
      'untuk melihat penugasan seluruh petugas.'
    )
  }

  return 'Antrean ini sedang kosong.'
}

