import { useEffect, useState, type ReactNode } from 'react'

import { APIError } from '@/api/client'
import { useSelectedPortal } from '@/app/portal'
import { Button } from '@/components/Button'
import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage } from '@/components/ErrorMessage'
import { formatDate } from '@/components/format'

import { BandingHargaTabs } from './BandingHargaTabs'
import { DecisionConfirm } from './DecisionConfirm'
import { DecisionPanel } from './DecisionPanel'
import { DocumentPanel } from './DocumentPanel'
import { StatusSummary } from './StatusSummary'
import {
  useBandingHargaSalvageList,
  useBandingHargaSalvageMetadata,
  useBandingHargaSalvageSummary,
} from './api'
import {
  EMPTY_FILTER,
  type AppealRow,
  type DocumentScope,
  type FilterForm,
  type QueueInfo,
  type Tab,
  type TabColumn,
} from './types'

/**
 * Baris yang tombol Approve atau Reject-nya sedang ditekan.
 *
 * Barisnya dibawa UTUH, bukan hanya kuncinya, karena kotak penegasan menggambar kedua harga
 * yang sedang dipertentangkan — dan mencarinya kembali di daftar berarti kotak itu ikut
 * kosong ketika halaman berganti di belakangnya.
 */
type PendingDecision = { row: AppealRow; approve: boolean }

/**
 * Inbox Banding Harga Salvage — menu `MENU_ID 72`, pengganti harness `InboxRequestSalvage`.
 *
 * Isinya antrean **banding harga barang salvage**: balai lelang (SimasBid) menilai sebuah
 * barang tidak layak dijual pada harga yang diajukan PIC, lalu mengajukan harga tandingan.
 * Layar ini tempat komite ASM membacanya.
 *
 * Per `D-79` ia benar-benar Inbox — barisnya pekerjaan, ia hilang dari antrean begitu
 * diputuskan, dan "hanya milik saya" adalah aturan kewenangan, bukan sekadar penyaring.
 *
 * # Bedakan dari Inbox Salvage
 *
 * `MENU_ID 71` mengelola pengajuan salvage dari awal sampai lelang. Layar INI mengerjakan
 * satu hal saja — banding harga — dan tabel intinya pun berbeda. Di Pega keduanya harness
 * terpisah, dan menyatukannya akan menggabungkan dua layar yang memang tidak sama.
 *
 * # Susunan layar, dan dari mana bentuknya
 *
 * Diambil dari `Section/InboxReqSalvageASM-Section.xml` apa adanya: tabel ringkas "Status
 * Salvage / Jumlah" di atas, lalu grid berhalaman dengan kotak "Cari No Klaim" di atasnya.
 * Judul kolom TIDAK diterjemahkan dan TIDAK dibetulkan — `D-13` menetapkan tampilan meniru
 * Pega, dan itulah teks yang selama ini dibaca pengguna. Termasuk kolom "Object Name" yang
 * sebenarnya berisi jenis salvage.
 *
 * # Tiga hal yang mudah disalahpahami di layar ini
 *
 * PERTAMA — pencarian COCOK PERSIS, bukan mengandung. Mengetik separuh nomor klaim tidak
 * menghasilkan apa-apa, dan itu perilaku layar lama apa adanya (`P-5`).
 *
 * KEDUA — antrean yang tampil bisa jadi BUKAN milik Anda. Satu Operator ID melihat antrean
 * komite lain, dan satu lagi hanya melihat baris yang komite sebelumnya sudah putuskan.
 * Keduanya aturan bernama orang yang ditiru dari Pega atas keputusan Work Owner; layar
 * menyatakannya lewat `antrean.catatan_perwakilan` dan `antrean.catatan_giliran`.
 *
 * KETIGA — angka pada tabel ringkas SAMA dengan jumlah baris gridnya. Di layar lama keduanya
 * berbeda, dan perbaikan itu dinyatakan lewat `selisih_terencana`.
 *
 * KEEMPAT, dan yang paling mahal bila terlewat — menekan Approve **tidak selalu mengubah
 * harga barang**, dan **tidak pernah** memberi tahu balai lelang. Yang pertama perilaku layar
 * lama apa adanya: penerapan harga di sana dijaga syarat bernama satu orang. Yang kedua
 * keputusan sadar Work Owner, karena layanan pengirimannya menunjuk host dev tanpa
 * autentikasi. Keduanya dinyatakan pada kabar hasil tiap keputusan, bukan hanya di catatan
 * bawah — pesan "berhasil disimpan" saja akan menyiratkan lebih daripada yang terjadi.
 */
