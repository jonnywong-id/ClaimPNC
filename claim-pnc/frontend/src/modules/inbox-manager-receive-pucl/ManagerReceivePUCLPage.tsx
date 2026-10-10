import { type ReactNode } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'

import { APIError } from '@/api/client'
import { useSelectedPortal } from '@/app/portal'

import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage } from '@/components/ErrorMessage'
import { formatPegaDateTime } from '@/components/format'

import { ReceivePUCLTabs } from './ReceivePUCLTabs'
import { useManagerReceivePUCLList, useManagerReceivePUCLMetadata } from './api'
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

  /**
   * Membuka LAYAR KERJA klaim RCL/PUCL — padanan Open Assignment pada grid RCL/PUCL.
   *
   * Yang dialamatkan NOMOR CASE, bukan `referensi`. Layar tujuannya membaca
   * `POOLDATA.TC_PNC_PUCL` yang kuncinya `CLAIMID` — nomor case telanjang (`PNC-2266`) —
   * sedangkan `referensi` di tab ini berisi `PZINSKEY` (`ASM-FW-GCNMFW-WORK PNC-2266`).
   * Mengirim yang kedua menghasilkan "tidak ditemukan" untuk klaim yang sebenarnya ada.
   */
  function openClaim(row: WorkItem) {
    navigate(`/inbox-rcl-pucl/klaim/${encodeURIComponent(row.no_case)}`)
  }

  if (portal === null) {
    return (
      <PageFrame>
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
                  <CaseLink
                    item={row}
                    onOpen={tab.buka_layar_klaim ? openClaim : openDocument}
                  />
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

    </PageFrame>
  )
}

/**
 * Kepala layar.
 *
 * TIDAK ada tombol di sini, dan itu keputusan — bukan kelalaian. Versi sebelumnya memasang
 * tombol "Export Data" di sudut kanan; layar lama tidak punya padanannya sama sekali, dan
 * `D-13` menetapkan tampilan mengikuti Pega. Dicabut atas keputusan Work Owner 2026-10-10.
 */
function PageFrame({ children }: { children: ReactNode }) {
  return (
    <div className="mx-auto max-w-[96rem] px-4 py-8">
      <header className="border-b border-slate-200 pb-4">
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
      </header>
      {children}
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
 * Tiga perilaku berurutan pada sel yang sama, dan susunannya SAMA di kedua grid — hanya
 * activity dan kelas objek kerjanya yang berbeda:
 *
 *   grid        activity                        kelas
 *   ----------- ------------------------------- ----------------------------------
 *   Receive     `SetAssignmentInboxReceive_act`  ASM-FW-GCNMFW-Work-ReceiveDocument
 *   RCL/PUCL    `SetAssignmentInboxPUCL_act`     ASM-FW-GCNMFW-Work-PNC
 *
 * Keduanya diikuti `refresh` thisSection lalu `openAssignment`. Activity-nya sendiri nyaris
 * kosong — satu `Property-Set` ke `TempIns.pyNote`; ia kait pra-proses, dan yang bekerja
 * adalah **Open Assignment** bawaan Pega.
 *
 * # Koreksi: tab RCL/PUCL PUNYA tautan
 *
 * Versi sebelumnya menggambar nomor case tab itu sebagai teks biasa, dengan alasan tertulis
 * "di layar lama pun tidak". Itu **salah**, dan terbantah dari export: sel `.pyID` pada grid
 * RCL/PUCL bertanda `pyControlDisplayTitle = Link` dan `pyUIElement = link`, dengan
 * `pyActivity = SetAssignmentInboxPUCL_act` berparameter `inskey = .pzInsKey`. Dilaporkan
 * Work Owner 2026-10-10 dan diperbaiki.
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

  // Layar tujuannya belum punya data untuk baris ini.
  //
  // Pada tab RCL/PUCL, daftar dibaca dari antrean Pega sedangkan layar kerja klaim dibaca
  // dari `POOLDATA.TC_PNC_PUCL` — dan keduanya tidak dijamin memuat klaim yang sama.
  // Sebelum penanda ini ada, barisnya tetap digambar sebagai tautan dan mendarat di
  // "klaim tidak ditemukan pada entitas yang sedang dipilih", yang mengarahkan petugas
  // memeriksa pilihan portal padahal portalnya benar.
  //
  // Sebabnya dinyatakan di `title`, bukan digambar penuh: pada antrean berisi puluhan
  // baris, keterangan di setiap sel akan menenggelamkan daftarnya.
  if (!item.layar_klaim_siap) {
    return (
      <span
        className="cursor-help text-slate-500"
        title={
          `Layar kerja klaim ${item.no_case} belum dapat dibuka: klaim ini ada di antrean ` +
          'Pega tetapi belum punya baris di tabel datar RCL/PUCL milik aplikasi. ' +
          'Kerjakan lewat Pega, dan laporkan ke tim teknis bila seharusnya sudah ada.'
        }
      >
        {item.no_case}
      </span>
    )
  }

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

    // Kedua tab nomor case-nya membuka layar kerja — hanya layarnya yang berbeda. Penandanya
    // datang dari server, bukan disimpulkan dari kode tab di sini.
    if (column.kunci === 'no_case' && (tab.buka_layar_kerja || tab.buka_layar_klaim)) {
      return { ...base, render: openCase }
    }
    return base
  })
}

/**
 * cellText menyusun teks satu sel.
 *
 * # Tanggal ditulis seperti GRID Pega: `dd/MM/yy HH:mm`
 *
 * Versi sebelumnya memformat HANYA bentuk `YYYY-MM-DD` telanjang, lewat `formatDate` yang
 * menulis "13 Juni 2025". Akibatnya dua hal sekaligus, dan keduanya terlihat berdampingan
 * di layar:
 *
 *   - kolom berisi waktu lengkap (`2025-06-13T14:24:21.212+07:00`) tidak cocok dengan
 *     penyaringnya, sehingga digambar MENTAH — persis yang dilaporkan Work Owner
 *     2026-10-10;
 *   - yang cocok pun ditulis dalam bentuk yang tidak dipakai Pega di mana pun.
 *
 * `formatPegaDateTime` menjawab keduanya: ia bentuk sel grid Pega apa adanya, dan ia
 * mengembalikan teks yang tidak dikenalinya APA ADANYA — sehingga kekhawatiran lama tentang
 * kolom yang bentuknya belum dapat diperiksa tanpa DDL (`R-08`) tetap terjaga. Nilai aneh
 * tetap terbaca dan dapat ditelusuri, bukan berubah menjadi teks yang salah.
 *
 * Teks kosong menjadi tanda pisah, bukan sel kosong yang tidak dapat dibedakan dari kolom
 * yang gagal dimuat. Di layar ini itu penting khusus: satu kolom memang SELALU kosong.
 */
function cellText(row: WorkItem, column: TabColumn): string {
  const value = row[column.kunci]

  if (value == null || value === '') return '—'

  return formatPegaDateTime(String(value))
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
