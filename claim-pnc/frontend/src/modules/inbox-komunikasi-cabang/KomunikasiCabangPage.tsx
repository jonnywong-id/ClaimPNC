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
  useKomunikasiCabangAction,
  useKomunikasiCabangList,
  useKomunikasiCabangMetadata,
} from './api'
import { isActionField } from './types'
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

  // Aksi tulis dikumpulkan dalam SATU mutation, bukan satu per tombol.
  //
  // Keempatnya dijawab peladen dengan alasan yang sama, dan yang membedakannya hanyalah nama
  // tindakan yang ikut dicatat. Memisahkannya menjadi empat hook akan menggandakan
  // penanganan galat yang identik.
  const action = useKomunikasiCabangAction()

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
      onRefresh={() => void list.refetch()}
      // `isRefetching`, BUKAN `isFetching`: yang kedua juga bernilai benar selama pemuatan
      // PERTAMA, sehingga tombolnya berbunyi "Menyegarkan…" sebelum seorang pun menekannya —
      // menyatakan sesuatu sedang dikerjakan atas perintah pengguna, padahal tidak.
      refreshing={list.isRefetching}
      onAdd={() => action.mutate({ tindakan: 'tambah' })}
      adding={action.isPending}
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

          {/*
            Jawaban aksi tulis digambar DI ATAS tabel, bukan di dekat tombolnya.

            Tombolnya ada di dalam baris, dan baris dapat berada jauh di bawah layar yang
            terlihat. Pesan yang muncul di sebelahnya akan terlewat justru oleh orang yang
            menekannya.
          */}
          {action.isError && <ActionNotice message={messageOf(action.error)} />}

          <div className="mt-4">
            <DataTable<Conversation>
              columns={columnsFor(tab, {
                onOpen: openConversation,
                onFinish: (row) =>
                  action.mutate({
                    tindakan: 'selesai-komunikasi',
                    komunikasi: row.komunikasi,
                  }),
                finishing: action.isPending,
              })}
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
  onRefresh,
  refreshing,
  onAdd,
  adding,
  children,
}: {
  tab: Tab | undefined
  /** Tombol unduh hanya berguna bila ada yang dapat diunduh. */
  exportable: boolean
  exportColumns?: ExportColumn[]
  branch?: BranchScope | undefined

  /**
   * Kedua tombol bilah atas dibiarkan OPSIONAL supaya kerangka ini tetap dapat dipakai
   * pada keadaan galat, tempat keduanya memang tidak berguna — layar yang tidak dapat
   * memuat daftarnya tidak punya apa pun untuk disegarkan.
   */
  onRefresh?: () => void
  refreshing?: boolean
  onAdd?: () => void
  adding?: boolean

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
        <div className="flex flex-col items-end gap-2">
          <div className="flex flex-wrap items-center justify-end gap-2">
            {onRefresh && <RefreshButton onRefresh={onRefresh} busy={refreshing === true} />}
            {onAdd && <AddButton onAdd={onAdd} busy={adding === true} />}
          </div>
          <ExportButton tab={tab} enabled={exportable} columns={exportColumns} />
        </div>
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
 * Jawaban peladen atas aksi tulis yang belum tersedia.
 *
 * Kalimatnya datang dari PELADEN, bukan ditulis di sini: ia menyebut mengapa tindakannya
 * belum ada dan ke mana pengguna harus pergi, dan menuliskannya di dua tempat berarti yang
 * di layar akan tetap berbunyi "belum tersedia" lama setelah tindakannya tersedia.
 *
 * `role="alert"` supaya pembaca layar mengumumkannya — tombolnya ditekan, dan jawabannya
 * muncul di tempat lain di halaman.
 */
function ActionNotice({ message }: { message: string }) {
  return (
    <div
      role="alert"
      className="mt-3 rounded-kartu border border-amber-200 bg-amber-50 px-4 py-3"
    >
      <p className="text-xs text-slate-700">{message}</p>
    </div>
  )
}

/**
 * Tombol **"Refresh"**.
 *
 * # Satu-satunya tombol layar lama yang benar-benar BEKERJA di sini
 *
 * Ia ada di kedua grid layar lama (`pyAction = refresh`), dan ia **tidak menulis apa pun** —
 * sehingga tidak ada satu pun alasan menahannya (`P-1` hanya menyangkut penulisan).
 *
 * Ia lebih berguna di sini daripada di Pega: kotak percakapan berubah saat petugas lain
 * membalas, dan cache layar ini berumur 15 detik. Tanpa tombol ini, satu-satunya cara
 * memaksa pembacaan ulang adalah menyegarkan seluruh halaman — yang ikut membuang tab dan
 * nomor halaman yang sedang dibuka.
 */
function RefreshButton({ onRefresh, busy }: { onRefresh: () => void; busy: boolean }) {
  return (
    <Button tone="kedua" disabled={busy} onClick={onRefresh}>
      {busy ? 'Menyegarkan…' : 'Refresh'}
    </Button>
  )
}

/**
 * Tombol **"Tambah"** — membuka percakapan baru.
 *
 * # Kenapa tombolnya ada tetapi formnya tidak
 *
 * Karena form yang dapat diisi tetapi menolak saat dikirim lebih buruk daripada tidak ada:
 * pengguna sudah mengetik pesannya, dan pesan itu hilang. Alasan yang sama dipakai kotak
 * balasan pada layar detail.
 *
 * Tombolnya tetap digambar karena ia ADA di layar lama, dan tombol yang hilang tanpa
 * penjelasan dilaporkan sebagai kerusakan. Penekanannya menjawab alasan dari peladen.
 *
 * "Kirim Pesan" — tombol kirim di dalam form itu — karena itu belum punya tempat di sini.
 * Ia akan lahir bersama formnya, bila kepemilikan tabelnya kelak berpindah (`P-1`).
 */
function AddButton({ onAdd, busy }: { onAdd: () => void; busy: boolean }) {
  return (
    <Button tone="kedua" disabled={busy} onClick={onAdd}>
      Tambah
    </Button>
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
 * Tombol **"Detail Komunikasi"** — satu per baris.
 *
 * # Ia kolom sungguhan di layar lama, bukan tambahan
 *
 * Kedua grid menggambarnya sebagai kolom tersendiri berjudul "Button", dan aksinya terbaca
 * dari section apa adanya: `runDataTransform` atas `DetailKomunikasi_dt` dengan parameter
 * `KOMID = .ClaimNo`, lalu `localAction` `DETAILKOMUNIKASICABANG_11`.
 *
 * Versi pertama modul ini melewatkannya dan menggantinya dengan tautan pada sel "Pesan" —
 * yang di Pega tidak ada sama sekali. Itu bukan sekadar beda tampilan: sel yang menjadi
 * tautan mengubah cara seluruh kolom terbaca, dan tombol yang hilang membuat pengguna
 * mencarinya di bilah aksi yang memang tidak memilikinya.
 */
function DetailButton({
  item,
  onOpen,
}: {
  item: Conversation
  onOpen: (row: Conversation) => void
}) {
  return (
    <button
      type="button"
      onClick={() => onOpen(item)}
      // Nama lengkapnya dibaca pembaca layar, sementara yang terlihat tetap pendek supaya
      // kolomnya tidak melebar. Kedua kolom tombol berjudul "Button" yang sama, sehingga
      // tanpa nama ini keduanya tidak dapat dibedakan tanpa melihat.
      aria-label={`Detail Komunikasi percakapan ${item.komunikasi}`}
      title={`Buka utas percakapan ${item.komunikasi}`}
      className={[
        'rounded-kontrol border border-slate-300 px-2.5 py-1 text-xs font-medium',
        'whitespace-nowrap text-slate-700',
        'transition-colors duration-150 ease-halus hover:bg-slate-100',
        'focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500/50',
      ].join(' ')}
    >
      Detail Komunikasi
    </button>
  )
}

/**
 * Tombol **"Selesai Komunikasi"** — satu per baris.
 *
 * # Apa yang dilakukannya di Pega, dan kenapa itu penting diketahui
 *
 * `runActivity` atas `EndKomunikasiCabang` dengan `KOMID = .ClaimNo`, lalu me-refresh grid.
 * Activity itu mengubah `CASEID` menjadi `CABANG SELESAI` — sehingga barisnya **hilang dari
 * kedua tab**. Ia bukan penanda yang dapat dibatalkan; ia mengeluarkan percakapan dari layar.
 *
 * # Kenapa tombolnya tetap digambar meski pasti ditolak
 *
 * Karena ia ADA di layar lama, dan layar yang kehilangan tombolnya tanpa penjelasan
 * dilaporkan sebagai kerusakan. Penekanannya menjawab alasan dari peladen (`501`) — bukan
 * diam, dan bukan halaman kosong.
 *
 * Nadanya sengaja **tidak** merah: ia bukan tombol berbahaya yang sedang dicegah, melainkan
 * tombol yang belum tersedia.
 */
function FinishButton({
  item,
  onFinish,
  busy,
}: {
  item: Conversation
  onFinish: (row: Conversation) => void
  busy: boolean
}) {
  return (
    <button
      type="button"
      disabled={busy}
      onClick={() => onFinish(item)}
      aria-label={`Selesai Komunikasi percakapan ${item.komunikasi}`}
      title={`Tutup percakapan ${item.komunikasi}`}
      className={[
        'rounded-kontrol border border-slate-300 px-2.5 py-1 text-xs font-medium',
        'whitespace-nowrap text-slate-700',
        'transition-colors duration-150 ease-halus hover:bg-slate-100',
        'focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500/50',
        'disabled:cursor-not-allowed disabled:opacity-55',
      ].join(' ')}
    >
      Selesai Komunikasi
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
  actions: {
    onOpen: (row: Conversation) => void
    onFinish: (row: Conversation) => void
    finishing: boolean
  },
): Column<Conversation>[] {
  return tab.kolom.map((column) => {
    // Kolom TOMBOL tidak punya isian pada baris. `value` dibiarkan kosong dengan sengaja —
    // itulah yang dicari dan diurutkan `DataTable`, dan mengurutkan menurut markup tombol
    // tidak berarti apa pun.
    if (isActionField(column.kunci)) {
      const render =
        column.kunci === 'aksi_detail'
          ? (row: Conversation) => <DetailButton item={row} onOpen={actions.onOpen} />
          : (row: Conversation) => (
              <FinishButton
                item={row}
                onFinish={actions.onFinish}
                busy={actions.finishing}
              />
            )

      return {
        key: column.kunci,
        title: column.judul,
        value: () => '',
        noSort: true,
        render,
      }
    }

    return {
      key: column.kunci,
      title: column.judul,
      value: (row) => cellText(row, column),
    }
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
  // Kolom tombol tidak pernah sampai ke sini; columnsFor menanganinya lebih dulu.
  if (isActionField(column.kunci)) return ''

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
