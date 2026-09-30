import { type ReactNode } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'

import { APIError } from '@/api/client'
import { useSelectedPortal } from '@/app/portal'
import { Button } from '@/components/Button'
import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage } from '@/components/ErrorMessage'
import { formatDate } from '@/components/format'

import { ReceivePUCLTabs } from './ReceivePUCLTabs'
import {
  useExportManagerReceivePUCL,
  useManagerReceivePUCLList,
  useManagerReceivePUCLMetadata,
} from './api'
import type { Tab, TabColumn, WorkItem } from './types'

/**
 * Inbox Manager Receive / PUCL — menu `MENU_ID 56`, pengganti harness
 * `ReceiveDoucument_Harness`.
 *
 * Isinya pandangan PENYELIA atas dua antrean yang berbeda sama sekali, disatukan dalam satu
 * layar karena keduanya sama-sama menunggu tindakan manajer:
 *
 *   Receive    berkas penerimaan dokumen klaim yang masih punya penugasan terbuka
 *   RCL/PUCL   klaim yang ditolak (RCL) atau diproses ulang (PUCL)
 *
 * Per `D-79` keduanya benar-benar Inbox: barisnya diambil dari tabel penugasan Pega dan
 * hilang begitu penugasannya selesai. Karena itu modul ini milik `U-3`, bukan `U-6`.
 *
 * # Susunan layar, dan dari mana bentuknya
 *
 * Diambil dari `Section/InboxManagerReceive_Section-Section.xml`: bilah tab, lalu grid
 * berhalaman 50 baris. Nama kolom TIDAK diterjemahkan — `D-13` menetapkan tampilan meniru
 * Pega, dan itulah teks yang selama ini dibaca pengguna.
 *
 * Yang TIDAK ada di sana dan ditambahkan di sini: judul untuk kedua grid Receive, dan tombol
 * ekspor. Keduanya dinyatakan ke pengguna lewat panel "Yang berbeda dari layar lama" di
 * bawah tabel, bukan disimpan sebagai komentar.
 *
 * # Kenapa kolomnya datang dari server
 *
 * Karena ketiga tab punya kolom yang berbeda, dan daftar itu adalah hasil pembacaan export
 * Pega yang tercatat di backend. Menyalinnya ke sini berarti daftar yang sama hidup di dua
 * tempat — dan kedua tab Receive yang kolomnya identik akan menjadi tempat paling mudah
 * keduanya menyimpang tanpa ketahuan.
 */