export function InboxBandingHargaSalvagePage() {
  const [filter, setFilter] = useState<FilterForm>(EMPTY_FILTER)
  const [page, setPage] = useState(1)

  /**
   * Nomor klaim yang panel riwayatnya sedang terbuka. Kosong berarti tertutup.
   *
   * Disimpan di layar, bukan di alamat: panel ini dibuka dan ditutup berulang kali sambil
   * menelusuri satu halaman, dan menaruhnya di alamat akan membuat tombol kembali menempuh
   * setiap pembukaan itu satu per satu.
   */
  const [riwayatKlaim, setRiwayatKlaim] = useState('')

  /** Baris yang sedang ditegaskan keputusannya, dan hasil keputusan terakhir. */
  const [pending, setPending] = useState<PendingDecision | null>(null)
  const [hasil, setHasil] = useState('')

  /** Banding yang dialog dokumennya sedang terbuka. Null berarti tertutup. */
  const [dokumen, setDokumen] = useState<DocumentScope | null>(null)

  /**
   * Isi kotak cari yang sedang DIKETIK, terpisah dari kata kunci yang sudah dikirim.
   *
   * Keduanya dipisah karena pencarian di layar ini cocok persis: hampir setiap keadaan
   * setengah-ketik menghasilkan nol baris, dan mengirim satu permintaan per huruf berarti
   * memicu layar kosong berulang kali untuk satu nomor klaim yang sedang diketik.
   */
  const [draft, setDraft] = useState('')

  // Ketikan menunggu jeda sebelum dikirim. Jedanya cukup panjang untuk menelan satu nomor
  // klaim yang diketik cepat, dan cukup pendek untuk tidak terasa seperti layar menggantung.
  useEffect(() => {
    if (draft === filter.cari) return

    const timer = setTimeout(() => {
      setFilter((previous) => ({ ...previous, cari: draft }))
      setPage(1)
      setRiwayatKlaim('')
    }, 350)
    return () => clearTimeout(timer)
  }, [draft, filter.cari])

  const portal = useSelectedPortal((state) => state.alias)
  const meta = useBandingHargaSalvageMetadata()
  const list = useBandingHargaSalvageList(filter, page, meta.isSuccess)
  const summary = useBandingHargaSalvageSummary(meta.isSuccess)

  const tabs: Tab[] = meta.data?.tab ?? []
  const active = filter.tab || meta.data?.tab_bawaan || ''
  const tab = tabs.find((candidate) => candidate.kode === active)

  /**
   * Berpindah tab MEMBERSIHKAN kotak cari dan mengembalikan ke halaman pertama.
   *
   * Membawa kata kunci lama ke tab baru akan menampilkan antrean yang tampak kosong padahal
   * isinya ada — dan pada pencarian yang cocok persis, kekosongan itu hampir selalu terjadi.
   */
  function selectTab(code: string) {
    setFilter({ ...EMPTY_FILTER, tab: code })
    setDraft('')
    setPage(1)

    // Panel riwayat ikut ditutup. Ia milik satu klaim pada satu daftar; membiarkannya
    // terbuka setelah berpindah tab menampilkan rincian baris yang tidak lagi terlihat.
    setRiwayatKlaim('')

    // Kotak penegasan ikut ditutup, karena barisnya tidak lagi terlihat. Pesan hasil TIDAK
    // ikut dibuang: pengguna yang baru saja menyetujui sesuatu lalu berpindah ke History
    // justru sedang memeriksa hasil itu.
    setPending(null)
    setDokumen(null)
  }

  /**
   * Tombol "Refresh" pada harness lama.
   *
   * Ia memuat ulang daftar DAN tabel ringkas sekaligus. Memuat salah satunya saja akan
   * membuat angka ringkas dan jumlah baris grid berselisih sesaat — persis keadaan yang
   * baru saja diperbaiki dari layar lama.
   */
  function refreshAll() {
    list.refetch()
    summary.refetch()
  }

  if (portal === null) {
    return (
      <PageFrame>
        <ErrorMessage
          title="Pilih entitas lebih dulu"
          description={
            'Banding harga salvage milik satu badan hukum, dan aplikasi ini melayani empat. ' +
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

  const info = list.data?.paginasi
  const queue = list.data?.antrean ?? summary.data?.antrean

  return (
    <PageFrame>
      <QueueNotices queue={queue} />
      <DecisionResultNotice message={hasil} onDismiss={() => setHasil('')} />

      <div className="mt-4">
        <StatusSummary
          rows={summary.data?.baris ?? []}
          active={active}
          onSelect={selectTab}
          isLoading={summary.isPending}
        />
      </div>

      <div className="mt-4">
        <BandingHargaTabs tabs={tabs} active={active} onSelect={selectTab} />
      </div>

      {tab && (
        <>
          <p className="mt-3 text-sm text-slate-600">{tab.keterangan}</p>

          <SearchBar
            label={meta.data?.label_cari ?? 'Cari No Klaim'}
            placeholder={meta.data?.petunjuk_cari ?? ''}
            value={draft}
            onChange={setDraft}
          />

          <div className="mt-4">
            <DataTable<AppealRow>
              columns={columnsFor(tab, setRiwayatKlaim, setDokumen, (row, approve) => {
                setPending({ row, approve })

                // Pesan hasil sebelumnya dibuang saat keputusan BARU dimulai. Membiarkannya
                // berarti pengguna membaca kabar keputusan barang sebelumnya tepat di atas
                // kotak keputusan barang yang berbeda.
                setHasil('')
              })}
              rows={list.data?.baris ?? []}
              rowKey={rowKeyFor(tab)}
              title={tab.nama}
              label={`Daftar ${tab.nama}`}
              actions={
                <Button
                  tone="kedua"
                  onClick={refreshAll}
                  disabled={list.isFetching || summary.isFetching}
                >
                  Refresh
                </Button>
              }
              // Kotak cari bawaan disembunyikan: layar ini punya kotaknya sendiri di atas,
              // dan yang kedua hanya akan menyaring halaman yang sedang terbuka — hasilnya
              // menyesatkan pada data berhalaman.
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
              emptyMessage={emptyMessageFor(tab, filter)}
              {...(info
                ? {
                    pagination: {
                      page: info.halaman,
                      size: info.ukuran,
                      total: info.total,
                      totalPage: info.total_halaman,
                      onPageChange: setPage,
                      isLoading: list.isFetching,
                    },
                  }
                : {})}
            />
            <DocumentPanel scope={dokumen} onClose={() => setDokumen(null)} />
            <DecisionConfirm
              row={pending?.row ?? null}
              approve={pending?.approve ?? false}
              onCancel={() => setPending(null)}
              onDecided={(message) => {
                setPending(null)
                setHasil(message)
              }}
            />
            <DecisionPanel
              claimNo={riwayatKlaim}
              columns={meta.data?.kolom_rincian ?? []}
              onClose={() => setRiwayatKlaim('')}
            />
          </div>
        </>
      )}

      <Notes
        title="Yang sengaja berbeda dari layar lama"
        lines={meta.data?.selisih_terencana ?? []}
      />
      <Notes title="Yang belum tersedia" lines={meta.data?.keterbatasan ?? []} />
    </PageFrame>
  )
}

function PageFrame({ children }: { children: ReactNode }) {
  return (
    <div className="mx-auto max-w-[96rem] px-4 py-8">
      <header className="border-b border-slate-200 pb-4">
        <h1 className="text-xl font-semibold text-slate-900">Inbox Banding Harga Salvage</h1>
        <p className="mt-1 text-sm text-slate-600">
          Banding harga dari balai lelang atas barang salvage: yang menunggu keputusan Anda,
          dan yang sudah Anda putuskan.
        </p>
      </header>
      {children}
    </div>
  )
}

/**
 * Keterangan tentang antrean SIAPA yang sedang dibaca.
 *
 * Ia digambar di ATAS segalanya, bukan di catatan bawah, karena ia mengubah arti seluruh
 * angka di layar ini. Petugas yang melihat antrean komite lain tanpa diberi tahu akan
 * menyimpulkan antreannya sendiri kosong — dan itu kesimpulan yang tidak akan ia laporkan
 * sebagai kerusakan.
 */
function QueueNotices({ queue }: { queue: QueueInfo | undefined }) {
  if (!queue) return null

  const lines = [queue.catatan_perwakilan, queue.catatan_giliran].filter(
    (line) => line !== '',
  )
  if (lines.length === 0) return null

  return (
    <div
      className="mt-4 rounded-kartu border border-amber-200 bg-amber-50 px-4 py-3"
      role="status"
    >
      <ul className="space-y-1 text-sm text-amber-900">
        {lines.map((line) => (
          <li key={line}>{line}</li>
        ))}
      </ul>
    </div>
  )
}

/**
 * Kotak "Cari No Klaim" di atas tabel.
 *
 * Jangkauannya dinyatakan tepat di bawah kotaknya, bukan hanya di catatan bawah. Alasannya
 * konkret: pencariannya COCOK PERSIS, sehingga mengetik separuh nomor klaim menghasilkan nol
 * baris. Tanpa keterangan itu, pengguna akan menyimpulkan antreannya kosong.
 */
function SearchBar({
  label,
  placeholder,
  value,
  onChange,
}: {
  label: string
  placeholder: string
  value: string
  onChange: (text: string) => void
}) {
  return (
    <div className="mt-4 w-full sm:w-80">
      <label
        htmlFor="inbox-banding-harga-salvage-cari"
        className="block text-sm font-medium text-slate-700"
      >
        {label}
      </label>
      {/*
        Kotak ini sengaja TIDAK dinonaktifkan saat permintaan sedang berjalan.
        Menonaktifkannya berarti huruf yang diketik selama permintaan itu HILANG — dan
        karena setiap ketikan memicu permintaan, kotak yang menonaktifkan diri akan menelan
        sebagian besar nomor klaim yang diketik cepat.
      */}
      <input
        id="inbox-banding-harga-salvage-cari"
        type="search"
        value={value}
        placeholder={placeholder}
        onChange={(event) => onChange(event.target.value)}
        className={[
          'mt-1 block w-full rounded-kontrol border border-slate-300 px-3 py-2 text-sm',
          'focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500/30',
        ].join(' ')}
      />
      <p className="mt-1 text-xs text-slate-500">
        Harus nomor klaim UTUH — pencarian di layar ini cocok persis, sama seperti di layar
        lama. Separuh nomor tidak menghasilkan baris.
      </p>
    </div>
  )
}

/**
 * Catatan di bawah tabel.
 *
 * Isinya datang dari SERVER, bukan ditulis tetap di sini, supaya hilang dengan sendirinya
 * begitu penghalangnya hilang. Tanpa catatan ini, angka ringkas yang berbeda dari Pega dan
 * keputusan yang tidak sampai ke balai lelang akan dilaporkan berulang kali sebagai
 * kerusakan — atau, yang lebih buruk, tidak dilaporkan sama sekali.
 */
function Notes({ title, lines }: { title: string; lines: string[] }) {
  if (lines.length === 0) return null

  return (
    <section className="mt-6 rounded-kartu border border-slate-200 bg-slate-50 px-4 py-3">
      <h2 className="text-sm font-medium text-slate-800">{title}</h2>
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
 * Kolom "Action" ada di KEDUA grid, dan isinya berbeda — persis seperti di layar lama:
 *
 *   Request        Approve dan Reject (`Section/ButtonApproveRejectedRequest`)
 *   History Cheker rincian keputusan (`Flow Action/DetailHistoryRequestSalvage`)
 *
 * Tombol "Lihat File" milik layar lama belum ada di sini; ia membuka `DokumenBandingSalvage`,
 * satu-satunya artefak layar ini yang masih kurang. Ketiadaannya dinyatakan lewat
 * `keterbatasan`, bukan lewat tombol yang menolak.
 */
function columnsFor(
  tab: Tab,
  onOpenHistory: (claimNo: string) => void,
  onOpenDocuments: (scope: DocumentScope) => void,
  onDecide: (row: AppealRow, approve: boolean) => void,
): Column<AppealRow>[] {
  const columns: Column<AppealRow>[] = tab.kolom.map((column) => ({
    key: column.kunci,
    title: column.judul,
    value: (row) => valueOf(row, column),
    render: (row) => renderCell(row, column),
    alignRight: column.angka,
  }))

  if (isHistoryTab(tab)) {
    columns.push({
      key: 'aksi',
      title: '',
      width: '10rem',
      noSort: true,
      alignRight: true,
      value: () => '',
      render: (row) => (
        <Button tone="kedua" onClick={() => onOpenHistory(row.no_klaim)}>
          Lihat Riwayat
        </Button>
      ),
    })
    return columns
  }

  columns.push({
    key: 'aksi',
    title: 'Action',
    width: '19rem',
    noSort: true,
    alignRight: true,
    value: () => '',

    /*
      Keduanya bernada `kedua`, bukan satu `utama` dan satu `halus`.

      Alasannya bukan selera: keduanya sama-sama keputusan atas nilai uang, dan tidak ada
      satu pun yang "lebih dituju" — menyetujui bukan jalan yang benar, sebagaimana menolak
      pun bukan. Menggambar Approve sebagai tombol utama biru akan mengarahkan mata ke
      salah satunya pada layar yang justru menuntut pilihan sadar.
    */
    render: (row) => (
      <div className="flex flex-wrap justify-end gap-2">
        <Button tone="kedua" onClick={() => onDecide(row, true)}>
          Approve
        </Button>
        <Button tone="kedua" onClick={() => onDecide(row, false)}>
          Reject
        </Button>
        <Button
          tone="halus"
          onClick={() =>
            onOpenDocuments({
              no_klaim: row.no_klaim,
              detail_object: row.detail_object,
              id_salvage: row.id_salvage,
              nama_barang: row.nama_barang,
            })
          }
        >
          Lihat File
        </Button>
      </div>
    ),
  })

  return columns
}

/**
 * Kabar hasil keputusan yang baru saja tersimpan.
 *
 * Kalimatnya datang dari SERVER, bukan disusun di sini, dan itu bukan kerapian: pada layar
 * ini "disetujui" tidak selalu berarti "harganya berubah" — penerapan harga hanya dilakukan
 * jenjang komite terakhir. Kalimat yang disusun layar akan menyatakan sesuatu yang belum
 * tentu terjadi, karena layar tidak tahu langkah mana yang benar-benar berjalan.
 *
 * Ia bertahan sampai ditutup atau sampai keputusan berikutnya dimulai, bukan hilang sendiri
 * setelah beberapa detik: isinya memuat satu hal yang harus dibaca sampai habis — bahwa
 * balai lelang belum diberi tahu.
 */
function DecisionResultNotice({
  message,
  onDismiss,
}: {
  message: string
  onDismiss: () => void
}) {
  if (message === '') return null

  return (
    <div
      className="mt-4 flex items-start gap-3 rounded-kartu border border-emerald-200 bg-emerald-50 px-4 py-3"
      role="status"
    >
      <p className="flex-1 text-sm text-emerald-900">{message}</p>
      <Button tone="halus" onClick={onDismiss}>
        Tutup
      </Button>
    </div>
  )
}

/**
 * isHistoryTab mengenali grid History Cheker dari BENTUKNYA, bukan dari kode tabnya.
 *
 * Kode tab ditetapkan server, dan menuliskannya di sini berarti nilai yang sama hidup di dua
 * tempat. Yang membedakan keduanya nyata dan tidak berubah: hanya grid Request yang punya
 * kolom "Detail Object", karena hanya ia yang barisnya satu BARANG.
 */
function isHistoryTab(tab: Tab): boolean {
  return !tab.kolom.some((column) => column.kunci === 'detail_object')
}

/**
 * rowKeyFor menyusun kunci baris yang UNIK pada masing-masing tab.
 *
 * Keduanya berbeda, dan itu bukan pilihan gaya:
 *
 *   Request        satu baris = satu barang, dikunci IDDETAILSALVAGE
 *   History Cheker satu baris = satu pengajuan, dan satu klaim dapat punya beberapa
 *
 * Memakai nomor klaim saja akan membuat React menganggap dua baris History sebagai satu
 * baris yang sama — dan salah satunya lenyap dari layar tanpa satu pun galat.
 */
function rowKeyFor(tab: Tab): (row: AppealRow) => string {
  if (isHistoryTab(tab)) {
    return (row) => `${row.no_klaim}|${row.id_salvage}`
  }
  return (row) => `${row.no_klaim}|${row.detail_object}`
}

/** valueOf mengambil teks polos satu sel — yang dicari dan diurutkan DataTable. */
function valueOf(row: AppealRow, column: TabColumn): string {
  const raw = row[column.kunci]
  if (raw == null) return ''
  return String(raw)
}

/**
 * renderCell menggambar satu sel.
 *
 *   tanggal      diformat mengikuti kebiasaan layar lain
 *   nilai uang   diratakan kanan dengan angka tabular, supaya kolomnya sejajar
 *   sel kosong   digambar sebagai tanda hubung, bukan ruang kosong
 *
 * Yang terakhir penting di layar ini: kolom "Note Checker" memang sering kosong — di layar
 * lama ia bahkan SELALU kosong — dan sel kosong tanpa tanda tidak dapat dibedakan dari kolom
 * yang gagal dimuat.
 */
function renderCell(row: AppealRow, column: TabColumn): ReactNode {
  const raw = valueOf(row, column)
  if (raw === '') return <span className="text-slate-400">—</span>

  if (column.kunci === 'tanggal_request') {
    return formatDate(raw)
  }

  if (column.angka) {
    return <span className="tabular-nums">{formatMoney(raw)}</span>
  }

  return raw
}

/**
 * formatMoney menggambar nilai uang dengan pemisah ribuan.
 *
 * Nilainya datang sebagai TEKS dan tetap teks sampai di sini — `D-51` menetapkan nilai uang
 * disimpan presisi penuh dan hanya dibulatkan SAAT DITAMPILKAN. Inilah tempat "saat
 * ditampilkan" itu.
 *
 * Nilai yang tidak terbaca sebagai angka digambar apa adanya, bukan diganti nol. Nol yang
 * tidak dapat dibedakan dari kegagalan pembacaan adalah kelas cacat yang sama dengan
 * `GETSELISIHJAM` (`D-49` butir 10).
 */
function formatMoney(raw: string): string {
  const parsed = Number(raw.replace(',', '.'))
  if (!Number.isFinite(parsed)) return raw
  return parsed.toLocaleString('id-ID', { maximumFractionDigits: 2 })
}

/**
 * emptyMessageFor menjelaskan MENGAPA antreannya kosong, bukan sekadar menyatakan kosong.
 *
 * Pada pencarian yang cocok persis, kekosongan hampir selalu berarti pengguna mengetik
 * separuh nomor klaim — dan tanpa keterangan itu, ia akan menyimpulkan datanya hilang.
 */
function emptyMessageFor(tab: Tab, filter: FilterForm): string {
  const keyword = filter.cari.trim()
  if (keyword !== '') {
    return (
      `Tidak ada yang cocok dengan "${keyword}". Pencarian di layar ini COCOK PERSIS — ` +
      'ketik nomor klaim utuh, bukan sebagiannya.'
    )
  }
  return `Tidak ada banding harga pada antrean ${tab.nama}.`
}

/** messageOf mengambil pesan yang layak dibaca pengguna dari sebuah galat. */
function messageOf(error: unknown): string {
  if (error instanceof APIError) return error.message
  return 'Coba lagi beberapa saat lagi. Bila terus berulang, hubungi tim teknis.'
}
