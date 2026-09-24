import { type ReactNode } from 'react'
import { useSearchParams } from 'react-router-dom'

import { APIError } from '@/api/client'
import { useSelectedPortal } from '@/app/portal'
import { Button } from '@/components/Button'
import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage } from '@/components/ErrorMessage'
import { formatDate } from '@/components/format'

import { ConversationDetail } from './ConversationDetail'
import { KomunikasiCabangTabs } from './KomunikasiCabangTabs'
import {
  useExportKomunikasiCabang,
  useKomunikasiCabangList,
  useKomunikasiCabangMetadata,
} from './api'
import type {
  BranchScope,
  Conversation,
  ExportColumn,
  Tab,
  TabColumn,
} from './types'

/**
 * Inbox Komunikasi Cabang — menu `MENU_ID 70`, pengganti harness `InboxKomunikasiCabang`.
 *
 * Isinya kotak percakapan antara KANTOR PUSAT dan CABANG seputar sebuah klaim, dipartisi
 * menurut sudah-belumnya dibalas:
 *
 *   Belum Dijawab   pesan belum dibalas sama sekali — urut dari yang PALING LAMA menunggu
 *   Sudah Dijawab   sudah dibalas — urut dari balasan TERBARU
 *
 * Per `D-79` ia benar-benar Inbox: barisnya adalah pekerjaan yang menunggu dijawab, dan
 * barisnya HILANG begitu percakapannya ditutup — tombol "Selesai Komunikasi" mengubah kanal
 * percakapannya, dan kedua grid menyaring kanal itu. Karena itu modul ini milik `U-3`, bukan
 * `U-6`.
 *
 * # Hal yang paling penting dipahami tentang layar ini
 *
 * Ia DISARING MENURUT CABANG pemanggilnya — berbeda dari modul inbox lain yang antreannya
 * bersama. Batas itu diturunkan dari login lewat HRD dan master pengguna asuransi, dan
 * petugas yang cabangnya tidak dapat diturunkan dilayani sebagai KANTOR PUSAT (`P-5`,
 * keputusan Work Owner 2026-09-24).
 *
 * Karena itu keterangan batas cabang di bawah judul bukan hiasan: ia satu-satunya hal di
 * layar yang menyatakan percakapan SIAPA yang sedang dilihat.
 */
