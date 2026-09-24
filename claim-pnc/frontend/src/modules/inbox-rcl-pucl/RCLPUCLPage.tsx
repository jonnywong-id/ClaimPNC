import { useState, type ReactNode } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'

import { APIError } from '@/api/client'
import { useSelectedPortal } from '@/app/portal'
import { Button } from '@/components/Button'
import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage } from '@/components/ErrorMessage'
import { FormField } from '@/components/FormField'
import { formatDate } from '@/components/format'

import { RCLPUCLTabs } from './RCLPUCLTabs'
import { useExportRCLPUCL, useRCLPUCLList, useRCLPUCLMetadata } from './api'
import type { DateRange, ReportColumn, Tab, TabColumn, WorkItem } from './types'

/**
 * Inbox RCL/PUCL — menu `MENU_ID 61`, pengganti harness `RCLPUCL_Harness`.
 *
 * Isinya antrean klaim yang **ditolak (RCL)** atau **diproses ulang (PUCL)**, dipartisi
 * menurut perjalanan surat PUCL-nya:
 *
 *   Cetak Surat          surat BELUM dicetak — menunggu tindakan paling awal
 *   Kelengkapan Dokumen  surat sudah dicetak, menunggu kelengkapan dari tertanggung
 *   Klaim MSIG           sama, tetapi jalur MSIG
 *
 * Per `D-79` ia benar-benar Inbox: barisnya diambil dari tabel penugasan Pega dan hilang
 * begitu penugasannya selesai. Karena itu modul ini milik `U-3`, bukan `U-6`.
 *
 * # Susunan layar, dan dari mana bentuknya
 *
 * Diambil dari ketiga section-nya: bilah tab, dua isian tanggal pada tab pertama, lalu
 * grid berhalaman 50 baris. Judul kolom TIDAK diterjemahkan — `D-13` menetapkan tampilan
 * meniru Pega, dan itulah teks yang selama ini dibaca pengguna.
 *
 * # Hal yang paling mudah disalahpahami di layar ini
 *
 * KETIGA tab punya kolom yang IDENTIK. Tidak ada satu pun petunjuk visual yang membedakan
 * tab satu dari yang lain selain judulnya — sehingga keterangan tab dan catatan per tab
 * bukan hiasan di sini, melainkan satu-satunya hal yang menjelaskan mengapa sebuah klaim
 * ada di tab ini dan tidak di tab sebelah.
 */
