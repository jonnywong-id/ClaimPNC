import { useState } from 'react'

import { APIError } from '@/api/client'
import { useSelectedPortal } from '@/app/portal'
import { Button } from '@/components/Button'
import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage } from '@/components/ErrorMessage'
import { FormField } from '@/components/FormField'
import { SelectField, type SelectOption } from '@/components/SelectField'

import { ReportKPIAdmin } from './ReportKPIAdmin'
import { ReportKPIPICTeknik } from './ReportKPIPICTeknik'
import { ReportKPITabs } from './ReportKPITabs'
import {
  useExportReportKPI,
  useReportKPIAdjusters,
  useReportKPIDetail,
  useReportKPIMetadata,
  useReportKPISummary,
} from './api'
import type {
  Component,
  DetailRow,
  FilterInput,
  Grid,
  MetadataResponse,
  Scores,
  SummaryRow,
} from './types'

/**
 * Report KPI PNC — butir menu `MENU_ID 84`, pengganti harness `ReportKPIHarness`.
 *
 * # Tiga tab, dua di antaranya sudah dibangun
 *
 *   KPI PIC Teknik  belum — alasannya tertulis pada bilah tab
 *   KPI Adjuster    kinerja adjuster EKSTERNAL, sembilan komponen per kasus survei
 *   KPI Admin       kinerja tim ADMIN REGISTRASI, berupa kartu skor per kelompok
 *
 * Berkas ini menggambar kerangkanya dan tab KPI Adjuster; tab KPI Admin digambar
 * `ReportKPIAdmin`. Keduanya dipisah karena bentuk keluarannya memang berbeda — yang satu
 * dua tabel, yang lain kartu skor beserta satu tabel.
 *
 * # Susunan layar, dan dari mana bentuknya
 *
 * Diambil dari `Section/ReportKPI_Section-Section.xml`: bilah tiga tab, penyaring
 * (Pilih Tipe Report · Pilih Adjuster · Periode Dari–Sampai · tombol Cari · Export Data),
 * lalu DUA grid — "Summary KPI Adjuster" dan "Detail KPI Adjuster". Judul kolom TIDAK
 * diterjemahkan: `D-13` menetapkan tampilan meniru Pega, dan itulah teks yang selama ini
 * dibaca pengguna.
 *
 * # Dua hal yang paling mudah disalahpahami di layar ini
 *
 * PERTAMA — layar ini **membaca saja**. Di Pega, menekan Cari MENGHITUNG ULANG penilaian
 * tiap kasus lalu MENYIMPANNYA. Di sini tabelnya hanya dibaca (`P-1`), sehingga kasus yang
 * belum pernah dihitung Pega belum muncul. Itu dinyatakan di panel selisih terencana.
 *
 * KEDUA — kesembilan kolom komponen berisi angka yang sama-sama 1–5. Satu kolom yang
 * tergeser tidak terlihat sebagai kerusakan; ia hanya terlihat sebagai nilai yang berbeda.
 * Karena itu susunan kolomnya dibangun dari `komponen` yang dikirim server, bukan ditulis
 * ulang di sini.
 */
