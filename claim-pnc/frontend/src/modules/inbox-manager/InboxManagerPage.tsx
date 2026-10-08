import { useEffect, useState, type ReactNode } from 'react'

import { APIError } from '@/api/client'
import { Button } from '@/components/Button'
import { ErrorMessage } from '@/components/ErrorMessage'

import { DashboardPanels } from './DashboardPanels'
import { ManagerTabs, QueueSubTabs } from './ManagerTabs'
import { QueuePanel } from './QueuePanel'
import {
  useExportInboxManager,
  useInboxManagerDecide,
  useInboxManagerList,
  useInboxManagerMetadata,
} from './api'
import type {
  DashboardFilterInput,
  Filter,
  Panel,
  PeriodInput,
  Tab,
  Verdict,
} from './types'

/** Penyaring kosong — padanan pilihan "All" di layar lama. */
const EMPTY_FILTER: DashboardFilterInput = { reinsurer: '', kategori_os: '' }

/**
 * Inbox Manager — menu `MENU_ID 58`, pengganti harness `UserInbox_Harness`.
 *
 * Isinya MEJA KERJA PENYELIA: tiga dashboard dan sembilan antrean persetujuan dalam satu
 * layar.
 *
 * # Susunan layar, dan dari mana bentuknya
 *
 * Harness-nya cangkang — satu `pyInclude` ke `Section/InboxManager_Sec`. Section itulah yang
 * menggambar ringkasan pencacah lalu menyertakan TIGA BELAS kontainer bersyarat
 * `FlagManager.AlasanKlaim==1` sampai `==13`. Angka 1..13 itu kode tab yang sesungguhnya, dan
 * dipakai apa adanya sebagai kode tab di sini.
 *
 * Di Pega, pencacah di kepala layar DAN kontainer di bawahnya adalah dua hal terpisah:
 * mengeklik pencacah menetapkan `FlagManager.AlasanKlaim`, lalu kontainer yang cocok muncul.
 * Di sini keduanya disatukan menjadi bilah tab berangka.
 *
 * # Modul ini MENULIS, dan itu membedakannya dari seluruh layar inbox lain
 *
 * Layar inbox lain baca-saja karena tabel objek kerja dan tabel penugasan masih dimiliki Pega
 * (`P-1`). Yang ditulis di sini adalah kolom PERSETUJUAN pada tabel POOLDATA.
 *
 * Satu jalur sengaja DITAHAN: menyetujui pembayaran pada tab Payment Klaim Akseptasi. Di Pega
 * persetujuan itu ikut menjalankan transfer ke kasir, dan rantainya belum lengkap. Alasannya
 * ditampilkan di tempat tombolnya, bukan disembunyikan.
 */
