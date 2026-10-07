import { useMemo, useState, type ReactNode } from 'react'

import { APIError } from '@/api/client'
import { Button } from '@/components/Button'
import { ErrorMessage } from '@/components/ErrorMessage'

import { DashboardPanels } from './DashboardPanels'
import { ManagerTabs } from './ManagerTabs'
import { QueuePanel } from './QueuePanel'
import {
  useExportInboxManager,
  useInboxManagerCounters,
  useInboxManagerDecide,
  useInboxManagerList,
  useInboxManagerMetadata,
} from './api'
import type { Counter, PeriodInput, Tab, Verdict } from './types'

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

  const metadata = useInboxManagerMetadata()
  const counters = useInboxManagerCounters()

  const tabs = metadata.data?.tab ?? []
  const activeCode = tabCode || metadata.data?.tab_bawaan || ''
  const activeTab = tabs.find((tab) => tab.kode === activeCode)

  const list = useInboxManagerList(
    activeCode,
    page,
    activeTab?.punya_penyaring_periode ? period : null,
    Boolean(activeCode),
  )

  const decide = useInboxManagerDecide()
  const exportQueue = useExportInboxManager()

  const counterList = useMemo<Counter[]>(
    () => counters.data?.pencacah ?? [],
    [counters.data],
  )

  function selectTab(code: string) {
    setTabCode(code)
    setPage(1)
    setResultMessage('')
    decide.reset()
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

      <ManagerTabs
        tabs={tabs}
        counters={counterList}
        active={activeCode}
        onSelect={selectTab}
      />

      {counters.isError ? (
        <ErrorMessage
          tone="gangguan"
          title="Angka di bilah tab tidak dapat dimuat"
          description={messageOf(
            counters.error,
            'Pencacah tidak dapat diambil. Isi tiap bagian tetap dapat dibuka.',
          )}
        />
      ) : null}

      {activeTab ? (
        <TabBody
          tab={activeTab}
          counters={counterList}
          list={list}
          page={page}
          onPageChange={setPage}
          period={period}
          onPeriodChange={(next) => {
            setPeriod(next)
            setPage(1)
          }}
          onSelectTab={selectTab}
          onDecide={submitDecision}
          isDeciding={decide.isPending}
          resultMessage={resultMessage}
          decideError={decide.isError ? messageOf(decide.error, 'Keputusan gagal.') : undefined}
        />
      ) : null}

      <PlannedDifferences items={metadata.data?.selisih_terencana ?? []} />
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
}: Readonly<{
  exportable: boolean
  isExporting: boolean
  onExport: () => void
  children: ReactNode
}>) {
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
          {exportable ? null : (
            <span className="text-xs text-slate-500">
              Hanya antrean persetujuan yang dapat diekspor.
            </span>
          )}
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
function LineBusinessNote({ line, loading }: Readonly<{ line: string; loading: boolean }>) {
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
  counters: Counter[]
  list: ReturnType<typeof useInboxManagerList>
  page: number
  onPageChange: (page: number) => void
  period: PeriodInput | null
  onPeriodChange: (period: PeriodInput | null) => void
  onSelectTab: (code: string) => void
  onDecide: (verdict: Verdict, keys: string[], reason: string) => void
  isDeciding: boolean
  resultMessage: string
  decideError?: string | undefined
}

/** Isi tab yang sedang terbuka, digambar menurut JENISNYA. */
function TabBody(props: Readonly<TabBodyProps>) {
  const { tab, list } = props

  if (list.isError) {
    return (
      <ErrorMessage
        tone="gangguan"
        title={`"${tab.nama}" tidak dapat dimuat`}
        description={messageOf(list.error, 'Isi bagian ini tidak dapat diambil.')}
      />
    )
  }

  if (tab.jenis === 'ringkasan') {
    return <OverviewPanel counters={props.counters} onSelectTab={props.onSelectTab} />
  }

  if (tab.jenis === 'dashboard') {
    return (
      <div className="space-y-4">
        {tab.punya_penyaring_periode ? (
          <PeriodFilter
            tab={tab}
            value={props.period}
            onChange={props.onPeriodChange}
            applied={list.data?.periode}
          />
        ) : null}

        <DashboardPanels
          panels={list.data?.panel ?? []}
          isLoading={list.isLoading}
          refreshedAt={list.data?.disegarkan_pada}
        />
      </div>
    )
  }

  return (
    <div className="space-y-3">
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
 * Tab "Approval Master" — indeks kesembilan antrean di bawahnya.
 *
 * Di Pega ia `Section/InboxManager_Section2`, yang menggambar pintasan ke tiap antrean. Di
 * sini pintasan itu menjadi kartu berangka, dan angkanya datang dari pencacah yang sama dengan
 * bilah tab — bukan dihitung ulang.
 */
function OverviewPanel({
  counters,
  onSelectTab,
}: Readonly<{
  counters: Counter[]
  onSelectTab: (code: string) => void
}>) {
  const children = counters.filter((counter) => counter.induk)

  if (children.length === 0) {
    return (
      <p className="text-sm text-slate-600">
        Angka antrean belum dapat dimuat. Bagian di bilah tab tetap dapat dibuka satu per satu.
      </p>
    )
  }

  return (
    <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
      {children.map((counter) => (
        <button
          key={counter.tab}
          type="button"
          onClick={() => onSelectTab(counter.tab)}
          className="rounded-kontrol border border-slate-200 bg-white p-4 text-left transition hover:border-blue-300 hover:bg-blue-50/40 focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500/50"
        >
          <p className="text-sm font-medium text-slate-800">{counter.label}</p>

          {counter.tidak_tersedia ? (
            <p className="mt-1 text-sm text-amber-800">{counter.tidak_tersedia}</p>
          ) : (
            <p className="mt-1 text-2xl font-semibold tabular-nums text-slate-900">
              {counter.jumlah}
              <span className="ml-1 text-sm font-normal text-slate-500">menunggu</span>
            </p>
          )}
        </button>
      ))}
    </div>
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
}: Readonly<{
  tab: Tab
  value: PeriodInput | null
  onChange: (period: PeriodInput | null) => void
  applied?: { dari: string; sampai: string } | undefined
}>) {
  const mode = value?.bentuk ?? 'bulan'

  function update(next: Partial<PeriodInput>) {
    onChange({
      bentuk: mode,
      bulan: value?.bulan ?? '',
      dari: value?.dari ?? '',
      sampai: value?.sampai ?? '',
      ...next,
    })
  }

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
              value={value?.bulan ?? ''}
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
                value={value?.dari ?? ''}
                onChange={(event) => update({ dari: event.target.value })}
                className="rounded-kontrol border border-slate-300 px-3 py-2 text-sm"
              />
            </label>
            <label className="text-sm">
              <span className="mb-1 block font-medium text-slate-700">Sampai</span>
              <input
                type="date"
                value={value?.sampai ?? ''}
                onChange={(event) => update({ sampai: event.target.value })}
                className="rounded-kontrol border border-slate-300 px-3 py-2 text-sm"
              />
            </label>
          </>
        )}

        {value ? (
          <Button type="button" tone="halus" onClick={() => onChange(null)}>
            Hapus penyaring
          </Button>
        ) : null}
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
function PlannedDifferences({ items }: Readonly<{ items: string[] }>) {
  if (items.length === 0) return null

  return (
    <details className="rounded-kontrol border border-slate-200 bg-slate-50 p-4">
      <summary className="cursor-pointer text-sm font-medium text-slate-700">
        Yang berbeda dari layar lama ({items.length})
      </summary>
      <ul className="mt-3 list-disc space-y-2 pl-5 text-sm text-slate-600">
        {items.map((item) => (
          <li key={item}>{item}</li>
        ))}
      </ul>
    </details>
  )
}

/** messageOf membaca pesan galat yang layak dibaca pengguna. */
function messageOf(error: unknown, fallback: string): string {
  if (error instanceof APIError && error.message) return error.message
  if (error instanceof Error && error.message) return error.message
  return fallback
}