export function ReportKPIPage() {
  const portal = useSelectedPortal((state) => state.alias)

  const metadata = useReportKPIMetadata()

  // Penyaring dipegang DUA kali: satu yang sedang diketik, satu yang sudah dikirim.
  //
  // Pemisahan ini mengikuti layar lama, yang punya tombol "Cari" — bukan menyaring saat
  // mengetik. Alasannya masih berlaku di sini: mengubah satu isian memicu tiga permintaan
  // sekaligus (ringkasan, rincian, daftar pilihan), dan melakukannya per ketikan pada
  // tanggal yang setengah jadi hanya menghasilkan penolakan validasi beruntun.
  const [draft, setDraft] = useState<FilterInput>(emptyFilter)
  const [applied, setApplied] = useState<FilterInput | null>(null)
  const [page, setPage] = useState(1)

  // Tab yang sedang terbuka. Kosong berarti "ikuti tab bawaan dari server" — layar tidak
  // menebak sendiri kode tabnya, karena kode itu kontrak yang dimiliki server.
  const [tab, setTab] = useState('')

  const active = applied ?? emptyFilter
  const searched = applied !== null

  const summary = useReportKPISummary(active, searched)
  const detail = useReportKPIDetail(active, page, searched)

  // Daftar pilihan adjuster memakai penyaring yang SEDANG DIKETIK, bukan yang sudah
  // dikirim — pengguna memilih adjuster SEBELUM menekan Cari, dan daftar yang menunggu
  // Cari akan selalu kosong pada pemakaian pertama.
  //
  // Ia hanya diambil setelah tipe report dan kedua tanggal terisi; tanpa itu permintaannya
  // pasti ditolak validasi, dan penolakan yang tidak diminta pengguna hanya mengisi log.
  // Tanpa argumen, dan itu disengaja: daftarnya master dan terisi sejak layar dibuka —
  // bukan menunggu isian lengkap. Lihat useReportKPIAdjusters.
  const adjusters = useReportKPIAdjusters()

  const exportFile = useExportReportKPI()

  const components = metadata.data?.komponen ?? []
  const tabs = metadata.data?.tab ?? []

  // Pelanggaran per isian datang dari server, bukan dihitung ulang di layar.
  //
  // Aturannya hidup di domain (`internal/reportkpi/query.go`), dan menyalinnya ke sini
  // berarti dua tempat yang dapat berselisih — dengan yang di layar selalu menang lebih
  // dulu, sehingga selisihnya tidak pernah terlihat.
  const violations = violationsOf(summary.error ?? detail.error)

  const summaryGrid = gridOf(metadata.data, 'ringkasan')
  const detailGrid = gridOf(metadata.data, 'rincian')

  // Tab yang digambar: pilihan pengguna bila ada, kalau tidak tab bawaan dari server.
  const activeTab = tab || (metadata.data?.tab_bawaan ?? '')

  function search() {
    setApplied(draft)
    setPage(1)
  }

  return (
    // Jarak tepi dan lebar maksimum MENGIKUTI modul lain — bandingkan
    // `report-klaim/ReportKlaimPage.tsx`. Sebelumnya hanya `space-y-6`, sehingga isinya
    // menempel ke sidebar dan terbaca berbeda dari layar sebelahnya.
    <div className="mx-auto max-w-6xl space-y-5 p-4 sm:p-6">
      <header className="space-y-1">
        <h1 className="text-xl font-semibold text-slate-900">Report KPI</h1>
        <p className="text-sm text-slate-600">
          Penilaian kinerja adjuster eksternal dan tim admin registrasi
          {portal ? ` — entitas ${portal}` : ''}.
        </p>
      </header>

      {metadata.isError && (
        <ErrorMessage
          title="Keterangan layar tidak dapat diambil"
          description={messageOf(metadata.error)}
          tone="gangguan"
        />
      )}

      {tabs.length > 0 && (
        <ReportKPITabs tabs={tabs} active={activeTab} onSelect={setTab} />
      )}

      {/*
        Tab KPI Admin digambar komponen tersendiri. Ia BUKAN varian dari tab Adjuster:
        penyaringnya berbeda, bentuk keluarannya berbeda (kartu skor, bukan tabel), dan
        kolomnya berbeda. Menyatukannya menjadi satu komponen bersyarat akan membuat setiap
        perubahan pada salah satu tab menyentuh yang lain.
      */}
      {activeTab === 'admin' ? (
        <ReportKPIAdmin metadata={metadata.data} />
      ) : activeTab === 'pic-teknik' ? (
        <ReportKPIPICTeknik metadata={metadata.data} />
      ) : (
        <>
          <FilterForm
            value={draft}
            onChange={setDraft}
            reportTypes={metadata.data?.tipe_report ?? []}
            adjusters={adjusters.data?.adjuster ?? []}
            adjustersLoading={adjusters.isFetching}
            violations={violations}
            onSearch={search}
            onExport={(grid) => exportFile.mutate({ filter: active, grid })}
            exportEnabled={searched}
            exporting={exportFile.isPending}
          />

          {exportFile.isError && (
            <ErrorMessage
              title="Berkas tidak dapat diunduh"
              description={messageOf(exportFile.error)}
              tone="gangguan"
            />
          )}

          {!searched ? (
            <p className="rounded-kotak border border-slate-200 bg-slate-50 px-4 py-6 text-sm text-slate-600">
              Pilih tipe report dan periode, lalu tekan <strong>Cari</strong>. Layar lama
              pun menuntut keduanya sebelum menampilkan apa pun.
            </p>
          ) : (
            <>
              {/*
                Hanya grid yang DIPILIH yang digambar — Pega pun satu grid, ditentukan
                isian "Pilih Tipe Report". Menggambar keduanya sekaligus memaksa dua
                tombol ekspor, dan itu yang dulu membuat layar ini berbeda.
              */}
              {active.variant !== VARIANT_DETAIL && (
              <DataTable<SummaryRow>
                title={summaryGrid?.judul ?? 'Summary KPI Adjuster'}
                label="Ringkasan KPI per adjuster"
                // Tipe report diambil dari penyaring yang DIJAWAB server, bukan dari isian
                // yang sedang diketik: kolomnya harus cocok dengan baris yang sedang
                // tampil, bukan dengan penyaring yang belum dikirim.
                columns={summaryColumns(
                  summaryGrid,
                  components,
                  summary.data?.penyaring.tipe_report ?? active.tipeReport,
                )}
                rows={summary.data?.baris ?? []}
                rowKey={(row) => `${row.adjuster}|${row.tipe}`}
                isLoading={summary.isPending}
                error={
                  summary.isError ? (
                    <ErrorMessage
                      title="Ringkasan tidak dapat diambil"
                      description={messageOf(summary.error)}
                      tone="gangguan"
                    />
                  ) : undefined
                }
                emptyMessage={emptyMessage}
              />
              )}

              {active.variant === VARIANT_DETAIL && (
              <DataTable<DetailRow>
                title={detailGrid?.judul ?? 'Detail KPI Adjuster'}
                label="Rincian KPI per kasus survei"
                columns={detailColumns(
                  detailGrid,
                  components,
                  detail.data?.penyaring.tipe_report ?? active.tipeReport,
                )}
                rows={detail.data?.baris ?? []}
                rowKey={(row) => row.no_case}
                isLoading={detail.isPending}
                error={
                  detail.isError ? (
                    <ErrorMessage
                      title="Rincian tidak dapat diambil"
                      description={messageOf(detail.error)}
                      tone="gangguan"
                    />
                  ) : undefined
                }
                emptyMessage={emptyMessage}
                hideSearch
                pagination={{
                  page: detail.data?.paginasi.halaman ?? 1,
                  size: detail.data?.paginasi.ukuran ?? 50,
                  total: detail.data?.paginasi.total ?? 0,
                  totalPage: detail.data?.paginasi.total_halaman ?? 0,
                  onPageChange: setPage,
                  isLoading: detail.isFetching,
                }}
              />
              )}
            </>
          )}

        </>
      )}
    </div>
  )
}