export function InboxManagerPage() {
  const [tabCode, setTabCode] = useState('')
  const [page, setPage] = useState(1)
  const [period, setPeriod] = useState<PeriodInput | null>(null)
  const [resultMessage, setResultMessage] = useState('')

  /*
    Penyaring dashboard disimpan DUA KALI: yang sedang diisi pengguna, dan yang sudah
    diterapkan.

    Itu bukan kerumitan yang dicari-cari — layar lama punya tombol "Filter" tersendiri, jadi
    mengubah dropdown di sana TIDAK langsung memuat ulang angkanya. Menyatukan keduanya akan
    membuat tombol itu tidak ada gunanya, dan setiap sentuhan dropdown menembak server.
  */
  const [draftFilter, setDraftFilter] = useState<DashboardFilterInput>(EMPTY_FILTER)
  const [filter, setFilter] = useState<DashboardFilterInput>(EMPTY_FILTER)

  const metadata = useInboxManagerMetadata()

  const tabs = metadata.data?.tab ?? []
  const activeCode = tabCode || metadata.data?.tab_bawaan || ''
  const activeTab = tabs.find((tab) => tab.kode === activeCode)

  const list = useInboxManagerList(
    activeCode,
    page,
    activeTab?.punya_penyaring_periode ? period : null,
    filter,
    Boolean(activeCode),
  )

  const decide = useInboxManagerDecide()
  const exportQueue = useExportInboxManager()

  /*
    Jenjang tab dibaca dari jawaban server, tidak ditebak dari kode tab.

    `induk` pada tab yang sedang terbuka menunjuk induknya; pada tab tingkat atas ia kosong,
    dan tab itulah induknya sendiri. Tidak ada satu pun kode tab yang ditulis di sini —
    menuliskan "4" akan membuat layar diam-diam salah bila urutan kontainer Pega berubah.
  */
  const parentTab = tabs.find((tab) => tab.kode === activeTab?.induk)
  const branchTab = parentTab ?? activeTab
  /*
    Bilah tingkat kedua memuat tab yang BENAR-BENAR ada di bilah tab Pega — delapan, bukan
    sembilan. Penolakan Klaim berinduk Approval Master pada pohon pencacah, tetapi
    `InboxManager_Section2` tidak menyertakannya; ia dibuka dari kartu ringkasan, sama seperti
    di Pega yang membukanya dari pencacahnya.
  */
  const childTabs = tabs.filter(
    (tab) => tab.induk === branchTab?.kode && tab.dalam_bilah_induk === true,
  )

  function selectTab(code: string) {
    /*
      Tab yang isinya HANYA bilah sub-tab langsung diteruskan ke sub-tab pertamanya.

      Di Pega, memilih "Approval Master" tidak menampilkan halaman antara: bilah tabnya
      muncul dengan sub-tab pertama sudah terbuka. Menyalakan tabnya sendiri akan menggambar
      layar kosong yang menuntut satu klik lagi untuk sampai ke tempat yang sama.

      Yang dicari kodenya BUKAN ditulis di sini: diambil dari anak pertama yang server
      nyatakan ada di bilah tab induknya.
    */
    const target = tabs.find((tab) => tab.kode === code)
    if (target?.jenis === 'ringkasan') {
      const first = tabs.find((tab) => tab.induk === code && tab.dalam_bilah_induk === true)
      if (first) code = first.kode
    }

    setTabCode(code)
    setPage(1)
    setResultMessage('')
    decide.reset()

    // Penyaring dashboard DIBUANG saat tab berganti. Ia hanya berlaku pada satu tab, dan
    // membiarkannya menempel akan membuat tab lain memuat dengan penyaring yang tidak
    // terlihat di layarnya.
    setDraftFilter(EMPTY_FILTER)
    setFilter(EMPTY_FILTER)
  }

  function submitDecision(verdict: Verdict, keys: string[], reason: string) {
    if (!activeTab) return

    setResultMessage('')
    decide.mutate(
      { tab: activeTab.kode, keputusan: verdict, kunci: keys, alasan: reason },
      { onSuccess: (response) => setResultMessage(response.pesan) },
    )
  }

  const exportable = activeTab?.jenis === 'antrean'

  if (metadata.isError) {
    return (
      <PageFrame
        exportable={false}
        isExporting={false}
        onExport={() => {
          /* tidak ada yang dapat diekspor saat bentuk layar gagal dimuat */
        }}
      >
        <ErrorMessage
          tone="gangguan"
          title="Layar tidak dapat dimuat"
          description={messageOf(metadata.error, 'Keterangan layar tidak dapat diambil.')}
        />
      </PageFrame>
    )
  }

  return (
    <PageFrame
      exportable={exportable}
      isExporting={exportQueue.isPending}
      onExport={() => activeTab && exportQueue.mutate(activeTab.kode)}
    >

      {/*
        Lini bisnis yang terbaca ditampilkan APA ADANYA, termasuk saat kosong.

        Ia yang menyaring ketiga dashboard, dan dua penyelia yang membandingkan layar
        masing-masing akan melihat angka berbeda tanpa satu pun petunjuk kenapa bila nilai
        ini disembunyikan.
      */}
      <LineBusinessNote line={metadata.data?.lini_bisnis_anda ?? ''} loading={metadata.isLoading} />

      <ManagerTabs tabs={tabs} active={activeCode} onSelect={selectTab} />

      {/*
        Bilah tingkat kedua hanya muncul pada cabang yang MEMANG punya anak. Pada ketiga
        dashboard ia tidak digambar sama sekali — bukan digambar kosong.
      */}
      {branchTab && childTabs.length > 0 ? (
        <QueueSubTabs
          parent={branchTab}
          tabs={childTabs}
          active={activeCode}
          onSelect={selectTab}
        />
      ) : null}

      {activeTab ? (
        <TabBody
          tab={activeTab}
          list={list}
          page={page}
          onPageChange={setPage}
          period={period}
          onPeriodChange={(next) => {
            setPeriod(next)
            setPage(1)
          }}
          draftFilter={draftFilter}
          onDraftChange={setDraftFilter}
          onApplyFilter={() => setFilter(draftFilter)}
          onClearFilter={() => {
            setDraftFilter(EMPTY_FILTER)
            setFilter(EMPTY_FILTER)
          }}
          onSelectTab={selectTab}
          onDecide={submitDecision}
          isDeciding={decide.isPending}
          resultMessage={resultMessage}
          decideError={decide.isError ? messageOf(decide.error, 'Keputusan gagal.') : undefined}
        />
      ) : null}

      {/*
        Blok "Yang berbeda dari layar lama" DICABUT dari layar (2026-10-07). Ia tidak ada di
        Pega, dan catatan tempelan seperti itu sudah pernah ditolak Work Owner pada modul
        lain — termasuk saat dicoba disembunyikan di balik mode pengembangan.

        Daftarnya TETAP dikirim server (`selisih_terencana`): ia dipakai uji kesetaraan
        gerbang 1 untuk memetakan setiap selisih ke butir `P-5` (`D-54`). Yang dicabut hanya
        penggambarannya.
      */}
    </PageFrame>
  )
}

