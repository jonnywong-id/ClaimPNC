import { useState, type ReactNode } from 'react'
import { useNavigate } from 'react-router-dom'

import { APIError } from '@/api/client'
import { useSelectedPortal } from '@/app/portal'
import { Button } from '@/components/Button'
import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage } from '@/components/ErrorMessage'
import { formatDate } from '@/components/format'

import { ManagerAdminTabs } from './ManagerAdminTabs'
import {
  useExportInboxManagerAdmin,
  useInboxManagerAdminList,
  useInboxManagerAdminMetadata,
} from './api'
import type { MetadataResponse, Tab, TabColumn, WorkItem } from './types'

/**
 * Inbox Manager Admin — menu `MENU_ID 57`, pengganti harness `InboxManagerAdmin_Harness`.
 *
 * Isinya pandangan PENYELIA atas antrean registrasi klaim, dipecah menurut unit organisasi
 * admin:
 *
 *   Manajemen Admin - Non MBU    AdminPNC
 *   Manajemen Admin - PA         AdminPA
 *   Manajemen Admin - Travel     AdminTRAVEL
 *
 * Per `D-79` ia benar-benar Inbox: barisnya diambil dari tabel penugasan Pega dan hilang
 * begitu klaimnya selesai. Karena itu modul ini milik `U-3`, bukan `U-6`.
 *
 * # Susunan layar, dan dari mana bentuknya
 *
 * Diambil dari `Section/InboxManagerAdmin_Section-Section.xml`: judul "My Inbox", lalu tiga
 * kontainer grid berhalaman 50 baris. Nama kolom TIDAK diterjemahkan — `D-13` menetapkan
 * tampilan meniru Pega, dan itulah teks yang selama ini dibaca pengguna.
 *
 * Yang TIDAK ada di sana dan ditambahkan di sini: bilah tab. Ketiga grid di Pega adalah
 * kontainer terpisah yang tampil bersamaan bagi pengguna yang berhak atas lebih dari satu.
 *
 * # Layar ini dapat sah-sah saja TIDAK punya satu tab pun
 *
 * Dan itu bukan kerusakan. Visibilitas tab mengikuti jabatan pengguna persis seperti di Pega
 * (keputusan Work Owner 2026-09-26), sementara jabatan di sistem baru datang dari HCQ dan
 * berisi hal seperti "IT SPECIALIST" — bukan kode lini bisnis. Sampai `TKT-F3-004` selesai,
 * sebagian besar pengguna memang tidak akan melihat tab apa pun.
 *
 * Layar menjelaskan keadaan itu dengan menyebut jabatan yang terbaca sistem beserta ketiga
 * jabatan yang membuka tab — bukan menampilkan layar kosong tanpa sebab.
 */