/* ─────────────────────────── Penyaring ─────────────────────────── */

type FilterFormProps = {
  value: FilterInput
  onChange: (value: FilterInput) => void
  reportTypes: MetadataResponse['tipe_report']
  adjusters: string[]
  adjustersLoading: boolean
  violations: Record<string, string>
  onSearch: () => void
  onExport: (grid: string) => void
  exportEnabled: boolean
  exporting: boolean
}

/**
 * Penyaring tab KPI Adjuster — keempat isian yang ada di section lama, urutannya sama.
 *
 * Ia `<form>`, bukan sekumpulan input berdampingan, supaya menekan Enter di dalam isian
 * tanggal menjalankan pencarian. Pada layar yang isian terakhirnya tanggal, itu yang paling
 * sering dilakukan pengguna.
 */
function FilterForm({
  value,
  onChange,
  reportTypes,
  adjusters,
  adjustersLoading,
  violations,
  onSearch,
  onExport,
  exportEnabled,
  exporting,
}: FilterFormProps) {
  const typeOptions: SelectOption[] = reportTypes.map((item) => ({
    value: item.kode,
    label: item.judul,
  }))

  const adjusterOptions: SelectOption[] = adjusters.map((name) => ({
    value: name,
    label: name,
  }))

  return (
    <form
      className="rounded-kotak border border-slate-200 bg-white p-4"
      onSubmit={(event) => {
        event.preventDefault()
        onSearch()
      }}
    >
      {/*
        SATU baris: kelima isian, Cari, lalu Export Data — seperti bilah penyaring Pega.
        Sebelumnya isiannya digambar sebagai kisi empat kolom dengan tombolnya di baris
        tersendiri, sehingga bilahnya dua kali lebih tinggi daripada layar lama.

        `items-end` menjajarkan dasar isian dengan dasar tombol; tanpa itu tombolnya naik
        sejajar label, bukan sejajar kotak isian.
      */}
      <div className="flex flex-wrap items-end gap-3">
        {/*
          Urutannya mengikuti harness Pega persis: Status Survey, Dari, Sampai, Adjuster,
          lalu Tipe Report. Dibaca dari `Harness/ReportKPIHarness-Harness.xml` baris
          10591, 10948, 11192, 11478, dan 11732 — bukan dari tangkapan layar.
        */}
        <SelectField
          id="kpi-status-survey"
          label="Pilih Status Survey"
          options={typeOptions}
          emptyText="--Pilih--"
          value={value.tipeReport}
          error={violations['tipe_report']}
          onChange={(event) => onChange({ ...value, tipeReport: event.target.value })}
        />

        <FormField
          id="kpi-dari"
          label="Dari"
          type="date"
          className="w-44"
          value={value.dari}
          failure={violations['dari']}
          onChange={(event) => onChange({ ...value, dari: event.target.value })}
        />

        <FormField
          id="kpi-sampai"
          label="Sampai"
          type="date"
          className="w-44"
          value={value.sampai}
          failure={violations['sampai']}
          onChange={(event) => onChange({ ...value, sampai: event.target.value })}
        />

        <SelectField
          id="kpi-adjuster"
          label="Pilih Adjuster"
          options={adjusterOptions}
          // "All" apa adanya seperti layar lama (`pyNoSelectionText`), bukan terjemahan
          // "Semua adjuster" — `D-13` menetapkan teks layar mengikuti Pega.
          emptyText={adjustersLoading ? 'Memuat…' : 'All'}
          value={value.adjuster}
          error={violations['adjuster']}
          onChange={(event) => onChange({ ...value, adjuster: event.target.value })}
        />

        {/*
          Isian ini SEBELUMNYA tidak ada, dan ketiadaannya memaksa tab menggambar KEDUA
          grid sekaligus beserta dua tombol ekspor. Pega menggambar satu grid, dipilih di
          sini (`TempAdjComp.AcceptedNo`).
        */}
        <SelectField
          id="kpi-tipe-report"
          label="Pilih Tipe Report"
          options={VARIANT_OPTIONS}
          emptyText="--Pilih--"
          value={value.variant}
          error={violations['variant']}
          onChange={(event) => onChange({ ...value, variant: event.target.value })}
        />

        <Button type="submit" tone="utama">
          Cari
        </Button>

        {/*
          SATU tombol, seperti Pega — karena yang diunduh sudah dinyatakan di "Pilih Tipe
          Report". Sebelumnya ada dua tombol, dan itu akibat tidak adanya isian tersebut.
        */}
        <Button
          onClick={() => onExport(value.variant === VARIANT_DETAIL ? 'rincian' : 'ringkasan')}
          disabled={!exportEnabled || exporting}
        >
          Export Data
        </Button>
      </div>
    </form>
  )
}