/**
 * Pembungkus halaman.
 *
 * # Kenapa ia ada, dan kenapa ukurannya BUKAN pilihan bebas
 *
 * Setiap layar modul membungkus isinya sendiri — tidak ada pembungkus bersama di `App.tsx`.
 * Layar tanpa pembungkus karena itu menempel ke bilah samping tanpa jarak sama sekali, dan
 * terlihat berbeda dari seluruh layar lain meski isinya benar.
 *
 * Lebar dan jaraknya disalin dari modul Inbox Manager Admin — layar yang harness-nya klon
 * `UserInbox_Harness`, sehingga keduanya memang layak terlihat sekeluarga. `max-w-[96rem]`,
 * bukan `max-w-7xl` yang dipakai layar berkolom sedikit: layar ini menggambar dua grid
 * berdampingan dengan sampai sembilan kolom.
 *
 * # Tombol ekspor berada DI SINI, bukan di atas tabelnya
 *
 * Sama seperti modul-modul sebelumnya, dan alasannya bukan kerapian: tempat yang tetap
 * membuat pengguna tidak perlu mencarinya ulang setiap berpindah tab. Ia dimatikan — bukan
 * disembunyikan — pada tab yang tidak dapat diekspor, sehingga tidak ada tombol yang muncul
 * dan hilang.
 */
function PageFrame({
  exportable,
  isExporting,
  onExport,
  children,
}: {
  exportable: boolean
  isExporting: boolean
  onExport: () => void
  children: ReactNode
}) {
  return (
    <div className="mx-auto max-w-[96rem] px-4 py-8">
      <header className="flex flex-wrap items-start justify-between gap-4 border-b border-slate-200 pb-4">
        <div>
          <h1 className="text-xl font-semibold text-slate-900">Inbox Manager</h1>
          <p className="mt-1 text-sm text-slate-600">
            Meja kerja penyelia: tiga dashboard dan sembilan antrean persetujuan dalam satu
            layar.
          </p>
          {/*
            Sifat layar ini dinyatakan di layar, bukan hanya di kode.

            Ia satu-satunya layar inbox yang MENULIS, dan tidak satu pun tabnya menyaring
            menurut pengguna yang login. Tanpa keterangan ini, petugas yang terbiasa dengan
            layar "milik saya" akan mengira daftarnya keliru karena memuat pekerjaan orang
            lain.
          */}
          <p className="mt-2 text-xs text-slate-500">
            Layar ini menampilkan pengajuan{' '}
            <span className="font-medium">seluruh petugas</span>, bukan hanya milik Anda.
            Setiap keputusan yang Anda kirim tercatat atas nama Anda.
          </p>
        </div>

        <div className="flex flex-col items-end gap-1">
          <Button type="button" tone="kedua" onClick={onExport} disabled={!exportable || isExporting}>
            {isExporting ? 'Menyiapkan…' : 'Export'}
          </Button>
          {!exportable ? (
            <span className="text-xs text-slate-500">
              Hanya antrean persetujuan yang dapat diekspor.
            </span>
          ) : null}
        </div>
      </header>

      <div className="mt-5 space-y-5">{children}</div>
    </div>
  )
}

