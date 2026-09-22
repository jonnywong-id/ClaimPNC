import { useState, type ReactNode } from 'react'
import { useNavigate } from 'react-router-dom'

import { APIError, callAPI } from '@/api/client'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'
import { Button } from '@/components/Button'
import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage } from '@/components/ErrorMessage'
import { formatDate } from '@/components/format'

import { NonPropTabs } from './NonPropTabs'
import {
  useClaimTreatyNonPropList,
  useClaimTreatyNonPropMetadata,
  useExportClaimTreatyNonProp,
} from './api'
import {
  EMPTY_FILTER,
  type FilterForm,
  type PageInfo,
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
 * Diambil dari `Section/InboxClaimNonProp_Harness-Section.xml` apa adanya: bilah tab,
 * tombol pembuat klaim, tombol ekspor, DUA checkbox, lalu grid berhalaman. Nama kolom
 * TIDAK diterjemahkan — `D-13` menetapkan tampilan meniru Pega, dan itulah teks yang
 * selama ini dibaca pengguna.
 *
 * # Kenapa kolomnya datang dari server
 *
 * Karena kedua tab yang terisi punya kolom yang berbeda, dan daftar itu adalah hasil
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

  if (portal === null) {
    return (
      <PageFrame filter={filter} exportable={false}>
        <ErrorMessage
          title="Pilih entitas lebih dulu"
          description={
            'Antrean klaim treaty milik satu badan hukum, dan aplikasi ini melayani ' +
            'empat. Pilih portal di bilah atas untuk membukanya.'
          }
          tone="gangguan"
        />
      </PageFrame>
    )
  }

  if (meta.isError) {
    return (
      <PageFrame filter={filter} exportable={false}>
        <ErrorMessage
          title="Layar tidak dapat dibuka"
          description={messageOf(meta.error)}
          tone="gangguan"
        />
      </PageFrame>
    )
  }

  return (
    <PageFrame filter={filter} exportable={!blocked && (list.data?.paginasi.total ?? 0) > 0}>
      <div className="mt-4">
        <NonPropTabs tabs={tabs} active={active} onSelect={selectTab} />
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
                <DataTable<WorkItem>
                  columns={columnsFor(tab, (row) => <DetailButton item={row} />)}
                  rows={list.data?.baris ?? []}
                  rowKey={(row) => `${row.referensi}|${row.no_klaim}`}
                  title={tab.nama}
                  // Kotak cari bawaan disembunyikan: hasilnya akan menyaring HANYA
                  // halaman yang sedang terbuka, sehingga pengguna dapat diberi tahu
                  // "tidak ada" untuk baris yang sebenarnya ada di halaman berikutnya.
                  //
                  // Layar lama pun tidak punya kotak cari: tak satu pun dari keempat
                  // kuerinya menyaring menurut kata kunci.
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
        </>
      )}

      <PlannedDifferences lines={meta.data?.selisih_terencana ?? []} />
    </PageFrame>
  )
}