export function InboxManagerAdminPage() {
  const [tabCode, setTabCode] = useState('')
  const [page, setPage] = useState(1)

  const portal = useSelectedPortal((state) => state.alias)
  const meta = useInboxManagerAdminMetadata()

  const tabs: Tab[] = meta.data?.tab ?? []
  const active = tabCode || meta.data?.tab_bawaan || ''
  const tab = tabs.find((candidate) => candidate.kode === active)

  // Tanpa tab yang boleh dilihat, isinya TIDAK diminta. Permintaannya pasti ditolak server
  // dengan 403, dan galat yang sudah diketahui pasti terjadi bukan galat yang layak
  // ditampilkan — yang layak ditampilkan adalah penjelasannya.
  const hasTab = tabs.length > 0
  const list = useInboxManagerAdminList(active, page, meta.isSuccess && hasTab)

  /**
   * Berpindah tab mengembalikan ke halaman pertama.
   *
   * Tanpa itu, berpindah dari halaman 4 sebuah tab ke tab yang hanya punya 2 halaman akan
   * menampilkan tabel kosong yang terbaca seperti antrean yang memang kosong.
   */
  function selectTab(code: string) {
    setTabCode(code)
    setPage(1)
  }

  if (portal === null) {
    return (
      <PageFrame tab={active} exportable={false}>
        <ErrorMessage
          title="Pilih entitas lebih dulu"
          description={
            'Antrean registrasi klaim milik satu badan hukum, dan aplikasi ini melayani ' +
            'empat. Pilih portal di bilah atas untuk membukanya.'
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
      exportable={hasTab && (list.data?.paginasi.total ?? 0) > 0}
    >
      {meta.isSuccess && !hasTab ? (
        <NoTabNotice meta={meta.data} />
      ) : (
        <>
          <div className="mt-4">
            <ManagerAdminTabs tabs={tabs} active={active} onSelect={selectTab} />
          </div>

          {tab && (
            <>
              <p className="mt-3 text-sm text-slate-600">{tab.keterangan}</p>

              <div className="mt-4">
                <DataTable<WorkItem>
                  columns={columnsFor(tab, (row) => <DetailButton item={row} />)}
                  rows={list.data?.baris ?? []}
                  rowKey={(row) => `${row.referensi}|${row.id}`}
                  title={tab.nama}
                  label={`Antrean ${tab.nama}`}
                  // Kotak cari bawaan disembunyikan: hasilnya akan menyaring HANYA halaman
                  // yang sedang terbuka, sehingga pengguna dapat diberi tahu "tidak ada"
                  // untuk baris yang sebenarnya ada di halaman berikutnya.
                  //
                  // Layar lama pun tidak punya kotak cari: Report Definition-nya tidak
                  // menyaring menurut kata kunci sama sekali.
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
            </>
          )}
        </>
      )}

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
          <h1 className="text-xl font-semibold text-slate-900">Inbox Manager Admin</h1>
          <p className="mt-1 text-sm text-slate-600">
            Pandangan penyelia atas antrean registrasi klaim, dipisah menurut unit
            organisasi admin.
          </p>
          {/*
            Sifat "pandangan penyelia" dinyatakan di layar, bukan hanya di kode.

            Tidak satu pun tab di sini menyaring menurut pengguna yang login — berbeda dari
            sebagian layar inbox lain, yang punya tab "milik saya". Tanpa keterangan ini,
            petugas yang terbiasa dengan layar itu akan mengira daftarnya keliru karena
            memuat pekerjaan orang lain.
          */}
          <p className="mt-2 text-xs text-slate-500">
            Layar ini menampilkan pekerjaan{' '}
            <span className="font-medium">seluruh petugas</span> pada unit organisasi yang
            dipilih, bukan hanya milik Anda. Setiap pembukaannya tercatat.
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
 * Judulnya "Export To Excel" persis seperti `pyButtonLabel` di section (`D-13`), meski
 * berkasnya CSV — begitu pula sistem lama, yang memanggil `pxConvertResultsToCSV`.
 *
 * Tombolnya dimatikan saat tidak ada yang dapat diekspor. Berkas kosong yang tetap terunduh
 * adalah jawaban yang membingungkan: pengguna tidak dapat membedakannya dari ekspor yang
 * gagal diam-diam.
 */
function ExportButton({ tab, enabled }: { tab: string; enabled: boolean }) {
  const ekspor = useExportInboxManagerAdmin()

  return (
    <div className="flex flex-col items-end gap-1">
      <Button
        tone="kedua"
        disabled={!enabled || ekspor.isPending}
        onClick={() => ekspor.mutate(tab)}
      >
        {ekspor.isPending ? 'Menyiapkan berkas…' : 'Export To Excel'}
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
 * Keterangan saat pengguna tidak berhak atas satu tab pun.
 *
 * # Kenapa panjang, dan kenapa menyebut nilainya
 *
 * Karena pengguna yang menemuinya tidak punya cara lain mengetahui sebabnya. Tiga hal yang
 * disebut, dan ketiganya berguna saat ia melapor:
 *
 *   - lini bisnis yang terbaca sistem — kosong berarti datanya belum diisi;
 *   - ketiga lini bisnis yang membuka tab;
 *   - ketiga antrean yang ada, supaya ia tahu apa yang sedang tidak ia lihat.
 *
 * # Kenapa membedakan "kosong" dari "terisi tetapi lain"
 *
 * Karena keduanya menuntut tindakan yang berbeda, dan hanya pengguna yang dapat
 * menyampaikannya ke orang yang tepat. Kosong berarti barisnya di master pengguna belum
 * dilengkapi; terisi nilai lain berarti nilainya perlu dibetulkan.
 *
 * Isinya datang dari SERVER, bukan ditulis tetap di sini, supaya ia ikut berubah saat
 * aturannya berubah.
 */
function NoTabNotice({ meta }: { meta: MetadataResponse }) {
  const belumDiisi = meta.lini_bisnis_anda.trim() === ''

  return (
    <section className="mt-6 rounded-kartu border border-amber-200 bg-amber-50 px-4 py-4">
      <h2 className="text-sm font-semibold text-amber-900">
        Tidak ada antrean yang menjadi hak lini bisnis Anda
      </h2>

      <p className="mt-2 text-sm text-slate-700">
        Layar ini memisahkan antrean menurut unit organisasi admin, dan setiap lini bisnis
        hanya membuka unitnya sendiri — sama seperti di sistem lama.
      </p>

      <dl className="mt-3 space-y-1 text-sm text-slate-700">
        <div className="flex flex-wrap gap-x-2">
          <dt className="font-medium">Lini bisnis Anda terbaca sebagai:</dt>
          <dd>
            {belumDiisi ? (
              <span className="text-slate-500">(belum diisi)</span>
            ) : (
              meta.lini_bisnis_anda
            )}
          </dd>
        </div>
        <div className="flex flex-wrap gap-x-2">
          <dt className="font-medium">Lini bisnis yang membuka antrean:</dt>
          <dd>{meta.lini_bisnis_yang_diharapkan.join(', ')}</dd>
        </div>
      </dl>

      {meta.semua_tab.length > 0 && (
        <>
          <p className="mt-3 text-xs font-medium text-slate-700">
            Antrean yang ada di layar ini:
          </p>
          <ul className="mt-1 list-disc space-y-0.5 pl-5 text-xs text-slate-600">
            {meta.semua_tab.map((tab) => (
              <li key={tab.kode}>
                {tab.nama} — dibuka lini bisnis{' '}
                <span className="font-medium">{tab.lini_bisnis}</span>
              </li>
            ))}
          </ul>
        </>
      )}

      <p className="mt-3 text-xs text-slate-600">
        {belumDiisi
          ? 'Lini bisnis Anda belum diisi pada master pengguna. Laporkan ke tim teknis — ' +
            'melengkapinya tidak menuntut perubahan aplikasi, dan antrean Anda akan ' +
            'langsung terlihat sesudahnya.'
          : 'Bila Anda seharusnya termasuk salah satunya, laporkan ke tim teknis beserta ' +
            'nilai yang terbaca di atas — lini bisnis disimpan pada master pengguna dan ' +
            'dapat dibetulkan tanpa perubahan aplikasi.'}
      </p>
    </section>
  )
}

/**
 * Tombol rincian.
 *
 * Layar tujuannya adalah `MENU_ID 75` "View Claim" (`PNCViewClaim`) — modul tersendiri yang
 * belum dibangun. Tombolnya tetap dibangun mengikuti modul inbox lain, dan tujuannya
 * diarahkan ke rute yang sudah ada tempat keadaan itu dinyatakan apa adanya.
 *
 * Yang dikirim adalah `referensi`, kunci teknis Pega — nilai yang sama yang di layar lama
 * disusun menjadi kunci assignment oleh `SetAssignmentInboxReg_act`. Dengan begitu
 * menyalakan layar rincian kelak tidak menuntut perubahan kontrak API modul ini.
 */
function DetailButton({ item }: { item: WorkItem }) {
  const navigate = useNavigate()
  const key = item.referensi || item.id

  return (
    <Button
      tone="halus"
      disabled={key === ''}
      onClick={() => navigate(`/view-claim/${encodeURIComponent(key)}`)}
    >
      Lihat Detail
    </Button>
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
 * Dua perlakuan, dan masing-masing punya alasannya:
 *
 *  - Tanggal diformat HANYA bila bentuknya memang `YYYY-MM-DD`. Kolom Lama Waktu Klaim
 *    berisi kalimat ("1 year 5 months ago"), bukan tanggal, dan memaksanya lewat pemformat
 *    tanggal akan mengubahnya menjadi teks yang salah.
 *  - Teks kosong menjadi tanda pisah, bukan sel kosong yang tidak dapat dibedakan dari
 *    kolom yang gagal dimuat. Dua kolom di layar ini memang dapat kosong: Nama Sumber
 *    Bisnis pada klaim yang snapshot polisnya belum lengkap, dan Lama Waktu Klaim pada
 *    baris yang tanggal pendaftarannya belum ada.
 */
function cellText(row: WorkItem, column: TabColumn): string {
  const value = row[column.kunci]

  if (value == null || value === '') return '—'

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
 * "Tidak ada data" tidak cukup di layar ini, dan alasannya khusus: penyaring ketiga tab
 * adalah kolom `PXASSIGNEDORGUNIT` pada `POOLDATA.T_CLAIMLIST_ADMIN` — kolom yang diminta
 * ditambahkan saat sumber layar ini dipindahkan ke tabel itu (Work Owner 2026-09-27), dan
 * yang pengisiannya bergantung pada proses di luar aplikasi ini. Kolom yang ada tetapi
 * tidak pernah terisi mengembalikan nol baris TANPA satu pun galat — dan keterangan inilah
 * satu-satunya yang mengarahkan pengguna melapor alih-alih menganggapnya wajar.
 */
function emptyMessageFor(tab: Tab): string {
  return (
    `Tidak ada klaim yang menunggu pada unit organisasi ${tab.unit_organisasi}. ` +
    'Bila Anda yakin seharusnya ada, laporkan ke tim teknis — antrean ini disaring ' +
    'menurut unit organisasi penugasan, dan penugasan yang unitnya kosong tidak muncul ' +
    'di tab mana pun tanpa pesan galat.'
  )
}

/** messageOf mengambil pesan yang layak dibaca pengguna dari sebuah galat. */
function messageOf(error: unknown): string {
  if (error instanceof APIError) return error.message
  if (error instanceof Error && error.message !== '') return error.message
  return 'Coba lagi beberapa saat lagi. Bila terus berulang, hubungi tim teknis.'
}