/**
 * Keterangan lini bisnis pemanggil.
 *
 * Dua keadaan dibedakan, karena keduanya menuntut tindakan yang berbeda:
 *
 *   kosong  master pengguna perlu DILENGKAPI
 *   terisi  angka dashboard disaring menurut nilai itu
 *
 * Keduanya dulu menghasilkan layar yang sama, dan laporan pengguna karena itu tidak dapat
 * ditindaklanjuti.
 */
function LineBusinessNote({ line, loading }: { line: string; loading: boolean }) {
  if (loading) return null

  if (line === '') {
    return (
      <p className="rounded-kontrol border border-slate-200 bg-slate-50 p-3 text-sm text-slate-700">
        Lini bisnis Anda <span className="font-medium">belum diisi</span> pada master pengguna,
        sehingga ketiga dashboard menampilkan SELURUH lini bisnis pada entitas ini. Bila Anda
        seharusnya melihat satu lini saja, sampaikan ke tim teknis — nilainya dapat dilengkapi
        tanpa perubahan aplikasi.
      </p>
    )
  }

  return (
    <p className="text-sm text-slate-600">
      Angka dashboard disaring untuk lini bisnis{' '}
      <span className="font-medium text-slate-800">{line}</span>.
    </p>
  )
}

type TabBodyProps = {
  tab: Tab
  list: ReturnType<typeof useInboxManagerList>
  page: number
  onPageChange: (page: number) => void
  period: PeriodInput | null
  onPeriodChange: (period: PeriodInput | null) => void
  draftFilter: DashboardFilterInput
  onDraftChange: (filter: DashboardFilterInput) => void
  onApplyFilter: () => void
  onClearFilter: () => void
  onSelectTab: (code: string) => void
  onDecide: (verdict: Verdict, keys: string[], reason: string) => void
  isDeciding: boolean
  resultMessage: string
  decideError?: string | undefined
}