/* ─────────────────────────── Penyusun kolom ─────────────────────────── */

/**
 * summaryColumns menyusun kolom grid Summary: kolom tetap, lalu kesembilan komponen.
 *
 * Keduanya datang dari server. Menuliskan kesembilan komponen di sini akan membuat urutan
 * kolom hidup di dua tempat — dan pada sembilan kolom yang isinya sama-sama angka 1–5,
 * urutan yang berselisih tidak terlihat sebagai kerusakan.
 */
function summaryColumns(
  grid: Grid | undefined,
  components: Component[],
  reportType: string,
): Column<SummaryRow>[] {
  // `render` dipasang pada SELURUH kolom tetap, dan itu bukan hiasan.
  //
  // Tanpa `render`, DataTable menggambar sel kosong sebagai tanda hubung. Pada kolom yang
  // memang TIDAK PERNAH punya sumber — `Status` di tipe tunggal, `NO KLAIM`, dan
  // `STATUS SURVEY` — tanda hubung itu terbaca sebagai "nilainya hilang", padahal yang
  // benar adalah "kolom ini memang kosong di Pega juga".
  //
  // Kesembilan kolom nilai TIDAK diperlakukan begitu: di sana tanda hubung justru bermakna,
  // karena ia membedakan komponen yang belum dinilai dari komponen bernilai nol.
  const fixed: Column<SummaryRow>[] = columnsFor(grid, reportType).map((column) => ({
    key: column.kunci,
    title: column.judul,
    value: (row) => summaryCell(row, column.kunci),
    render: (row) => summaryCell(row, column.kunci),
  }))

  // KATEGORI digambar SESUDAH kesembilan komponen — itulah tempatnya di grid layar lama.
  const trailing: Column<SummaryRow>[] = (grid?.kolom_akhir ?? []).map((column) => ({
    key: column.kunci,
    title: column.judul,
    value: (row) => summaryCell(row, column.kunci),
    render: (row) => summaryCell(row, column.kunci),
  }))

  return [...fixed, ...scoreColumns<SummaryRow>(components), ...trailing]
}

