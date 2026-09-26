import { useState, type FormEvent, type ReactNode } from 'react'
import { useNavigate } from 'react-router-dom'

import { APIError } from '@/api/client'
import { useSelectedPortal } from '@/app/portal'
import { Button } from '@/components/Button'
import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage } from '@/components/ErrorMessage'
import { formatDate } from '@/components/format'

import { ComplianceTabs } from './ComplianceTabs'
import {
  useInboxComplianceList,
  useInboxComplianceMetadata,
  useSendToPostAudit,
} from './api'
import type { PageInfo, Tab, TabColumn, WorkItem } from './types'

/**
 * Inbox Compliance — menu `MENU_ID 47`, pengganti harness `inboxCompliance_Harness`.
 *
 * Isinya antrean pemeriksaan kepatuhan. Per `D-79` ia benar-benar Inbox: barisnya pekerjaan
 * yang menunggu di workbasket `CompliancePNC`, hilang begitu klaimnya selesai, dan punya
 * tenggat berupa kolom Aging.
 *
 * # Susunan layar, dan dari mana bentuknya
 *
 * Diambil dari `Section/InputCompliance_Section-Section.xml` apa adanya: judul, lalu dua
 * tab, lalu grid. Nama kolom TIDAK diterjemahkan — `D-13` menetapkan tampilan meniru Pega,
 * dan itulah teks yang selama ini dibaca pengguna.
 *
 * # Kenapa TIDAK ada kotak cari maupun penyaring
 *
 * Karena sistem lama pun tidak punya. `InputCompliance_Section` tidak memuat satu pun field
 * masukan, dropdown, atau tombol — hanya dua kontainer tab.
 *
 * Ini mudah salah duga, sehingga perlu dicatat: `GetInboxRegisterCompliance-Act` memang
 * memuat parameter bernama `CARI1` dan `CARI2`, dan nama itu di layar lain berarti kotak
 * cari. Di sini keduanya adalah dua argumen TANGGAL untuk `GETSELISIHJAM` — `"SYSDATE"` dan
 * `TanggalBuatCompliance + 7 jam`. Menambahkan kotak cari berarti mengarang kemampuan yang
 * tidak pernah ada.
 *
 * # Kenapa kolomnya datang dari server
 *
 * Karena kedua tab punya kolom yang berbeda, dan daftar itu adalah hasil pembacaan export
 * Pega yang tercatat di `internal/inboxcompliance/tab.go`. Menyalinnya ke sini berarti
 * daftar yang sama hidup di dua tempat.
 */