export function ManagerReceivePUCLPage() {
  // Tab dan nomor halaman hidup di ALAMAT, bukan di state komponen.
  //
  // Alasannya satu dan nyata: mengklik nomor case membuka layar kerja di rute lain, dan
  // komponen ini dilepas. State yang hanya ada di memori akan hilang, sehingga petugas yang
  // kembali dari sebuah berkas mendarat di tab pertama halaman pertama — padahal ia sedang
  // mengerjakan halaman ketiga. Pada antrean yang dibuka berpuluh kali sehari, itu bukan
  // ketidaknyamanan kecil.
  const [params, setParams] = useSearchParams()

  const tabCode = params.get('tab') ?? ''
  const page = Math.max(1, Number(params.get('halaman') ?? '1') || 1)

  const navigate = useNavigate()

  const portal = useSelectedPortal((state) => state.alias)
  const meta = useManagerReceivePUCLMetadata()

  const tabs: Tab[] = meta.data?.tab ?? []
  const active = tabCode || meta.data?.tab_bawaan || ''
  const tab = tabs.find((candidate) => candidate.kode === active)

  // Tab terhalang TIDAK meminta isinya. Permintaannya pasti ditolak server dengan 422, dan
  // galat yang sudah diketahui pasti terjadi bukan galat yang layak ditampilkan — yang
  // layak ditampilkan adalah alasan terhalangnya.
  const blocked = tab?.terhalang === true
  const list = useManagerReceivePUCLList(active, page, meta.isSuccess && !blocked)

  /**
   * Berpindah tab mengembalikan ke halaman pertama.
   *
   * Tanpa itu, berpindah dari halaman 4 sebuah tab ke tab yang hanya punya 2 halaman akan
   * menampilkan tabel kosong yang terbaca seperti antrean yang memang kosong.
   */
  function selectTab(code: string) {
    setParams({ tab: code, halaman: '1' }, { replace: true })
  }

  /** Berpindah halaman, menjaga tab yang sedang terbuka. */
  function setPage(next: number) {
    setParams({ tab: active, halaman: String(next) }, { replace: true })
  }

  /**
   * Membuka LAYAR KERJA penerimaan dokumen — yang di Pega dijalankan Open Assignment.
   *
   * Tab dan halaman ikut ke alamat tujuan supaya tombol kembali mendarat di tempat yang
   * sama. Kuncinya dikodekan: `pzInsKey` memuat SPASI
   * (`ASM-FW-GCNMFW-WORK RCV-900001`), dan spasi mentah di dalam alamat bukan alamat yang
   * sah.
   */
  function openDocument(row: WorkItem) {
    const back = new URLSearchParams({ tab: active, halaman: String(page) })
    navigate(
      `/inbox-manager-receive-pucl/dokumen/${encodeURIComponent(row.referensi)}?${back}`,
    )
  }

  if (portal === null) {
    return (
      <PageFrame tab={active} exportable={false}>
        <ErrorMessage
          title="Pilih entitas lebih dulu"
          description={
            'Antrean penerimaan dokumen dan RCL/PUCL milik satu badan hukum, dan aplikasi ' +
            'ini melayani empat. Pilih portal di bilah atas untuk membukanya.'
          }
          tone="gangguan"
        />
      </PageFrame>
    )
  }

  if (meta.isError) {
    return (
      <PageFrame tab={active} exportable={false}>
        <ErrorMessage
          title="Layar tidak dapat dibuka"
          description={messageOf(meta.error)}
          tone="gangguan"
        />
      </PageFrame>
    )
  }

  return (
    <PageFrame
      tab={active}
      exportable={!blocked && (list.data?.paginasi.total ?? 0) > 0}
    >
      <div className="mt-4">
        <ReceivePUCLTabs tabs={tabs} active={active} onSelect={selectTab} />
      </div>

      {tab && (
        <>
          <p className="mt-3 text-sm text-slate-600">{tab.keterangan}</p>

          {tab.terhalang ? (
            <BlockedNotice tab={tab} />
          ) : (
            <div className="mt-4">
              <DataTable<WorkItem>
                columns={columnsFor(tab, (row) => (
                  <CaseLink item={row} onOpen={openDocument} />
                ))}
                rows={list.data?.baris ?? []}
                rowKey={(row) => `${row.referensi}|${row.no_case}`}
                title={tab.nama}
                label={`Antrean ${tab.nama}`}
                // Kotak cari bawaan disembunyikan: hasilnya akan menyaring HANYA halaman
                // yang sedang terbuka, sehingga pengguna dapat diberi tahu "tidak ada"
                // untuk baris yang sebenarnya ada di halaman berikutnya.
                //
                // Layar lama pun tidak punya kotak cari: ketiga Report Definition-nya
                // tidak menyaring menurut kata kunci sama sekali.
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
                emptyMessage={emptyMessageFor(tab)}
                pagination={{
                  page: list.data?.paginasi.halaman ?? 1,
                  size: list.data?.paginasi.ukuran ?? 50,
                  total: list.data?.paginasi.total ?? 0,
                  totalPage: list.data?.paginasi.total_halaman ?? 1,
                  onPageChange: setPage,
                  isLoading: list.isFetching,
                }}
              />
            </div>
          )}
        </>
      )}

      <PlannedDifferences lines={meta.data?.selisih_terencana ?? []} />
    </PageFrame>
  )
}

function PageFrame({
  tab,
  exportable,
  children,
}: {
  tab: string
  /** Tombol ekspor hanya berguna bila ada yang dapat diekspor. */
  exportable: boolean
  children: ReactNode
}) {
  return (
    <div className="mx-auto max-w-[96rem] px-4 py-8">
      <header className="flex flex-wrap items-start justify-between gap-4 border-b border-slate-200 pb-4">
        <div>
          <h1 className="text-xl font-semibold text-slate-900">
            Inbox Manager Receive / PUCL
          </h1>
          <p className="mt-1 text-sm text-slate-600">
            Pandangan penyelia atas dua antrean: berkas penerimaan dokumen klaim yang masih
            punya penugasan terbuka, dan klaim yang ditolak atau diproses ulang.
          </p>
          {/*
            Sifat "pandangan penyelia" dinyatakan di layar, bukan hanya di kode.

            Tidak satu pun tab di sini menyaring menurut pengguna yang login — berbeda dari
            seluruh layar inbox lain, yang setidaknya punya satu tab "milik saya". Tanpa
            keterangan ini, petugas yang terbiasa dengan layar inbox lain akan mengira
            daftarnya keliru karena memuat pekerjaan orang lain.
          */}
          <p className="mt-2 text-xs text-slate-500">
            Layar ini menampilkan pekerjaan <span className="font-medium">seluruh
            petugas</span> pada portal yang sedang dipilih, bukan hanya milik Anda. Setiap
            pembukaannya tercatat.
          </p>
        </div>
        <ExportButton tab={tab} enabled={exportable} />
      </header>
      {children}
    </div>
  )
}