export function KomunikasiCabangPage() {
  // Tab dan nomor halaman hidup di ALAMAT, bukan di state komponen.
  //
  // Alasannya sama dengan modul inbox lain: petugas yang membuka sebuah percakapan lalu
  // menyegarkan halaman harus mendarat di tempat yang sama, bukan di tab pertama halaman
  // pertama. Nomor percakapan yang sedang dibuka ikut, dengan alasan tambahan — alamatnya
  // dapat dibagikan ke rekan yang berhak melihatnya.
  const [params, setParams] = useSearchParams()

  const tabCode = params.get('tab') ?? ''
  const page = Math.max(1, Number(params.get('halaman') ?? '1') || 1)
  const opened = params.get('komunikasi') ?? ''

  const portal = useSelectedPortal((state) => state.alias)
  const meta = useKomunikasiCabangMetadata()

  const tabs: Tab[] = meta.data?.tab ?? []
  const active = tabCode || meta.data?.tab_bawaan || ''
  const tab = tabs.find((candidate) => candidate.kode === active)

  const list = useKomunikasiCabangList(active, page, meta.isSuccess)

  /**
   * Berpindah tab mengembalikan ke halaman pertama dan MENUTUP panel detail.
   *
   * Panel ditutup karena percakapan yang sedang dibuka belum tentu ada di tab tujuan —
   * membiarkannya terbuka akan menampilkan detail sebuah baris yang tidak terlihat di
   * tabel di atasnya, dan itu terbaca seperti kekeliruan.
   */
  function selectTab(code: string) {
    setParams({ tab: code, halaman: '1' }, { replace: true })
  }

  /** Berpindah halaman, menjaga tab yang sedang terbuka dan menutup panel detail. */
  function setPage(next: number) {
    setParams({ tab: active, halaman: String(next) }, { replace: true })
  }

  /** Membuka panel Detail Komunikasi untuk satu percakapan. */
  function openConversation(row: Conversation) {
    setParams(
      { tab: active, halaman: String(page), komunikasi: row.komunikasi },
      { replace: true },
    )
  }

  /** Menutup panel detail tanpa mengubah tab maupun halaman. */
  function closeConversation() {
    setParams({ tab: active, halaman: String(page) }, { replace: true })
  }

  if (portal === null) {
    return (
      <PageFrame tab={tab} exportable={false}>
        <ErrorMessage
          title="Pilih entitas lebih dulu"
          description={
            'Percakapan cabang milik satu badan hukum, dan aplikasi ini melayani empat. ' +
            'Pilih portal di bilah atas untuk membukanya.'
          }
          tone="gangguan"
        />
      </PageFrame>
    )
  }

  if (meta.isError) {
    return (
      <PageFrame tab={tab} exportable={false}>
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
      tab={tab}
      exportable={(list.data?.paginasi.total ?? 0) > 0}
      exportColumns={meta.data?.kolom_ekspor ?? []}
      branch={list.data?.batas_cabang}
    >
      <div className="mt-4">
        <KomunikasiCabangTabs
          tabs={tabs}
          active={active}
          summary={list.data?.ringkasan}
          onSelect={selectTab}
        />
      </div>

      {tab && (
        <>
          <p className="mt-3 text-sm text-slate-600">{tab.keterangan}</p>

          {tab.catatan && <TabNotice text={tab.catatan} />}

          <div className="mt-4">
            <DataTable<Conversation>
              columns={columnsFor(tab, (row) => (
                <MessageLink item={row} onOpen={openConversation} />
              ))}
              rows={list.data?.baris ?? []}
              rowKey={(row) => `${row.komunikasi}|${row.tanggal}`}
              title={tab.nama}
              label={`Percakapan ${tab.nama}`}
              // Kotak cari bawaan disembunyikan: hasilnya akan menyaring HANYA halaman yang
              // sedang terbuka, sehingga pengguna dapat diberi tahu "tidak ada" untuk baris
              // yang sebenarnya ada di halaman berikutnya.
              //
              // Layar lama pun tidak punya pencarian yang berfungsi: kotak "Filter"-nya
              // menyalin kedua isian ke variabel lokal lalu tidak pernah memakainya lagi.
              hideSearch
              isLoading={list.isPending}
              error={
                list.isError ? (
                  <ErrorMessage
                    title="Percakapan tidak dapat dimuat"
                    description={messageOf(list.error)}
                    tone="gangguan"
                  />
                ) : undefined
              }
              emptyMessage={emptyMessageFor(tab, list.data?.batas_cabang)}
              pagination={{
                page: list.data?.paginasi.halaman ?? 1,
                size: list.data?.paginasi.ukuran ?? 20,
                total: list.data?.paginasi.total ?? 0,
                totalPage: list.data?.paginasi.total_halaman ?? 1,
                onPageChange: setPage,
                isLoading: list.isFetching,
              }}
            />
          </div>
        </>
      )}

      {opened !== '' && (
        <ConversationDetail id={opened} onClose={closeConversation} />
      )}

      <PlannedDifferences lines={meta.data?.selisih_terencana ?? []} />
    </PageFrame>
  )
}

function PageFrame({
  tab,
  exportable,
  exportColumns = [],
  branch,
  children,
}: {
  tab: Tab | undefined
  /** Tombol unduh hanya berguna bila ada yang dapat diunduh. */
  exportable: boolean
  exportColumns?: ExportColumn[]
  branch?: BranchScope | undefined
  children: ReactNode
}) {
  return (
    <div className="mx-auto max-w-[96rem] px-4 py-8">
      <header className="flex flex-wrap items-start justify-between gap-4 border-b border-slate-200 pb-4">
        <div>
          <h1 className="text-xl font-semibold text-slate-900">
            Inbox Komunikasi Cabang
          </h1>
          <p className="mt-1 text-sm text-slate-600">
            Percakapan antara kantor pusat dan cabang seputar klaim yang sedang berjalan.
          </p>
          {branch && <BranchNotice branch={branch} />}
        </div>
        <ExportButton tab={tab} enabled={exportable} columns={exportColumns} />
      </header>
      {children}
      <WriteActionsNotice />
    </div>
  )
}

/**
 * Keterangan batas data — percakapan SIAPA yang sedang dilihat.
 *
 * # Kenapa ia ada, dan kenapa nadanya berubah
 *
 * Karena layar ini disaring menurut cabang, dan petugas yang tidak tahu daftarnya sedang
 * disaring akan menyimpulkan tidak ada percakapan — padahal yang benar adalah tidak ada
 * percakapan DI CABANGNYA. Pelajaran itu sudah tercatat pada modul Inbox Laporan Klaim.
 *
 * Nadanya berubah menjadi PERINGATAN ketika kode cabang pemanggil tidak dapat dibaca. Dalam
 * keadaan itu ia sedang melihat percakapan kantor pusat — bukan percakapan cabangnya — dan
 * itu pelebaran batas data yang tidak menghasilkan satu pun galat. Kalimatnya sendiri datang
 * dari server, supaya penjelasannya berubah di satu tempat saat keputusannya ditinjau ulang.
 */
function BranchNotice({ branch }: { branch: BranchScope }) {
  const warning = !branch.terbaca

  return (
    <p
      className={[
        'mt-2 rounded-kontrol px-2 py-1 text-xs',
        warning ? 'bg-amber-50 text-amber-900' : 'text-slate-500',
      ].join(' ')}
      role={warning ? 'alert' : undefined}
    >
      {branch.keterangan}
    </p>
  )
}

/**
 * Catatan yang berlaku pada satu tab saja.
 *
 * Isinya datang dari SERVER, bukan ditulis tetap di sini, supaya ia hilang dengan sendirinya
 * begitu keadaannya berubah.
 */
function TabNotice({ text }: { text: string }) {
  return (
    <div className="mt-3 rounded-kartu border border-sky-200 bg-sky-50 px-4 py-3">
      <p className="text-xs text-slate-700">{text}</p>
    </div>
  )
}

/**
 * Tombol unduh.
 *
 * # SELURUH tombol ini adalah KEMAMPUAN BARU
 *
 * Layar lama TIDAK punya tombol ekspor sama sekali. Ia ditambahkan karena percakapan yang
 * menumpuk tidak dapat ditelusuri lewat layar berhalaman, dan kemampuan barunya dinyatakan
 * lewat `selisih_terencana` — bukan disamarkan sebagai paritas.
 *
 * Kolom berkasnya disebutkan di bawah tombol karena ia memuat TIGA kolom yang tidak ada di
 * tabel mana pun: tujuan pesan, tanggal jawaban, dan status register.
 */
function ExportButton({
  tab,
  enabled,
  columns,
}: {
  tab: Tab | undefined
  enabled: boolean
  columns: ExportColumn[]
}) {
  const ekspor = useExportKomunikasiCabang()

  return (
    <div className="flex max-w-sm flex-col items-end gap-1">
      <Button
        tone="kedua"
        disabled={!enabled || ekspor.isPending || tab === undefined}
        onClick={() => ekspor.mutate(tab?.kode ?? '')}
      >
        {ekspor.isPending ? 'Menyiapkan berkas…' : 'Export To Excel'}
      </Button>

      {columns.length > 0 && (
        <p className="text-right text-xs text-slate-500">
          Berisi: {columns.map((column) => column.judul).join(' · ')}
        </p>
      )}

      {ekspor.isError && (
        <p className="text-right text-xs text-red-700" role="alert">
          {messageOf(ekspor.error)}
        </p>
      )}
    </div>
  )
}

/**
 * Keterangan keempat tindakan yang ada di layar lama tetapi belum tersedia di sini.
 *
 * # Kenapa dinyatakan, bukan dibiarkan hilang begitu saja
 *
 * Karena keempatnya adalah pekerjaan NYATA yang dilakukan pengguna layar ini setiap hari.
 * Layar yang kehilangan tombolnya tanpa penjelasan akan dilaporkan sebagai kerusakan, dan
 * penggunanya tidak akan tahu ia masih harus mengerjakannya lewat Pega.
 *
 * Satu di antaranya bahkan tidak dapat dibangun sekalipun diputuskan: activity di balik
 * tombol "Balas" tidak ada di export mana pun.
 */
function WriteActionsNotice() {
  return (
    <section className="mt-6 rounded-kartu border border-amber-200 bg-amber-50 px-4 py-3">
      <h2 className="text-sm font-medium text-amber-900">
        Yang masih dikerjakan lewat Pega
      </h2>
      <p className="mt-2 text-xs text-slate-700">
        <span className="font-medium">Kirim Pesan</span>,{' '}
        <span className="font-medium">Balas</span>,{' '}
        <span className="font-medium">Selesai Komunikasi</span>, dan{' '}
        <span className="font-medium">Tambah</span> belum tersedia di sini. Keempatnya
        mengubah isi percakapan — menutup percakapan bahkan menghilangkannya dari kedua tab
        — dan selama Pega dan sistem baru berjalan berdampingan, data itu hanya boleh diubah
        dari satu sistem. Layar ini untuk membaca dan mengunduh; tindakannya kerjakan di
        Pega.
      </p>
    </section>
  )
}

/**
 * Tautan pada kolom "Pesan".
 *
 * # Kenapa pada sel Pesan, bukan kolom tombol tersendiri
 *
 * Karena layar lama TIDAK punya kolom tombol di dalam grid — tombol "Detail Komunikasi"
 * berada di bilah aksi, dan yang menentukan percakapan mana yang dibuka adalah baris yang
 * sedang dipilih.
 *
 * Menaruh tautannya pada sel "Pesan" mempertahankan jumlah kolom apa adanya (`D-13`)
 * sekaligus memberi sasaran klik yang jelas per baris. Kolom "Pesan" dipilih karena ia satu-
 * satunya kolom yang ada di KEDUA tab dan selalu terisi — penyaring kedua grid menuntut
 * pesannya tidak kosong.
 */
function MessageLink({
  item,
  onOpen,
}: {
  item: Conversation
  onOpen: (row: Conversation) => void
}) {
  if (item.pesan === '') return <span className="text-slate-400">—</span>

  return (
    <button
      type="button"
      onClick={() => onOpen(item)}
      title={`Buka percakapan ${item.komunikasi}`}
      className={[
        'rounded-kontrol text-left font-medium text-blue-700 underline-offset-2',
        'transition-colors duration-150 ease-halus hover:underline',
        'focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500/50',
      ].join(' ')}
    >
      {item.pesan}
    </button>
  )
}

/**
 * Selisih terhadap Pega yang sudah diputuskan, ditampilkan di bawah tabel.
 *
 * Isinya datang dari SERVER, bukan ditulis tetap di sini. Tanpa catatan ini, beberapa hal
 * akan dilaporkan berulang kali sebagai kerusakan oleh orang yang membandingkan kedua layar
 * berdampingan — terutama pemisahan menjadi tab, urutan kedua tab yang berlawanan, dan
 * angka pencacah yang tidak sama dengan jumlah baris tabel.
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
 * `value` tetap mengembalikan TEKS polos meski selnya digambar sebagai tautan. Keduanya
 * dipisah dengan sengaja oleh `DataTable`: yang dicari dan diurutkan adalah teksnya, yang
 * dilihat pengguna adalah gambarnya.
 */
function columnsFor(
  tab: Tab,
  openConversation: (row: Conversation) => ReactNode,
): Column<Conversation>[] {
  return tab.kolom.map((column) => {
    const base: Column<Conversation> = {
      key: column.kunci,
      title: column.judul,
      value: (row) => cellText(row, column),
    }

    if (column.kunci === 'pesan') {
      return { ...base, render: openConversation }
    }
    return base
  })
}

/**
 * cellText menyusun teks satu sel.
 *
 * Dua perlakuan, dan masing-masing punya alasannya:
 *
 *  - Tanggal diformat HANYA bila bentuknya memang `YYYY-MM-DD`. Kolom tanggal di layar ini
 *    dibaca dari kolom yang bentuknya tidak dapat diperiksa tanpa DDL (`R-08`). Memaksa
 *    pemformatan akan mengubah nilai yang tidak dikenali menjadi teks yang salah.
 *  - Teks kosong menjadi tanda pisah, bukan sel kosong yang tidak dapat dibedakan dari kolom
 *    yang gagal dimuat.
 */
function cellText(row: Conversation, column: TabColumn): string {
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
 * emptyMessageFor menjelaskan kotak percakapan kosong menurut sebabnya.
 *
 * "Tidak ada data" tidak cukup di layar ini, dan alasannya khas: daftarnya DISARING menurut
 * cabang, sehingga kosong dapat berarti "tidak ada pekerjaan" MAUPUN "batas cabang Anda
 * bukan yang Anda kira". Hanya yang kedua yang menuntut laporan.
 */
function emptyMessageFor(tab: Tab, branch: BranchScope | undefined): string {
  const scope =
    branch === undefined
      ? ''
      : branch.kantor_pusat
        ? ' pada percakapan kantor pusat'
        : ` pada cabang ${branch.kode}`

  const unresolved =
    branch !== undefined && !branch.terbaca
      ? ' Perhatikan kode cabang Anda tidak terbaca, sehingga yang ditampilkan adalah ' +
        'percakapan kantor pusat — bukan cabang Anda. Bila Anda petugas cabang, laporkan.'
      : ''

  const showsReply = tab.kolom.some((column) => column.kunci === 'jawaban_terakhir')

  if (showsReply) {
    return (
      `Belum ada percakapan yang sudah dibalas${scope}. Lihat tab "Belum Dijawab" — ` +
      `pesan yang menunggu jawaban ada di sana.${unresolved}`
    )
  }

  return (
    `Tidak ada pesan yang menunggu dijawab${scope}. Percakapan yang sudah dibalas ada di ` +
    `tab "Sudah Dijawab", dan yang sudah ditutup tidak lagi tampil di layar ini.${unresolved}`
  )
}

/** messageOf mengambil pesan yang layak dibaca pengguna dari sebuah galat. */
function messageOf(error: unknown): string {
  if (error instanceof APIError) return error.message
  if (error instanceof Error && error.message !== '') return error.message
  return 'Coba lagi beberapa saat lagi. Bila terus berulang, hubungi tim teknis.'
}