/** Isi tab yang sedang terbuka, digambar menurut JENISNYA. */
function TabBody(props: TabBodyProps) {
  const { tab, list } = props

  /*
    Galat pemuatan digambar sebagai PENGGANTI ISI, bukan pengganti seluruh badan tab.

    Penyaring periode harus tetap tergambar, karena galat yang paling sering muncul di sini
    justru BERASAL dari isian periode — rentang lintas tahun yang ditolak. Mengganti seluruh
    badan tab berarti pesan "perbaiki yang ditandai" muncul tanpa ada yang dapat diperbaiki:
    isiannya ikut hilang, dan satu-satunya jalan keluar adalah memuat ulang halaman.

    # Satu galat sengaja TIDAK digambar

    `sumber_antrean_tidak_terbaca` — jawaban 503 saat objek sumber antrean belum ada atau
    sedang tidak sah di basis data. Pesannya ditujukan ke tim teknis, bukan ke pengguna, dan
    sebabnya memang keadaan pengembangan: tabelnya belum dibuat. Menggambarnya membuat layar
    tampak rusak padahal tidak ada yang dapat dilakukan pengguna atasnya (Work Owner,
    2026-10-08).

    Servernya TIDAK diubah: ia tetap menjawab 503 dengan kode itu dan tetap mencatat nama
    objeknya di log, sehingga pemantauan masih dapat membedakannya dari kegagalan tak terduga.
    Yang berubah hanya apa yang digambar — antreannya tampil kosong, sama seperti antrean yang
    memang belum ada isinya.
  */
  const sourceDown =
    list.error instanceof APIError && list.error.kode === 'sumber_antrean_tidak_terbaca'

  const failure = list.isError && !sourceDown ? (
    <ErrorMessage
      tone="gangguan"
      title={`"${tab.nama}" tidak dapat dimuat`}
      description={messageOf(list.error, 'Isi bagian ini tidak dapat diambil.')}
    />
  ) : null

  /*
    Tab "Approval Master" TIDAK punya isi sendiri.

    Ia sempat menggambar kumpulan kartu berangka — satu per antrean. Itu bukan bentuk Pega:
    di sana memilih Approval Master langsung membuka bilah sub-tabnya, dan sub-tab pertama
    (`Master Bengkel`) sudah terbuka. Kartunya karena itu dicabut, dan layar BERPINDAH ke
    antrean pertama begitu tab ini dipilih — lihat selectTab.

    Yang tergambar selama perpindahan hanyalah keterangan singkat, bukan kekosongan.
  */
  if (tab.jenis === 'ringkasan') {
    if (failure) return failure
    return <p className="text-sm text-slate-600">Membuka antrean pertama…</p>
  }

  if (tab.jenis === 'dashboard') {
    /*
      Gridnya TETAP digambar meski isinya belum ada — termasuk saat permintaannya ditolak.

      Pega berperilaku begitu: pesan alert muncul, dan tabelnya tetap berdiri di tempatnya.
      Menghapus seluruh tabel saat penyaringnya salah membuat layar seolah kehilangan isinya,
      padahal yang salah hanya satu isian.

      Saat isinya tidak ada, bentuk gridnya diambil dari keterangan layar (`tab.panel`) yang
      memang memuat kolomnya tanpa barisnya. Satu-satunya yang tidak dapat dipulihkan dari
      sana adalah kolom TAHUN pada grid silang, karena tahun mana saja ditentukan data.
    */
    const panels: Panel[] =
      list.data?.panel ?? (tab.panel ?? []).map((shape) => ({ ...shape, baris: [] }))

    /*
      Penyaring digambar DI ANTARA grid kedua dan ketiga, mengikuti urutan layar lama: kedua
      grid ringkasan di atas, lalu Reinsurer, Kategori OS, tombolnya, baru tabel silang.

      Urutannya bukan selera — penyaring itu hanya mengenai grid ketiga, dan menaruhnya di
      atas kedua grid pertama akan menyiratkan ia menyaring ketiganya.
    */
    const summary = panels.slice(2)
    const filters = list.data?.penyaring ?? []

    return (
      <div className="space-y-4">
        {tab.punya_ekspor_detail ? <DetailExportBlock /> : null}

        {tab.punya_penyaring_periode ? (
          <PeriodFilter
            tab={tab}
            value={props.period}
            onChange={props.onPeriodChange}
            applied={list.data?.periode}
          />
        ) : null}

        {/*
          Grid digambar ATAS-BAWAH, bukan berdampingan — ketetapan Work Owner 2026-10-07.

          Pega memang menaruhnya bersebelahan, dan itu sempat ditiru. Berdampingan membuat
          kolomnya berdempetan: grid Produktivitas punya sembilan kolom dan grid Klaim
          sepuluh, sehingga separuh lebar layar memaksa setiap kolom menyempit sampai
          judulnya terpotong. Atas-bawah memberi tiap grid lebar penuh.
        */}
        {failure}

        <DashboardPanels
          panels={panels.slice(0, 2)}
          isLoading={list.isLoading}
          refreshedAt={list.data?.disegarkan_pada}
        />

        {filters.length > 0 ? (
          <DashboardFilters
            filters={filters}
            draft={props.draftFilter}
            onDraftChange={props.onDraftChange}
            onApply={props.onApplyFilter}
            onClear={props.onClearFilter}
          />
        ) : null}

        {summary.length > 0 ? (
          <DashboardPanels panels={summary} isLoading={list.isLoading} />
        ) : null}
      </div>
    )
  }

  /*
    Antrean pun TIDAK dihapus saat permintaannya gagal — kolomnya tetap digambar di bawah
    pesannya, sama seperti dashboard. Kolom antrean selalu diketahui dari keterangan layar,
    sehingga tidak ada yang perlu disimpulkan dari data.
  */
  return (
    <div className="space-y-3">
      {failure}

      <QueuePanel
        tab={tab}
        rows={list.data?.baris ?? []}
        pagination={list.data?.paginasi}
        isLoading={list.isLoading}
        page={props.page}
        onPageChange={props.onPageChange}
        onDecide={props.onDecide}
        isDeciding={props.isDeciding}
        resultMessage={props.resultMessage}
        decideError={props.decideError}
      />
    </div>
  )
}