/**
 * Tombol ekspor.
 *
 * Ia KEMAMPUAN BARU: layar lama tidak punya tombol ekspor sama sekali — tidak ada activity
 * ekspor yang dirujuk harness maupun section-nya. Penambahannya diputuskan Work Owner
 * 2026-09-22, dan dinyatakan ke pengguna lewat panel selisih terencana di bawah tabel.
 *
 * Tombolnya dimatikan saat tidak ada yang dapat diekspor. Berkas kosong yang tetap terunduh
 * adalah jawaban yang membingungkan: pengguna tidak dapat membedakannya dari ekspor yang
 * gagal diam-diam.
 */
function ExportButton({ tab, enabled }: { tab: string; enabled: boolean }) {
  const ekspor = useExportManagerReceivePUCL()

  return (
    <div className="flex flex-col items-end gap-1">
      <Button
        tone="kedua"
        disabled={!enabled || ekspor.isPending}
        onClick={() => ekspor.mutate(tab)}
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
 * Tautan pada kolom nomor case.
 *
 * # Kenapa TAUTAN pada nomor case, bukan tombol di ujung baris
 *
 * Karena begitulah layar lama. `Section/InboxManagerReceive_Section-Section.xml` menggambar
 * sel nomor case pada kedua grid Receive sebagai `pyUIElement = link` ber-`pyLabel = .pyID`,
 * dan `D-13` menetapkan tampilan mengikuti Pega.
 *
 * Versi pertama modul ini keliru di sini: nomor case digambar sebagai teks biasa, dan di
 * ujung baris ditambahkan kolom tombol "Lihat Detail" yang **tidak ada di Pega sama sekali**
 * — menunjuk rute `/view-claim/…` milik modul lain yang belum dibangun. Itu bukan sekadar
 * beda tampilan: kolom yang tidak pernah ada membuat pengguna mengira ada dua cara berbeda
 * membuka baris, dan menggeser lebar seluruh kolom lain.
 *
 * # Apa yang sebenarnya terjadi saat diklik di Pega
 *
 * Tiga perilaku berurutan pada sel yang sama:
 *
 *   1. `runActivity`    `SetAssignmentInboxReceive_act`, parameter `kunci = .pzInsKey`
 *   2. `refresh`        thisSection
 *   3. `openAssignment` `pyInsKey = TempIns.pyNote`
 *
 * Activity-nya sendiri nyaris kosong — `pyUsage = FLOW`, satu `Property-Set` ke
 * `TempIns.pyNote`. Ia kait pra-proses; yang bekerja adalah **Open Assignment** bawaan Pega,
 * yang membuka berkas penerimaan dokumen pada tahap alur kerjanya saat itu — dan flow action
 * yang menunggu di sana adalah `InputReceiveDocument`.
 *
 * # Hanya tab Receive
 *
 * `buka_layar_kerja` datang dari server, dan hanya tab Receive yang membawanya. Di layar lama
 * pun perilaku klik hanya dipasang pada kedua grid Receive; nomor case di grid RCL/PUCL
 * bukan tautan sama sekali.
 */
function CaseLink({
  item,
  onOpen,
}: {
  item: WorkItem
  onOpen: (row: WorkItem) => void
}) {
  if (item.no_case === '') return <span className="text-slate-400">—</span>

  // Tanpa kunci teknis, Open Assignment tidak punya apa pun untuk dibuka. Nomornya tetap
  // digambar sebagai teks alih-alih sebagai tautan yang pasti gagal.
  if (item.referensi === '') return <span>{item.no_case}</span>

  return (
    <button
      type="button"
      onClick={() => onOpen(item)}
      title={`Buka berkas ${item.no_case}`}
      className={[
        'rounded-kontrol text-left font-medium text-blue-700 underline-offset-2',
        'transition-colors duration-150 ease-halus hover:underline',
        'focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500/50',
      ].join(' ')}
    >
      {item.no_case}
    </button>
  )
}

/**
 * Selisih terhadap Pega yang sudah diputuskan, ditampilkan di bawah tabel.
 *
 * Isinya datang dari SERVER, bukan ditulis tetap di sini. Tanpa catatan ini, tiga hal akan
 * dilaporkan berulang kali sebagai kerusakan oleh orang yang membandingkan kedua layar
 * berdampingan: kolom "Jumlah Lembar Dokumen" yang selalu kosong, kolom "Jenis Klaim" yang
 * kini diturunkan dari Group Panel, dan tab RCL/PUCL yang isinya jauh lebih sedikit daripada
 * yang dikembalikan Report Definition aslinya.
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
 * # TIDAK ada kolom aksi tambahan
 *
 * Layar lama tidak punya satu pun, dan versi pertama modul ini keliru menambahkannya. Yang
 * membuka baris adalah **tautan pada sel nomor case** — itulah bentuknya di kedua grid
 * Receive (`D-13`).
 *
 * `value` tetap mengembalikan TEKS polos meski selnya digambar sebagai tautan. Keduanya
 * dipisah dengan sengaja oleh `DataTable`: yang dicari dan diurutkan adalah teksnya, yang
 * dilihat pengguna adalah gambarnya. Menyatukannya akan membuat pengurutan menelusuri markup
 * alih-alih nomor case.
 */
function columnsFor(tab: Tab, openCase: (row: WorkItem) => ReactNode): Column<WorkItem>[] {
  return tab.kolom.map((column) => {
    const base: Column<WorkItem> = {
      key: column.kunci,
      title: column.judul,
      value: (row) => cellText(row, column),
    }

    // Hanya tab Receive yang nomor case-nya membuka layar kerja. Penandanya datang dari
    // server, bukan disimpulkan dari kode tab di sini.
    if (column.kunci === 'no_case' && tab.buka_layar_kerja) {
      return { ...base, render: openCase }
    }
    return base
  })
}

/**
 * cellText menyusun teks satu sel.
 *
 * Dua perlakuan, dan masing-masing punya alasannya:
 *
 *  - Tanggal diformat HANYA bila bentuknya memang `YYYY-MM-DD`. Seluruh kolom tanggal di
 *    layar ini dibaca dari kolom yang bentuknya tidak dapat diperiksa tanpa DDL (`R-08`) —
 *    satu di antaranya bahkan disimpan sebagai teks oleh procedure yang mengisinya. Memaksa
 *    pemformatan akan mengubah nilai yang tidak dikenali menjadi teks yang salah, dan itu
 *    lebih buruk daripada menampilkannya apa adanya.
 *  - Teks kosong menjadi tanda pisah, bukan sel kosong yang tidak dapat dibedakan dari
 *    kolom yang gagal dimuat. Di layar ini itu penting khusus: satu kolom memang SELALU
 *    kosong, dan tanda pisah menyatakan "tidak ada isinya" alih-alih "gagal".
 */
function cellText(row: WorkItem, column: TabColumn): string {
  const value = row[column.kunci]

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
 * "Tidak ada data" tidak cukup di layar ini. Ketiga tab bisa kosong karena sebab yang
 * berbeda, dan dua di antaranya menunjuk KEKELIRUAN KONFIGURASI alih-alih antrean yang
 * memang sepi — Group Panel yang berbeda di produksi, atau akun antrean bersama yang
 * berganti nama. Keduanya tidak menghasilkan satu pun galat, sehingga keterangan inilah
 * satu-satunya yang mengarahkan pengguna melapor alih-alih menganggapnya wajar.
 */
function emptyMessageFor(tab: Tab): string {
  if (tab.antrean_bersama) {
    return (
      'Tidak ada klaim RCL/PUCL yang menunggu. Bila Anda yakin seharusnya ada, laporkan ' +
      'ke tim teknis — antrean ini disaring menurut akun antrean bersama, dan perubahan ' +
      'nama akun itu membuat daftarnya kosong tanpa pesan galat.'
    )
  }

  return (
    'Tidak ada berkas penerimaan dokumen pada lini bisnis ini. Bila Anda yakin seharusnya ' +
    'ada, periksa tab sebelahnya lebih dulu: kedua tab Receive dipisahkan Group Panel, ' +
    'dan berkas yang Group Panel-nya berbeda akan muncul di tab yang satunya.'
  )
}

/** messageOf mengambil pesan yang layak dibaca pengguna dari sebuah galat. */
function messageOf(error: unknown): string {
  if (error instanceof APIError) return error.message
  if (error instanceof Error && error.message !== '') return error.message
  return 'Coba lagi beberapa saat lagi. Bila terus berulang, hubungi tim teknis.'
}