export function RCLPUCLPage() {
  // Tab dan nomor halaman hidup di ALAMAT, bukan di state komponen.
  //
  // Alasannya satu dan nyata: mengklik Nomor Case membuka layar kerja di rute lain, dan
  // komponen ini dilepas. State yang hanya ada di memori akan hilang, sehingga petugas yang
  // kembali dari sebuah klaim mendarat di tab pertama halaman pertama — padahal ia sedang
  // mengerjakan halaman ketiga tab kedua. Pada antrean yang dibuka berpuluh kali sehari,
  // itu bukan ketidaknyamanan kecil.
  //
  // Rentang tanggal TIDAK ikut ke alamat: ia hanya dipakai tombol ekspor, tidak menyaring
  // tabel, dan menaruhnya di alamat akan menyiratkan ia bagian dari apa yang sedang dilihat.
  const [params, setParams] = useSearchParams()
  const [range, setRange] = useState<DateRange>({ dari: '', sampai: '' })

  const navigate = useNavigate()

  const tabCode = params.get('tab') ?? ''
  const page = Math.max(1, Number(params.get('halaman') ?? '1') || 1)

  const portal = useSelectedPortal((state) => state.alias)
  const meta = useRCLPUCLMetadata()

  const tabs: Tab[] = meta.data?.tab ?? []
  const active = tabCode || meta.data?.tab_bawaan || ''
  const tab = tabs.find((candidate) => candidate.kode === active)

  // Tab terhalang TIDAK meminta isinya. Permintaannya pasti ditolak server dengan 422, dan
  // galat yang sudah diketahui pasti terjadi bukan galat yang layak ditampilkan — yang
  // layak ditampilkan adalah alasan terhalangnya.
  const blocked = tab?.terhalang === true
  const list = useRCLPUCLList(active, page, meta.isSuccess && !blocked)

  /**
   * Berpindah tab mengembalikan ke halaman pertama.
   *
   * Tanpa itu, berpindah dari halaman 4 sebuah tab ke tab yang hanya punya 2 halaman akan
   * menampilkan tabel kosong yang terbaca seperti antrean yang memang kosong — dan pada
   * layar yang ketiga tabnya tampak sama, itu sangat mudah disalahartikan.
   */
  function selectTab(code: string) {
    setParams({ tab: code, halaman: '1' }, { replace: true })
  }

  /** Berpindah halaman, menjaga tab yang sedang terbuka. */
  function setPage(next: number) {
    setParams({ tab: active, halaman: String(next) }, { replace: true })
  }

  /**
   * Membuka layar kerja klaim — yang di Pega dijalankan Open Assignment.
   *
   * Tab dan halaman ikut ke alamat tujuan supaya tombol kembali mendarat di tempat yang
   * sama. Kuncinya dikodekan: `pzInsKey` memuat SPASI (`ASM-FW-GCNMFW-WORK PNC-1234`), dan
   * spasi mentah di dalam alamat bukan alamat yang sah.
   */
  function openCase(row: WorkItem) {
    const back = new URLSearchParams({ tab: active, halaman: String(page) })
    navigate(`/inbox-rcl-pucl/klaim/${encodeURIComponent(row.referensi)}?${back}`)
  }

  if (portal === null) {
    return (
      <PageFrame tab={tab} range={range} exportable={false}>
        <ErrorMessage
          title="Pilih entitas lebih dulu"
          description={
            'Antrean RCL/PUCL milik satu badan hukum, dan aplikasi ini melayani empat. ' +
            'Pilih portal di bilah atas untuk membukanya.'
          }
          tone="gangguan"
        />
      </PageFrame>
    )
  }

  if (meta.isError) {
    return (
      <PageFrame tab={tab} range={range} exportable={false}>
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
      range={range}
      // Laporan harian dapat diunduh meski tabelnya kosong: isinya memang bukan isi tabel,
      // dan rentang tanggalnya dapat memuat baris yang tidak satu pun tab tampilkan.
      exportable={
        !blocked &&
        (tab?.punya_laporan_rentang_tanggal === true ||
          (list.data?.paginasi.total ?? 0) > 0)
      }
      reportColumns={meta.data?.kolom_laporan ?? []}
    >
      <div className="mt-4">
        <RCLPUCLTabs tabs={tabs} active={active} onSelect={selectTab} />
      </div>

      {tab && (
        <>
          <p className="mt-3 text-sm text-slate-600">{tab.keterangan}</p>

          {tab.catatan && <TabNotice text={tab.catatan} />}

          {tab.punya_laporan_rentang_tanggal && (
            <DateRangeFilter value={range} onChange={setRange} />
          )}

          {tab.terhalang ? (
            <BlockedNotice tab={tab} />
          ) : (
            <div className="mt-4">
              <DataTable<WorkItem>
                columns={columnsFor(tab, (row) => (
                  <CaseLink item={row} onOpen={openCase} />
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
  range,
  exportable,
  reportColumns = [],
  children,
}: {
  tab: Tab | undefined
  range: DateRange
  /** Tombol ekspor hanya berguna bila ada yang dapat diekspor. */
  exportable: boolean
  reportColumns?: ReportColumn[]
  children: ReactNode
}) {
  return (
    <div className="mx-auto max-w-[96rem] px-4 py-8">
      <header className="flex flex-wrap items-start justify-between gap-4 border-b border-slate-200 pb-4">
        <div>
          <h1 className="text-xl font-semibold text-slate-900">Inbox RCL/PUCL</h1>
          <p className="mt-1 text-sm text-slate-600">
            Klaim yang ditolak (RCL) atau diproses ulang (PUCL), dikelompokkan menurut
            perjalanan surat PUCL-nya.
          </p>
          {/*
            Sifat "antrean bersama" dinyatakan di layar, bukan hanya di kode.

            Tidak satu pun tab di sini menyaring menurut pengguna yang login — penyaringnya
            akun antrean, bukan orang. Tanpa keterangan ini, petugas yang terbiasa dengan
            layar inbox lain akan mengira daftarnya keliru karena memuat pekerjaan orang
            lain.
          */}
          <p className="mt-2 text-xs text-slate-500">
            Ini <span className="font-medium">antrean bersama</span>: seluruh petugas
            RCL/PUCL pada portal yang sedang dipilih melihat daftar yang sama, bukan hanya
            pekerjaan Anda. Setiap pembukaannya tercatat.
          </p>
        </div>
        <ExportButton tab={tab} range={range} enabled={exportable} columns={reportColumns} />
      </header>
      {children}
      <WriteActionsNotice />
    </div>
  )
}

/**
 * Kedua isian tanggal tab "Cetak Surat".
 *
 * # Kenapa peringatannya sekeras ini
 *
 * Karena isian ini TIDAK menyaring tabel di bawahnya — hanya berkas ekspornya. Itu
 * perilaku layar lama: grid-nya dipasok Report Definition yang tidak menyaring tanggal
 * sama sekali, sementara tombol ekspornya menjalankan kueri yang berbeda. Work Owner
 * memutuskan mereplikasinya (`P-5`).
 *
 * Tanpa peringatan, pengguna akan mengisi tanggal, melihat tabel tidak berubah, lalu
 * melaporkannya sebagai kerusakan. Judulnya sendiri mengikuti layar lama apa adanya —
 * "FROM RCL/PUCL" dan "TO RCL/PUCL" (`D-13`).
 */
function DateRangeFilter({
  value,
  onChange,
}: {
  value: DateRange
  onChange: (next: DateRange) => void
}) {
  return (
    <section className="mt-4 rounded-kartu border border-slate-200 bg-slate-50 px-4 py-3">
      <div className="flex flex-wrap items-start gap-4">
        <FormField
          id="rcl-pucl-dari"
          label="FROM RCL/PUCL"
          type="date"
          value={value.dari}
          onChange={(event) => onChange({ ...value, dari: event.target.value })}
        />
        <FormField
          id="rcl-pucl-sampai"
          label="TO RCL/PUCL"
          type="date"
          value={value.sampai}
          onChange={(event) => onChange({ ...value, sampai: event.target.value })}
        />
      </div>
      <p className="mt-3 text-xs text-slate-600">
        <span className="font-medium">Kedua tanggal ini hanya dipakai tombol unduh.</span>{' '}
        Tabel di bawah tidak ikut tersaring — sama seperti di layar lama. Isinya pun
        berbeda: berkas laporan memuat klaim yang suratnya sudah dicetak, klaim yang sudah
        selesai, dan klaim Personal Accident di luar antrean ini.
      </p>
    </section>
  )
}

/**
 * Catatan yang berlaku pada satu tab saja.
 *
 * Isinya datang dari SERVER, bukan ditulis tetap di sini, supaya ia hilang dengan
 * sendirinya begitu keadaannya berubah — khususnya catatan tab "Klaim MSIG", yang berlaku
 * hanya sampai DBA memastikan kolom penandanya.
 */
function TabNotice({ text }: { text: string }) {
  return (
    <div className="mt-3 rounded-kartu border border-sky-200 bg-sky-50 px-4 py-3">
      <p className="text-xs text-slate-700">{text}</p>
    </div>
  )
}

/**
 * Tombol ekspor.
 *
 * # SATU tombol, DUA isi berkas
 *
 * Tab "Cetak Surat" mengunduh LAPORAN HARIAN berbasis rentang tanggal; dua tab lain
 * mengunduh salinan tabelnya. Itu perilaku layar lama — tombolnya satu dan sama di ketiga
 * tab, hanya activity di baliknya yang berbeda.
 *
 * Labelnya ikut berbeda supaya pengguna tahu apa yang akan diunduhnya SEBELUM menekan,
 * dan kolom berkasnya disebutkan di bawah tombol untuk alasan yang sama.
 */
function ExportButton({
  tab,
  range,
  enabled,
  columns,
}: {
  tab: Tab | undefined
  range: DateRange
  enabled: boolean
  columns: ReportColumn[]
}) {
  const ekspor = useExportRCLPUCL()
  const isReport = tab?.punya_laporan_rentang_tanggal === true

  return (
    <div className="flex max-w-sm flex-col items-end gap-1">
      <Button
        tone="kedua"
        disabled={!enabled || ekspor.isPending || tab === undefined}
        onClick={() =>
          ekspor.mutate({
            tab: tab?.kode ?? '',
            range: isReport ? range : undefined,
          })
        }
      >
        {ekspor.isPending
          ? 'Menyiapkan berkas…'
          : isReport
            ? 'Unduh Laporan Harian'
            : 'Export To Excel'}
      </Button>

      {isReport && columns.length > 0 && (
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
 * Keterangan kedua tindakan yang ada di layar lama tetapi belum tersedia di sini.
 *
 * # Kenapa dinyatakan, bukan dibiarkan hilang begitu saja
 *
 * Karena keduanya adalah pekerjaan NYATA yang dilakukan pengguna layar ini setiap hari —
 * mencetak surat PUCL/RCL dan mengirim pengingat. Layar yang kehilangan tombolnya tanpa
 * penjelasan akan dilaporkan sebagai kerusakan, dan penggunanya tidak akan tahu ia masih
 * harus mengerjakannya lewat Pega.
 *
 * Keduanya MENULIS ke objek kerja klaim, dan selama masa paralel tabel itu hanya boleh
 * ditulis satu sistem (`P-1`). Mencetak surat bahkan memindahkan klaimnya antartab.
 */
function WriteActionsNotice() {
  return (
    <section className="mt-6 rounded-kartu border border-amber-200 bg-amber-50 px-4 py-3">
      <h2 className="text-sm font-medium text-amber-900">
        Yang masih dikerjakan lewat Pega
      </h2>
      <p className="mt-2 text-xs text-slate-700">
        <span className="font-medium">Cetak Surat</span> dan{' '}
        <span className="font-medium">Reminder PUCL</span> belum tersedia di sini. Keduanya
        mengubah data klaim — mencetak surat bahkan memindahkan klaimnya dari tab
        &ldquo;Cetak Surat&rdquo; ke &ldquo;Kelengkapan Dokumen&rdquo; — dan selama Pega
        dan sistem baru berjalan berdampingan, data klaim hanya boleh diubah dari satu
        sistem. Layar ini untuk memantau dan mengunduh; tindakannya kerjakan di Pega.
      </p>
    </section>
  )
}

/**
 * Tautan pada kolom "Nomor Case".
 *
 * # Kenapa TAUTAN pada nomor case, bukan tombol di ujung baris
 *
 * Karena begitulah layar lama. Ketiga section RCL/PUCL menggambar sel "Nomor Case" sebagai
 * `pyUIElement = link` ber-`pyLabel = .pyID`, dan `D-13` menetapkan tampilan mengikuti Pega.
 *
 * Versi pertama modul ini keliru di sini: ia menambahkan kolom tombol "Lihat Detail" yang
 * **tidak ada di Pega sama sekali**. Itu bukan sekadar beda tampilan — kolom yang tidak
 * pernah ada membuat pengguna mengira ada dua cara berbeda membuka baris, dan menggeser
 * lebar seluruh kolom lain.
 *
 * # Apa yang sebenarnya terjadi saat diklik di Pega
 *
 * `pyEvent = click` menjalankan `pyAction = runActivity` atas
 * `SetAssignmentInboxPUCL_act`, dengan satu parameter: `inskey = .pzInsKey`.
 *
 * Activity itu sendiri **nyaris kosong** — `pyUsage = FLOW`, satu `Property-Set` ke
 * `TempIns.pyNote`. Ia kait pra-proses; yang bekerja adalah **Open Assignment** bawaan
 * Pega, yang membuka klaim pada tahap alur kerjanya saat itu supaya petugas dapat
 * MENGERJAKANNYA — bukan membacanya.
 *
 * Itu jalur TULIS: Open Assignment mengunci objek kerja, dan flow action yang menunggu di
 * sana (`SendtoRCLPUCL` → `SetDataLampiranSuratRCLPUCL_Act`) menyiapkan lampiran surat
 * RCL/PUCL — persis tindakan "Cetak Surat". Selama masa paralel, objek kerja dan
 * penugasannya masih dimiliki Pega (`P-1`).
 *
 * # Apa yang dibuka di sini
 *
 * Layar kerja `SendtoRCLPUCL` — kedua bagiannya, "Lampiran Surat" dan "Penerimaan
 * Dokumen". Ia **BACA SAJA**: bagian yang menulis masih dimiliki Pega (`P-1`).
 *
 * Section-nya sempat tidak ada di export dan modul ini sempat hanya menampilkan isi baris
 * grid. Work Owner menambahkan ketiga rule yang hilang pada 2026-09-24 — induknya beserta
 * kedua sub-section-nya — sehingga isinya kini terbaca dari bukti, bukan ditebak.
 *
 * # Kenapa PANEL, bukan halaman tujuan
 *
 * Karena yang dibuka bukan halaman baru melainkan tahap alur kerja klaim yang sama, dan
 * petugas kembali ke antreannya begitu selesai. Panel di halaman yang sama menjaga daftar
 * tetap di tempatnya — nomor halaman, tab, dan rentang tanggalnya tidak hilang.
 *
 * Mengarahkannya ke layar View Claim akan keliru: `ViewTempDetailClaim` memang ada, tetapi
 * dibuka `PNCInboxAdmin`, `PNCSearchKlaim`, `InboxManagerReopen1_Sec`, dan
 * `InputProgressClaim` — **tidak satu pun dari RCL/PUCL**.
 */
function CaseLink({ item, onOpen }: { item: WorkItem; onOpen: (row: WorkItem) => void }) {
  if (item.no_case === '') return <span className="text-slate-400">—</span>

  return (
    <button
      type="button"
      onClick={() => onOpen(item)}
      title={`Buka klaim ${item.no_case}`}
      className={[
        'rounded-kontrol text-left font-medium text-blue-700 underline-offset-2',
        'transition-colors duration-150 ease-halus hover:underline',
        'focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500/50',
        'disabled:cursor-not-allowed disabled:text-slate-400 disabled:no-underline',
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
 * berdampingan: tab "Klaim MSIG" yang kosong, isian tanggal yang tidak menyaring tabel,
 * dan berkas ekspor yang isinya berbeda dari tabel.
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
 * Layar lama tidak punya satu pun, dan versi pertama modul ini keliru menambahkannya.
 * Yang membuka baris adalah **tautan pada sel "Nomor Case"** — itulah bentuknya di ketiga
 * section RCL/PUCL (`D-13`).
 *
 * `value` tetap mengembalikan TEKS polos meski selnya digambar sebagai tautan. Keduanya
 * dipisah dengan sengaja oleh `DataTable`: yang dicari dan diurutkan adalah teksnya, yang
 * dilihat pengguna adalah gambarnya. Menyatukannya akan membuat pengurutan menelusuri
 * markup alih-alih nomor case.
 */
function columnsFor(tab: Tab, openCase: (row: WorkItem) => ReactNode): Column<WorkItem>[] {
  return tab.kolom.map((column) => {
    const base: Column<WorkItem> = {
      key: column.kunci,
      title: column.judul,
      value: (row) => cellText(row, column),
    }

    if (column.kunci === 'no_case') {
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
 *    layar ini dibaca dari kolom yang bentuknya tidak dapat diperiksa tanpa DDL (`R-08`).
 *    Memaksa pemformatan akan mengubah nilai yang tidak dikenali menjadi teks yang salah,
 *    dan itu lebih buruk daripada menampilkannya apa adanya.
 *  - Teks kosong menjadi tanda pisah, bukan sel kosong yang tidak dapat dibedakan dari
 *    kolom yang gagal dimuat. Di layar ini itu penting khusus: kolom "Tanggal Cetak Surat"
 *    memang SELALU kosong pada tab pertama, dan tanda pisah menyatakan "tidak ada isinya"
 *    alih-alih "gagal".
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
 * "Tidak ada data" tidak cukup di layar ini, dan alasannya lebih tajam daripada di layar
 * lain: ketiga tab punya kolom yang IDENTIK, sehingga pengguna yang melihat tabel kosong
 * tidak punya satu pun petunjuk apakah ia berada di tab yang salah, antreannya memang sepi,
 * atau ada yang keliru di konfigurasinya.
 *
 * Dua dari tiga pesan di bawah menunjuk KEKELIRUAN KONFIGURASI yang tidak menghasilkan satu
 * pun galat — dan keterangan inilah satu-satunya yang mengarahkan pengguna melapor alih-alih
 * menganggapnya wajar.
 */
function emptyMessageFor(tab: Tab): string {
  if (tab.punya_laporan_rentang_tanggal) {
    return (
      'Tidak ada klaim RCL/PUCL yang suratnya menunggu dicetak. Bila Anda yakin ' +
      'seharusnya ada, periksa tab "Kelengkapan Dokumen" lebih dulu — klaim yang suratnya ' +
      'sudah dicetak pindah ke sana dengan sendirinya.'
    )
  }

  if (tab.catatan) {
    // Tab "Klaim MSIG". Catatannya sudah menjelaskan kosongnya di atas grid, jadi pesan di
    // sini cukup mengarahkan ke sana alih-alih mengulangnya.
    return (
      'Tidak ada klaim jalur MSIG. Lihat catatan di atas tabel — tab ini memang ' +
      'diperkirakan kosong, dan penyebabnya sedang dipastikan ke DBA.'
    )
  }

  return (
    'Tidak ada klaim yang menunggu kelengkapan dokumen. Bila Anda yakin seharusnya ada, ' +
    'laporkan ke tim teknis: klaim yang penanda persetujuannya belum pernah diisi tidak ' +
    'muncul di sini, dan itu perilaku yang dibawa dari sistem lama.'
  )
}

/** messageOf mengambil pesan yang layak dibaca pengguna dari sebuah galat. */
function messageOf(error: unknown): string {
  if (error instanceof APIError) return error.message
  if (error instanceof Error && error.message !== '') return error.message
  return 'Coba lagi beberapa saat lagi. Bila terus berulang, hubungi tim teknis.'
}