/**
 * Kedua penyaring di bawah grid ringkasan — Reinsurer dan Kategori OS.
 *
 * # Pilihannya datang dari SERVER, termasuk pilihan "All"
 *
 * "Kategori OS" dibaca dari master tahapan progres klaim yang dapat berubah tanpa deploy, dan
 * di Pega pun begitu (`BrowseMstProgress1`). Menyusunnya di layar berarti kategori baru tidak
 * pernah muncul sampai ada yang mengubah kode.
 *
 * # Tombolnya DUA, dan keduanya ada di layar lama
 *
 * `pyButtonLabel Filter` dan `pyButtonLabel Clear Filter`. Memuat ulang setiap kali dropdown
 * disentuh akan membuat tombol Filter tidak ada gunanya, dan menembak server tiga kali untuk
 * satu perubahan yang disengaja.
 */
function DashboardFilters({
  filters,
  draft,
  onDraftChange,
  onApply,
  onClear,
}: {
  filters: Filter[]
  draft: DashboardFilterInput
  onDraftChange: (filter: DashboardFilterInput) => void
  onApply: () => void
  onClear: () => void
}) {
  function valueOf(key: string): string {
    return key === 'reinsurer' ? draft.reinsurer : draft.kategori_os
  }

  function change(key: string, value: string) {
    onDraftChange(
      key === 'reinsurer' ? { ...draft, reinsurer: value } : { ...draft, kategori_os: value },
    )
  }

  return (
    <div className="space-y-3 rounded-kontrol border border-slate-200 bg-slate-50 p-4">
      <div className="flex flex-wrap items-end gap-4">
        {filters.map((filter) => (
          <label key={filter.kunci} className="block text-sm">
            <span className="mb-1 block font-medium text-slate-700">{filter.label}</span>
            <select
              value={valueOf(filter.kunci)}
              onChange={(event) => change(filter.kunci, event.target.value)}
              className="min-w-48 rounded-kontrol border border-slate-300 bg-white px-3 py-2 text-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500"
            >
              {filter.pilihan.map((option) => (
                <option key={option.nilai || 'semua'} value={option.nilai}>
                  {option.label}
                </option>
              ))}
            </select>
          </label>
        ))}
      </div>

      <div className="flex flex-wrap items-center gap-2">
        <Button type="button" onClick={onApply}>
          Filter
        </Button>
        <Button type="button" tone="kedua" onClick={onClear}>
          Clear Filter
        </Button>
      </div>
    </div>
  )
}

/**
 * Blok "Export Data Detail Klaim".
 *
 * # Kenapa isiannya digambar tetapi tombolnya DIMATIKAN
 *
 * Karena bloknya memang ada di layar lama, dan menyembunyikannya berarti menyembunyikan
 * kemampuan yang dijanjikan. Tetapi isinya BELUM dapat dibangun dengan jujur: ekspor itu
 * `Activity/ExportDataDetailKlaim` lewat `GetExportDataDetailKlaim`, sebuah kueri 68 kolom di
 * atas `POOLDATA.PEGA_DASHBOARDPNC` — dan lima kolom yang dibutuhkannya dari tabel kerja
 * (`SURVEYORTYPE_1`, `ADJUSTERPIC_1`, `SURVEYORNAME_1`, `SURVEYORNAMEMARINE_1`,
 * `ISPENDINGCLOSE`) kosong di seluruh 1.014 baris `T_CLAIMLIST_ADMIN`.
 *
 * Berkas dengan kolom kosong yang tidak diterangkan lebih buruk daripada tombol yang
 * dimatikan beserta alasannya. Alasannya karena itu ditulis di tempat tombolnya.
 */