export function InboxCompliancePage() {
  const [activeTab, setActiveTab] = useState('')
  const [page, setPage] = useState(1)

  // Klaim yang sedang dikirim ke Post Audit, atau null bila formnya tertutup.
  const [sending, setSending] = useState<WorkItem | null>(null)

  const portal = useSelectedPortal((state) => state.alias)
  const meta = useInboxComplianceMetadata()

  const tabs: Tab[] = meta.data?.tab ?? []
  const active = activeTab || meta.data?.tab_bawaan || ''
  const tab = tabs.find((candidate) => candidate.kode === active)

  // Tab yang belum dapat dilayani tidak dipanggil ke server. Memanggilnya tetap aman —
  // server menjawab 503 beserta penjelasannya — tetapi memanggil sesuatu yang sudah pasti
  // gagal hanya menambah galat di log tanpa menambah keterangan apa pun bagi pengguna.
  const list = useInboxComplianceList(active, page, meta.isSuccess && tab?.tersedia === true)

  function selectTab(code: string) {
    setActiveTab(code)
    setPage(1)
  }

  if (portal === null) {
    return (
      <PageFrame>
        <ErrorMessage
          title="Pilih entitas lebih dulu"
          description={
            'Antrean kepatuhan milik satu badan hukum, dan aplikasi ini melayani empat. ' +
            'Pilih portal di bilah atas untuk membukanya.'
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
        <ComplianceTabs tabs={tabs} active={active} onSelect={selectTab} />
      </div>

      {tab && (
        <>
          <p className="mt-3 text-sm text-slate-600">{tab.keterangan}</p>

          {tab.tersedia ? (
            <div className="mt-4">
              <DataTable<WorkItem>
                columns={columnsFor(tab, (row) => (
                  <RowActions
                    item={row}
                    tab={tab}
                    onSend={() => setSending(row)}
                  />
                ))}
                rows={list.data?.baris ?? []}
                rowKey={(row) => `${row.referensi}|${row.nomor_case}`}
                title={tab.nama}
                // Kotak cari bawaan disembunyikan: sistem lama tidak punya pencarian di
                // layar ini, dan kotak bawaan hanya akan menyaring halaman yang sedang
                // terbuka — hasilnya menyesatkan pada data berhalaman.
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
                emptyMessage="Tidak ada klaim yang menunggu pemeriksaan kepatuhan."
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
          ) : (
            <PendingTab tab={tab} />
          )}
        </>
      )}

      <Notes limitations={meta.data?.keterbatasan ?? []} />

      {sending && (
        <SendPostAuditDialog claim={sending} onClose={() => setSending(null)} />
      )}
    </PageFrame>
  )
}

function PageFrame({ children }: { children: ReactNode }) {
  return (
    <div className="mx-auto max-w-[96rem] px-4 py-8">
      <header className="border-b border-slate-200 pb-4">
        <h1 className="text-xl font-semibold text-slate-900">Inbox Compliance</h1>
        <p className="mt-1 text-sm text-slate-600">
          Antrean pemeriksaan kepatuhan: klaim yang menunggu diperiksa, dan pemeriksaan Post
          Audit yang belum ditindaklanjuti.
        </p>
      </header>
      {children}
    </div>
  )
}

/**
 * Panel pengganti tabel pada tab yang belum dapat dilayani.
 *
 * Ia menyebut apa yang kurang dan siapa pemiliknya, bukan sekadar "belum tersedia".
 * Kalimatnya datang dari SERVER supaya hilang dengan sendirinya begitu penghalangnya
 * hilang — tanpa satu baris pun di berkas ini disunting.
 *
 * Nadanya "gangguan", bukan "galat": tidak ada yang rusak dan tidak ada yang perlu
 * diperbaiki pengguna. Yang ada adalah artefak yang sedang ditunggu.
 */
function PendingTab({ tab }: { tab: Tab }) {
  return (
    <div className="mt-4">
      <ErrorMessage
        title={`Tab ${tab.nama} belum menampilkan data`}
        description={tab.penghalang ?? 'Sumber datanya belum tersedia.'}
        tone="gangguan"
      />
    </div>
  )
}

/**
 * Tombol buka detail klaim.
 *
 * Layar tujuannya adalah `MENU_ID 75` "View Claim" (`PNCViewClaim`) — modul tersendiri yang
 * belum dibangun. Rutenya disamakan dengan yang dipakai Inbox Admin supaya kedua layar
 * antrean kerja menuju tempat yang sama.
 *
 * Yang dikirim adalah `referensi`, kunci teknis Pega. Dengan begitu menyalakan layar
 * rincian kelak tidak menuntut perubahan kontrak API modul ini.
 */
function DetailButton({ item }: { item: WorkItem }) {
  const navigate = useNavigate()
  const key = item.referensi || item.nomor_case

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
 * Bentuknya sama dengan Inbox Admin, Pelaporan Klaim, dan View History Claim supaya
 * keempatnya tidak terasa dirakit dari empat aplikasi berbeda. Ia hidup di sini, bukan di
 * dalam `DataTable`, karena komponen tabel baku belum mengenal paginasi server — itu
 * lingkup `TKT-U2-001`.
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
 * Catatan di bawah tabel: keterbatasan yang berlaku.
 *
 * Datang dari SERVER, bukan ditulis tetap di sini, supaya hilang dengan sendirinya begitu
 * penghalangnya hilang. Tanpa catatan ini, Aging yang berbeda dari angka TAT pada laporan
 * KPI akan dilaporkan berulang kali sebagai kerusakan — padahal keduanya memang memakai
 * dasar hitungan yang berbeda sejak di sistem lama.
 */
function Notes({ limitations }: { limitations: string[] }) {
  if (limitations.length === 0) return null

  return (
    <section className="mt-6 rounded-kartu border border-slate-200 bg-slate-50 px-4 py-3">
      <h2 className="text-sm font-medium text-slate-800">Yang perlu diketahui</h2>
      <ul className="mt-2 list-disc space-y-1 pl-5 text-xs text-slate-600">
        {limitations.map((line) => (
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
/**
 * Kontrol di ujung setiap baris.
 *
 * # Kenapa tombol Kirim hanya ada di tab Compliance
 *
 * Karena hanya di sana pengirimannya punya arti: barisnya adalah klaim yang SEDANG menunggu
 * diperiksa. Baris tab Post Audit sudah terkirim, dan tidak ada yang dapat dilakukan
 * atasnya dari layar ini.
 *
 * Tabnya diperiksa lewat kodenya, bukan lewat ada-tidaknya isian tertentu pada barisnya.
 * Isian yang kebetulan kosong akan membuat tombolnya hilang pada baris yang seharusnya
 * punya.
 */
function RowActions({
  item,
  tab,
  onSend,
}: {
  item: WorkItem
  tab: Tab
  onSend: () => void
}) {
  return (
    <div className="flex justify-end gap-2">
      {tab.kode === 'compliance' && (
        <Button tone="kedua" onClick={onSend} disabled={item.referensi === ''}>
          Kirim ke Post Audit
        </Button>
      )}
      <DetailButton item={item} />
    </div>
  )
}

/**
 * Form pengiriman satu klaim ke Post Audit.
 *
 * # Kenapa hanya satu isian
 *
 * Karena hanya satu yang benar-benar diketik. Nomor polis dan nama tertanggung diambil
 * server dari klaimnya sendiri — mengetiknya ulang membuka kemungkinan baris Post Audit
 * menyebut nama yang berbeda dari klaim yang dirujuknya, dan tabelnya tidak punya foreign
 * key yang akan menolaknya.
 *
 * Tanggal pun tidak diminta: layar Pega menampilkan waktu PENGIRIMAN, bukan tanggal yang
 * dipilih.
 */
function SendPostAuditDialog({
  claim,
  onClose,
}: {
  claim: WorkItem
  onClose: () => void
}) {
  const [remarks, setRemarks] = useState('')
  const send = useSendToPostAudit()

  function submit(event: FormEvent) {
    event.preventDefault()
    send.mutate(
      { referensi: claim.referensi, catatan: remarks },
      { onSuccess: onClose },
    )
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/40 px-4"
      role="dialog"
      aria-modal="true"
      aria-labelledby="kirim-post-audit-judul"
    >
      <form
        onSubmit={submit}
        className="w-full max-w-lg rounded-kartu bg-white p-5 shadow-lg"
      >
        <h2 id="kirim-post-audit-judul" className="text-base font-semibold text-slate-900">
          Kirim ke Post Audit
        </h2>

        {/*
          Klaim yang dikirim disebutkan ulang di sini, bukan hanya tersirat dari baris yang
          diklik. Pada antrean yang barisnya mirip satu sama lain, salah klik adalah
          kesalahan yang mudah terjadi dan tidak dapat dibatalkan — tidak ada penghapusan
          baris Post Audit.
        */}
        <dl className="mt-3 space-y-1 rounded-kontrol bg-slate-50 px-3 py-2 text-sm">
          <div className="flex gap-2">
            <dt className="w-32 shrink-0 text-slate-500">Nomor Case</dt>
            <dd className="font-medium text-slate-900">{claim.nomor_case || '—'}</dd>
          </div>
          <div className="flex gap-2">
            <dt className="w-32 shrink-0 text-slate-500">No Polis</dt>
            <dd className="text-slate-900">{claim.no_polis || '—'}</dd>
          </div>
          <div className="flex gap-2">
            <dt className="w-32 shrink-0 text-slate-500">Nama Tertanggung</dt>
            <dd className="text-slate-900">{claim.nama_tertanggung || '—'}</dd>
          </div>
        </dl>

        <label
          htmlFor="kirim-post-audit-catatan"
          className="mt-4 block text-sm font-medium text-slate-700"
        >
          Catatan
        </label>
        <textarea
          id="kirim-post-audit-catatan"
          rows={3}
          value={remarks}
          maxLength={4000}
          onChange={(event) => setRemarks(event.target.value)}
          className={[
            'mt-1 block w-full rounded-kontrol border border-slate-300 px-3 py-2 text-sm',
            'focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500/30',
          ].join(' ')}
        />
        {/*
          Catatan boleh kosong, dan itu dinyatakan — bukan dibiarkan pengguna menebaknya
          dari ada-tidaknya tanda bintang. Kolomnya memang nullable di basis data.
        */}
        <p className="mt-1 text-xs text-slate-500">Boleh dikosongkan.</p>

        {send.isError && (
          <div className="mt-3">
            <ErrorMessage
              title="Pengiriman gagal"
              description={messageOf(send.error)}
              tone="gangguan"
            />
          </div>
        )}

        <div className="mt-5 flex justify-end gap-2">
          <Button tone="kedua" onClick={onClose} disabled={send.isPending}>
            Batal
          </Button>
          {/*
            Tombol dinonaktifkan selama permintaan berjalan. Itu SATU-SATUNYA penahan
            pengiriman ganda hari ini: belum ada kunci idempotensi, dan tabelnya tidak
            punya constraint unik yang akan menolak baris kedua.
          */}
          <Button type="submit" disabled={send.isPending}>
            {send.isPending ? 'Mengirim…' : 'Kirim'}
          </Button>
        </div>
      </form>
    </div>
  )
}

function columnsFor(tab: Tab, action: (row: WorkItem) => ReactNode): Column<WorkItem>[] {
  const columns: Column<WorkItem>[] = tab.kolom.map((column) => ({
    key: column.kunci,
    title: column.judul,
    value: (row) => cellText(row, column),
    alignRight: column.kunci === 'aging' || column.kunci === 'outstanding',

    // Kolom Aging dan OutStanding TIDAK dapat diurutkan, dan itu disengaja.
    //
    // `DataTable` mengurutkan berdasarkan teks yang dikembalikan `value`, sedangkan isi
    // kolom ini berbentuk "2 days 3 hours ago". Mengurutkannya sebagai teks menaruh
    // "10 hours ago" sebelum "2 days ago" — urutan yang tampak masuk akal sampai
    // seseorang mengandalkannya untuk mencari pekerjaan yang paling lama menunggu.
    //
    // Angka mentahnya sudah dikirim server sebagai `aging_jam`; yang belum ada adalah
    // kemampuan `DataTable` mengurutkan dengan nilai selain teks yang ditampilkan. Itu
    // lingkup `TKT-U2-001`, bukan diselesaikan sepihak di satu layar.
    //
    // Sementara itu barisnya sudah datang terurut dari server — menurut tanggal kirim,
    // yang untuk kedua antrean ini berarti yang paling BARU di atas.
    noSort: column.kunci === 'aging' || column.kunci === 'outstanding',
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
 * Tanggal diformat ke `1 Juni 2026`; sisanya ditampilkan apa adanya. Nilai kosong menjadi
 * tanda pisah — bukan sel kosong yang tidak dapat dibedakan dari kolom yang gagal dimuat.
 *
 * Kolom Aging ikut jalur "apa adanya", dan itu disengaja: teksnya sudah disusun server
 * mengikuti bentuk sistem lama, dan menyusunnya ulang di sini berarti aturan yang sama
 * hidup di dua tempat.
 */
function cellText(row: WorkItem, column: TabColumn): string {
  const value = row[column.kunci]

  if (value === null || value === undefined || value === '') return '—'

  const text = String(value)

  if (isDate(text)) return formatDate(text)

  // Satu kolom membawa jamnya: Tanggal Kirim Audit Compliance. Tanggalnya diformat lewat
  // fungsi bersama yang sama, lalu jamnya ditempelkan — sehingga bentuk tanggalnya tetap
  // seragam dengan kolom lain, dan hanya jamnya yang ditambahkan.
  const stamp = text.match(/^(\d{4}-\d{2}-\d{2}) (\d{2}:\d{2})$/)
  if (stamp && stamp[1] && stamp[2]) return `${formatDate(stamp[1])} ${stamp[2]}`

  return text
}

/** isDate mengenali bentuk `YYYY-MM-DD` yang dikirim server untuk kolom tanggal saja. */
function isDate(text: string): boolean {
  return /^\d{4}-\d{2}-\d{2}$/.test(text)
}

/** messageOf mengambil pesan yang layak dibaca pengguna dari sebuah galat. */
function messageOf(error: unknown): string {
  if (error instanceof APIError) return error.message
  return 'Coba lagi beberapa saat lagi. Bila terus berulang, hubungi tim teknis.'
}