/** detailColumns menyusun kolom grid Detail dengan cara yang sama. */
function detailColumns(
  grid: Grid | undefined,
  components: Component[],
  reportType: string,
): Column<DetailRow>[] {
  const fixed: Column<DetailRow>[] = columnsFor(grid, reportType).map((column) => ({
    key: column.kunci,
    title: column.judul,
    value: (row) => detailCell(row, column.kunci),
    render: (row) => detailCell(row, column.kunci),
  }))

  const trailing: Column<DetailRow>[] = (grid?.kolom_akhir ?? []).map((column) => ({
    key: column.kunci,
    title: column.judul,
    value: (row) => detailCell(row, column.kunci),
    render: (row) => detailCell(row, column.kunci),
  }))

  return [...fixed, ...scoreColumns<DetailRow>(components), ...trailing]
}

/**
 * columnsFor membuang kolom yang tidak berlaku pada tipe report yang sedang ditampilkan.
 *
 * Satu-satunya hari ini: kolom TIPE pada grid Summary, yang di Pega hanya dikembalikan
 * kueri penggabung (`GetSummaryKPIAdjusterALL`). Penandanya datang dari server, bukan
 * ditebak dari nama kolom — berkas ekspor disaring dengan penanda yang sama di sana,
 * sehingga isi berkas dan isi layar tidak dapat berselisih.
 */
function columnsFor(grid: Grid | undefined, reportType: string) {
  return (grid?.kolom ?? []).filter(
    (column) => !column.hanya_tipe_gabungan || reportType === 'ALL',
  )
}

/**
 * scoreColumns menyusun kesembilan kolom nilai.
 *
 * Nilai kosong digambar sebagai tanda hubung, BUKAN sebagai 0. Keduanya berbeda artinya
 * pada laporan penilaian kinerja: yang pertama berarti komponennya belum dinilai, yang
 * kedua berarti adjuster tidak mendapat poin.
 */