function DetailExportBlock() {
  return (
    <section className="space-y-3 rounded-kontrol border border-slate-200 bg-white p-4">
      <h2 className="text-sm font-semibold text-slate-800">Export Data Detail Klaim</h2>

      <div className="flex flex-wrap items-end gap-4">
        <label className="block text-sm">
          <span className="mb-1 block font-medium text-slate-700">Dari</span>
          <input
            type="date"
            disabled
            className="rounded-kontrol border border-slate-300 bg-slate-50 px-3 py-2 text-sm text-slate-500"
          />
        </label>
        <label className="block text-sm">
          <span className="mb-1 block font-medium text-slate-700">Sampai</span>
          <input
            type="date"
            disabled
            className="rounded-kontrol border border-slate-300 bg-slate-50 px-3 py-2 text-sm text-slate-500"
          />
        </label>

        <Button type="button" disabled>
          Export to Excel
        </Button>
      </div>

      <p className="rounded-kontrol border border-amber-200 bg-amber-50 p-3 text-sm text-amber-900">
        <span className="font-medium">Ekspor ini belum dapat dijalankan. </span>
        Di Pega ia laporan 68 kolom di atas tabel cuplikan `PEGA_DASHBOARDPNC`, dan lima kolom
        yang dibutuhkannya belum terisi di tabel sumber yang dipakai layar ini. Membuatnya
        sekarang akan menghasilkan berkas berkolom kosong tanpa keterangan.
      </p>
    </section>
  )
}

/**
 * Penyaring periode.
 *
 * Kedua bentuknya dibaca dari `Section/DisplayInboxProduktivitas_Sect`, yang punya
 * `<pyCaption Periode>`, `<pyCaption Bulan & Tahun>`, `<pyCaption Dari>`, dan
 * `<pyCaption Sampai>`.
 *
 * Periode yang BENAR-BENAR dipakai ditampilkan di bawahnya, termasuk saat ia nilai bawaan.
 * Periode bawaan yang tidak ditampilkan membuat angka di layar tidak dapat dipertanggungkan:
 * dua orang akan membandingkan angka dari rentang yang berbeda tanpa menyadarinya.
 */