function PageFrame({
  filter,
  exportable,
  children,
}: {
  filter: FilterForm
  /** Tombol ekspor hanya berguna bila ada yang dapat diekspor. */
  exportable: boolean
  children: ReactNode
}) {
  return (
    <div className="mx-auto max-w-[96rem] px-4 py-8">
      <header className="flex flex-wrap items-start justify-between gap-4 border-b border-slate-200 pb-4">
        <div>
          <h1 className="text-xl font-semibold text-slate-900">
            Inbox Claim Treaty Non Prop
          </h1>
          <p className="mt-1 text-sm text-slate-600">
            Antrean klaim treaty non-proporsional — klaim yang dialihkan perusahaan
            asuransi lain (Ceding Co) kepada ASM sebagai penanggung kerugian di atas batas
            tertentu.
          </p>
        </div>
        <div className="flex flex-col items-end gap-2">
          <div className="flex flex-wrap items-center justify-end gap-2">
            <ExportButton filter={filter} enabled={exportable} />
            <CreateClaimButton />
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
function ExportButton({ filter, enabled }: { filter: FilterForm; enabled: boolean }) {
  const ekspor = useExportClaimTreatyNonProp()

  return (
    <div className="flex flex-col items-end gap-1">
      <Button
        tone="kedua"
        disabled={!enabled || ekspor.isPending}
        onClick={() => ekspor.mutate(filter)}
      >
        {ekspor.isPending ? 'Menyiapkan berkas…' : 'Export Data'}
      </Button>
      {ekspor.isError && (
        <p className="max-w-md text-right text-xs text-red-700" role="alert">
          {messageOf(ekspor.error)}
        </p>
      )}
    </div>
  )
}

/**
 * Tombol "Create Claim Treaty Non Prop".
 *
 * Ia DIGAMBAR meski belum membuat apa pun, mengikuti perlakuan yang sama dengan layar
 * Treaty Prop. Di sistem lama ia memanggil `CreateClaimTNonProp_Act`, yang membuat objek
 * kerja baru di tabel yang selama masa paralel masih dimiliki Pega (`P-1`).
 *
 * Menyembunyikannya akan membuat pengguna mengira fiturnya hilang; menghidupkannya akan
 * membuat dua sistem menulis tabel yang sama. Yang dilakukan tombol ini adalah bertanya ke
 * server lalu menampilkan jawabannya — sehingga alasannya datang dari satu tempat, dan
 * hilang dengan sendirinya begitu kepemilikan tabelnya berpindah.
 */
function CreateClaimButton() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const [notice, setNotice] = useState('')
  const [asking, setAsking] = useState(false)

  async function ask() {
    setAsking(true)
    try {
      await callAPI('/api/inbox-claim-treaty-non-prop/klaim', {
        metode: 'POST',
        token,
        portal,
      })
      // Jalur ini tercapai hanya bila server SUDAH dapat membuat klaim. Selama itu belum
      // terjadi, ia tidak pernah berjalan — dan bila kelak berjalan, pesan di bawah yang
      // pertama memberi tahu bahwa perilakunya berubah.
      setNotice('Pembuatan klaim treaty non-prop sudah tersedia. Muat ulang layar ini.')
    } catch (error) {
      setNotice(messageOf(error))
    } finally {
      setAsking(false)
    }
  }

  return (
    <div className="flex flex-col items-end gap-1">
      <Button tone="utama" onClick={ask} disabled={asking || portal === null}>
        Create Claim Treaty Non Prop
      </Button>
      {notice !== '' && (
        <p className="max-w-md text-right text-xs text-amber-900" role="status">
          {notice}
        </p>
      )}
    </div>
  )
}

/**
 * Keterangan tab yang digambar tetapi belum dapat diisi.
 *
 * Alasan dan pemiliknya datang dari SERVER, bukan ditulis tetap di sini, supaya keduanya
 * hilang dengan sendirinya begitu penghalangnya hilang. Menyebut pemiliknya penting:
 * penghalang tanpa alamat tidak pernah hilang.
 */
function BlockedNotice({ tab }: { tab: Tab }) {
  return (
    <div className="mt-4 rounded-kartu border border-amber-200 bg-amber-50 px-4 py-4">
      <h2 className="text-sm font-semibold text-amber-900">{tab.nama} belum tersedia</h2>
      <p className="mt-2 text-sm text-slate-700">{tab.alasan_terhalang}</p>
      {tab.pemilik_penghalang && (
        <p className="mt-2 text-xs text-slate-600">
          <span className="font-medium">Menunggu:</span> {tab.pemilik_penghalang}
        </p>
      )}
    </div>
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
}: {
  tab: Tab
  filter: FilterForm
  onSeeAll: (value: boolean) => void
  onTBA: (value: boolean) => void
}) {
  if (!tab.pakai_lihat_semua && !tab.pakai_lihat_tba && !tab.hanya_milik_saya) return null

  return (
    <div className="mt-4 flex flex-col gap-2">
      <div className="flex flex-wrap items-center gap-4">
        {tab.pakai_lihat_semua && (
          <label className="flex items-center gap-2 text-sm text-slate-700">
            <input
              type="checkbox"
              checked={filter.lihatSemua}
              onChange={(event) => onSeeAll(event.target.checked)}
              className="size-4 rounded border-slate-300 text-blue-600 focus:ring-2 focus:ring-blue-500/30"
            />
            See All Claim
          </label>
        )}

        {tab.pakai_lihat_tba && (
          <label className="flex items-center gap-2 text-sm text-slate-700">
            <input
              type="checkbox"
              checked={filter.lihatTBA}
              onChange={(event) => onTBA(event.target.checked)}
              className="size-4 rounded border-slate-300 text-blue-600 focus:ring-2 focus:ring-blue-500/30"
            />
            See TBA Claim
          </label>
        )}
      </div>

      <p className="text-xs text-slate-500" role="status">
        {filterExplanation(tab, filter)}
      </p>
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
 * Tombol rincian klaim.
 *
 * Layar tujuannya adalah `MENU_ID 75` "View Claim" (`PNCViewClaim`) — modul tersendiri
 * yang belum dibangun. Tombolnya tetap dibangun mengikuti layar Treaty Prop, dan tujuannya
 * diarahkan ke rute yang sudah ada tempat keadaan itu dinyatakan apa adanya.
 *
 * Yang dikirim adalah `referensi`, kunci teknis Pega. Dengan begitu menyalakan layar
 * rincian kelak tidak menuntut perubahan kontrak API modul ini.
 */
function DetailButton({ item }: { item: WorkItem }) {
  const navigate = useNavigate()
  const key = item.referensi || item.no_klaim

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
 * Bentuknya sama dengan layar Treaty Prop dan Inbox Admin supaya ketiganya tidak terasa
 * dirakit dari tiga aplikasi berbeda.
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
 * Selisih terhadap Pega yang sudah diputuskan, ditampilkan di bawah tabel.
 *
 * Isinya datang dari SERVER, bukan ditulis tetap di sini. Tanpa catatan ini, checkbox TBA
 * yang kini berdiri sendiri dan kolom "Status" yang menyatakan antrean akan dilaporkan
 * berulang kali sebagai kerusakan oleh orang yang membandingkan kedua layar berdampingan.
 */
function PlannedDifferences({ lines }: { lines: string[] }) {
  if (lines.length === 0) return null

  return (
    <section className="mt-6 rounded-kartu border border-slate-200 bg-slate-50 px-4 py-3">
      <h2 className="text-sm font-medium text-slate-800">
        Yang berbeda dari layar lama, dan itu disengaja
      </h2>
      <ul className="mt-2 list-disc space-y-1 pl-5 text-xs text-slate-600">
        {lines.map((line) => (
          <li key={line}>{line}</li>
        ))}
      </ul>
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

/**
 * cellText menyusun teks satu sel.
 *
 * Tiga perlakuan yang berbeda, dan masing-masing punya alasannya:
 *
 *  - Aging adalah ANGKA. Nol hari adalah nilai yang sah dan berarti "masuk hari ini",
 *    sehingga ia TIDAK boleh berubah menjadi tanda pisah seperti isian teks yang kosong.
 *  - Tanggal Kejadian diformat HANYA bila bentuknya memang `YYYY-MM-DD`. Ia dibaca dari
 *    blob JSON yang bentuknya tidak dapat diperiksa (`R-08`), sehingga memaksakan
 *    pemformatan akan mengubah nilai yang tidak dikenali menjadi teks yang salah — lebih
 *    buruk daripada menampilkannya apa adanya.
 *  - Teks kosong menjadi tanda pisah, bukan sel kosong yang tidak dapat dibedakan dari
 *    kolom yang gagal dimuat.
 */
function cellText(row: WorkItem, column: TabColumn): string {
  const value = row[column.kunci]

  if (typeof value === 'number') {
    return `${value} hari`
  }

  if (value === null || value === undefined || value === '') return '—'

  const text = String(value)
  return isDate(text) ? formatDate(text) : text
}

/** isDate mengenali bentuk `YYYY-MM-DD`. */
function isDate(text: string): boolean {
  return /^\d{4}-\d{2}-\d{2}$/.test(text)
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

/** messageOf mengambil pesan yang layak dibaca pengguna dari sebuah galat. */
function messageOf(error: unknown): string {
  if (error instanceof APIError) return error.message
  if (error instanceof Error && error.message !== '') return error.message
  return 'Coba lagi beberapa saat lagi. Bila terus berulang, hubungi tim teknis.'
}