function scoreColumns<T extends { nilai: Scores }>(components: Component[]): Column<T>[] {
  return components.map((component) => ({
    key: component.kode,
    title: component.judul,
    value: (row) => scoreText(row.nilai[component.kode]),

    // Angka dirapatkan ke kanan supaya koma desimalnya sejajar antar baris. Pada sembilan
    // kolom angka berdampingan, itu yang membuat kolomnya masih dapat dibaca sekilas.
    alignRight: true,
  }))
}

/** summaryCell mengambil isi satu sel tetap grid Summary. */
function summaryCell(row: SummaryRow, key: string): string {
  switch (key) {
    case 'adjuster':
      return row.adjuster
    case 'tipe':
      return row.tipe
    case 'kategori':
      return row.kategori
    default:
      return ''
  }
}

/**
 * detailCell mengambil isi satu sel tetap grid Detail.
 *
 * Dua kunci sengaja TIDAK punya cabang — `no_klaim` dan `status_survey`. Keduanya ada di
 * grid layar lama tetapi tidak ada di tabel sumbernya, sehingga kosong di Pega juga; ia
 * jatuh ke `default` dan digambar kosong. Lihat `FieldClaimNumberAdjuster` di screen.go.
 */
function detailCell(row: DetailRow, key: string): string {
  switch (key) {
    case 'adjuster':
      return row.adjuster
    case 'no_case':
      return row.no_case
    case 'kategori':
      return row.kategori
    default:
      return ''
  }
}

/** scoreText menggambar satu nilai komponen. */
function scoreText(value: number | null | undefined): string {
  if (value === null || value === undefined) return '—'
  return String(value)
}

/* ─────────────────────────── Bantuan kecil ─────────────────────────── */

const emptyFilter: FilterInput = {
  tipeReport: '',
  variant: '',
  adjuster: '',
  dari: '',
  sampai: '',
}

/**
 * Pilihan isian "Pilih Tipe Report" — nilainya diambil APA ADANYA dari Pega.
 *
 * Keduanya terbaca sebagai perbandingan di export: `TempAdjComp.AcceptedNo=="DATA SUMMARY"`
 * dan `=="DATA DETAIL"`. Nilainya ditulis persis begitu — berspasi dan berhuruf besar —
 * karena ia dibandingkan sebagai teks.
 */
const VARIANT_SUMMARY = 'DATA SUMMARY'
const VARIANT_DETAIL = 'DATA DETAIL'

const VARIANT_OPTIONS = [
  { value: VARIANT_SUMMARY, label: VARIANT_SUMMARY },
  { value: VARIANT_DETAIL, label: VARIANT_DETAIL },
]

const emptyMessage =
  'Tidak ada penilaian pada penyaring ini. Bila Anda yakin seharusnya ada, ingat bahwa ' +
  'layar ini hanya MEMBACA: penilaian baru muncul setelah dihitung dari Pega. Periksa ' +
  'juga periodenya — yang disaring adalah tanggal penilaian, bukan tanggal kejadian.'

/** gridOf mengambil keterangan satu grid pada tab KPI Adjuster. */
function gridOf(metadata: MetadataResponse | undefined, code: string): Grid | undefined {
  const tab = metadata?.tab.find((item) => item.kode === 'adjuster')
  return tab?.grid.find((item) => item.kode === code)
}

/** violationsOf mengambil pelanggaran per isian dari sebuah galat, bila ada. */
function violationsOf(error: unknown): Record<string, string> {
  if (error instanceof APIError) return error.violations()
  return {}
}

/** messageOf mengambil pesan yang layak dibaca pengguna dari sebuah galat. */
function messageOf(error: unknown): string {
  if (error instanceof APIError) return error.message
  if (error instanceof Error && error.message !== '') return error.message
  return 'Coba lagi beberapa saat lagi. Bila terus berulang, hubungi tim teknis.'
}