function PeriodFilter({
  tab,
  value,
  onChange,
  applied,
}: {
  tab: Tab
  value: PeriodInput | null
  onChange: (period: PeriodInput | null) => void
  applied?: { dari: string; sampai: string } | undefined
}) {
  /*
    Isiannya DRAF, dan baru berlaku saat tombolnya ditekan — mengikuti Pega, yang punya tombol
    tersendiri untuk ini pada kedua tab berperiode.

    Tanpa draf, setiap potongan isian tanggal menjadi permintaan tersendiri. Isian tanggal
    peramban diisi segmen demi segmen, sehingga mengetik tahun "2026" melewati keadaan "0002"
    lalu "0020" — keduanya sah secara bentuk, dan keduanya ditembakkan ke server sebagai
    permintaan yang pasti ditolak.
  */
  const [draft, setDraft] = useState<PeriodInput>(
    value ?? { bentuk: 'bulan', bulan: '', dari: '', sampai: '' },
  )

  // Draf disetel ulang ketika periode yang BERLAKU berubah dari luar — misalnya saat tab
  // berganti dan penyaringnya dibuang.
  useEffect(() => {
    setDraft(value ?? { bentuk: 'bulan', bulan: '', dari: '', sampai: '' })
  }, [value])

  const mode = draft.bentuk

  function update(next: Partial<PeriodInput>) {
    setDraft({ ...draft, ...next })
  }

  /*
    Tombolnya mati HANYA pada keadaan setengah terisi — rentang yang baru punya satu tanggal.
    Itu satu-satunya keadaan yang pasti ditolak server, dan menahannya di sini menghemat
    perjalanan yang hasilnya sudah diketahui.

    Isian KOSONG tetap dapat ditekan, dan artinya "tanpa penyaring periode". Mematikannya di
    sana akan membuat penyaring yang sudah terpasang tidak dapat dihapus sama sekali.
  */
  const complete =
    mode === 'bulan' ? draft.bulan !== '' : draft.dari !== '' && draft.sampai !== ''
  const empty = mode === 'bulan' ? draft.bulan === '' : draft.dari === '' && draft.sampai === ''

  return (
    <div className="space-y-2 rounded-kontrol border border-slate-200 bg-slate-50 p-4">
      <div className="flex flex-wrap items-end gap-3">
        <label className="text-sm">
          <span className="mb-1 block font-medium text-slate-700">Periode</span>
          <select
            value={mode}
            onChange={(event) =>
              update({ bentuk: event.target.value as PeriodInput['bentuk'] })
            }
            className="rounded-kontrol border border-slate-300 px-3 py-2 text-sm"
          >
            <option value="bulan">Bulan &amp; Tahun</option>
            <option value="rentang">Rentang tanggal</option>
          </select>
        </label>

        {mode === 'bulan' ? (
          <label className="text-sm">
            <span className="mb-1 block font-medium text-slate-700">Bulan &amp; Tahun</span>
            <input
              type="month"
              value={draft.bulan}
              onChange={(event) => update({ bulan: event.target.value })}
              className="rounded-kontrol border border-slate-300 px-3 py-2 text-sm"
            />
          </label>
        ) : (
          <>
            <label className="text-sm">
              <span className="mb-1 block font-medium text-slate-700">Dari</span>
              <input
                type="date"
                value={draft.dari}
                onChange={(event) => update({ dari: event.target.value })}
                className="rounded-kontrol border border-slate-300 px-3 py-2 text-sm"
              />
            </label>
            <label className="text-sm">
              <span className="mb-1 block font-medium text-slate-700">Sampai</span>
              <input
                type="date"
                value={draft.sampai}
                onChange={(event) => update({ sampai: event.target.value })}
                className="rounded-kontrol border border-slate-300 px-3 py-2 text-sm"
              />
            </label>
          </>
        )}

        {/*
          Satu tombol saja, dan labelnya datang dari server — Pega memberi label berbeda pada
          kedua tab. Tombol "Hapus penyaring" yang sempat ada di sini TIDAK ada di Pega;
          penyaring dikosongkan dengan mengosongkan isiannya lalu menekan tombol ini.
        */}
        <Button
          type="button"
          onClick={() => onChange(complete ? draft : null)}
          disabled={!complete && !empty}
        >
          {tab.label_terapkan_periode ?? 'Cari'}
        </Button>
      </div>

      <p className="text-sm text-slate-600">
        {applied ? (
          <>
            Angka di bawah dihitung untuk{' '}
            <span className="font-medium text-slate-800">
              {applied.dari} sampai {applied.sampai}
            </span>
            {tab.kode === '2' ? ', dibandingkan dengan rentang yang sama tahun lalu.' : '.'}
          </>
        ) : (
          'Tanpa penyaring periode: angka di bawah mencakup seluruh periode.'
        )}
      </p>
    </div>
  )
}

/**
 * Daftar selisih terhadap sistem lama yang sudah diputuskan.
 *
 * Ia ditampilkan kepada pengguna, bukan hanya dicatat di komentar: selisih yang tidak
 * dinyatakan akan dilaporkan berulang kali sebagai kerusakan oleh orang yang membandingkan
 * layar ini dengan Pega berdampingan (`D-54`).
 */
/** messageOf membaca pesan galat yang layak dibaca pengguna. */
/**
 * messageOf membaca pesan galat yang layak dibaca pengguna.
 *
 * # Pelanggaran per isian didahulukan di atas pesan amplopnya
 *
 * Pesan amplop galat validasi berbunyi "Permintaan belum benar. Perbaiki yang ditandai lalu
 * coba lagi." — kalimat yang tidak menyebut APA yang ditandai. Yang menyebutkannya ada di
 * `detail`, dan di layar ini salah satunya disalin langsung dari Pega:
 * "Periode Up To hanya untuk periode tahun yang sama".
 *
 * Seluruh pelanggaran digabung, bukan yang pertama saja — server memang mengirimkan semuanya
 * sekaligus, meniru Pega yang menampilkan semua pesannya bersamaan (`P-5`).
 */
function messageOf(error: unknown, fallback: string): string {
  if (error instanceof APIError) {
    const violations = Object.values(error.violations()).filter((text) => text !== '')
    if (violations.length > 0) return violations.join(' ')
    if (error.message) return error.message
  }
  if (error instanceof Error && error.message) return error.message
  return fallback
}
