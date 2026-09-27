import { useState, type ReactNode } from 'react'
import { useSearchParams } from 'react-router-dom'

import { APIError } from '@/api/client'
import { useSelectedPortal } from '@/app/portal'
import { Button } from '@/components/Button'
import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage } from '@/components/ErrorMessage'
import { formatDate } from '@/components/format'

import { DetailSalvagePanel } from './DetailSalvagePanel'
import { SalvageTabs } from './SalvageTabs'
import { StatusSummary } from './StatusSummary'
import { TambahSalvageForm } from './TambahSalvageForm'
import {
  useExportSalvage,
  useSalvageCounts,
  useSalvageDetail,
  useSalvageList,
  useSalvageMetadata,
} from './api'
import type { DetailKey, SalvageRow, Tab, TabColumn } from './types'

/**
 * Inbox Salvage — menu `MENU_ID 71`, pengganti harness `InboxSalvage`.
 *
 * Isinya pengelolaan **salvage**: nilai sisa barang rusak yang dapat dijual kembali, dan
 * yang MENGURANGI nilai bersih klaim (`CONTEXT.md`). Perjalanannya dari klaim yang belum
 * ditandai punya salvage sama sekali sampai barangnya terjual di balai lelang.
 *
 * # Susunan layar, dan dari mana bentuknya
 *
 * Diambil dari `Section/InboxSalvage-Section.xml` dan `InboxSalvageASM-Section.xml`:
 * tombol Tambah dan Refresh, tabel ringkas "Status Salvage / Jumlah", lalu grid berhalaman
 * 20 baris. Judul kolom TIDAK diterjemahkan — `D-13` menetapkan tampilan meniru Pega, dan
 * itulah teks yang selama ini dibaca pengguna.
 *
 * # Tiga hal yang paling mudah disalahpahami di layar ini
 *
 * PERTAMA — angka pada tabel ringkas TIDAK selalu sama dengan jumlah baris daftarnya.
 * Tiga baris menghitung populasi yang BERBEDA dari daftar yang dibukanya, dan itu keadaan
 * di Pega yang sengaja direplikasi (`P-5`).
 *
 * KEDUA — dua daftar berisi baris yang SAMA PERSIS. "Checker" dan "Salvage Diterima"
 * keduanya menyaring status transfer yang sama; di layar lama keduanya memang dua pintu
 * masuk ke kumpulan baris yang sama.
 *
 * KETIGA — pencarian pada tiga daftar COCOK PERSIS, bukan mengandung. Mengetik separuh
 * nomor klaim di sana tidak menghasilkan apa-apa, dan itu perilaku layar lama apa adanya.
 *
 * Ketiganya dinyatakan ke pengguna lewat `selisih_terencana` dan `catatan_daftar` — bukan
 * hanya tercatat di kode, karena selisih yang tidak dinyatakan akan dilaporkan berulang
 * kali sebagai kerusakan.
 *
 * # Portal Insurtech BELUM dibangun
 *
 * Layar lama punya isi tersendiri untuknya (`Section/InboxSalvageInsurtech`), dan judul
 * layarnya pun berbeda — harness memilih di antara keduanya dengan membandingkan
 * `TempGetApp.LSC_ID`. Keputusan Work Owner 2026-09-25: portal ASM dibangun lebih dulu.
 */
export function SalvageInboxPage() {
  // Daftar, halaman, dan kata kunci hidup di ALAMAT, bukan di state komponen.
  //
  // Alasannya sama dengan modul inbox lain: layar ini dibuka berpuluh kali sehari, dan
  // petugas yang kembali dari layar lain harus mendarat di tempat ia tinggalkan. Kata
  // kunci ikut, karena pencariannya menyaring DI SERVER — ia bagian dari apa yang sedang
  // dilihat, bukan preferensi tampilan.
  const [params, setParams] = useSearchParams()
  const [adding, setAdding] = useState(false)
  const [saved, setSaved] = useState('')

  // Pengajuan yang panel rinciannya sedang terbuka; kosong berarti tertutup.
  //
  // Ia state komponen, BUKAN bagian alamat seperti daftar dan halaman. Alasannya: rincian
  // dibuka untuk dibaca sekali lalu ditutup, bukan untuk ditinggalkan dan kembali — dan
  // menaruhnya di alamat membuat tombol "kembali" peramban menutup panel alih-alih
  // meninggalkan layar, yang bukan yang diharapkan orang.
  const [opened, setOpened] = useState<{ key: DetailKey; reference: string } | null>(null)

  const tabCode = params.get('daftar') ?? ''
  const page = Math.max(1, Number(params.get('halaman') ?? '1') || 1)
  const search = params.get('cari') ?? ''

  const portal = useSelectedPortal((state) => state.alias)
  const meta = useSalvageMetadata()
  const counts = useSalvageCounts()

  const tabs: Tab[] = meta.data?.daftar ?? []
  const active = tabCode || meta.data?.daftar_bawaan || ''
  const tab = tabs.find((candidate) => candidate.kode === active)

  const list = useSalvageList(active, page, search, meta.isSuccess)
  const exporting = useExportSalvage()

  // Permintaan rincian dipegang HALAMAN, bukan panelnya.
  //
  // Sebabnya: jawabannya menentukan APA yang digambar. Klaim yang belum punya pengajuan
  // masuk ke form "Menambahkan Data Salvage" — seperti di layar lama — bukan ke panel
  // rincian. Bila panelnya yang menembak server, keputusan itu diambil sesudah panelnya
  // terlanjur tergambar, dan permintaan yang sama berjalan dua kali.
  const detail = useSalvageDetail(opened?.key ?? 'pengajuan', opened?.reference ?? '')

  // Klaim tanpa pengajuan tidak dibuka sebagai panel, melainkan sebagai form pengajuan
  // baru yang sudah terisi klaimnya.
  const creatingFor =
    opened !== null && detail.data !== undefined && !detail.data.ada_pengajuan
      ? detail.data
      : null

  /** Berpindah daftar mengembalikan ke halaman pertama DAN mengosongkan pencarian. */
  function selectTab(code: string) {
    setParams(
      (current) => {
        const next = new URLSearchParams(current)
        next.set('daftar', code)
        next.delete('halaman')

        // Kata kunci dibuang, bukan dibawa.
        //
        // Alasannya bukan kerapian melainkan arti pencarian yang BERBEDA antardaftar: tiga
        // daftar mencocokkan persis, sepuluh mencocokkan sebagian. Membawa kata kunci
        // "PNC-20" dari daftar Histori ke daftar Checker menghasilkan daftar kosong yang
        // terlihat seperti kerusakan.
        next.delete('cari')
        return next
      },
      { replace: true },
    )

    // Panel rincian ikut ditutup. Pengajuan yang sedang dibuka adalah milik daftar yang
    // baru saja ditinggalkan, dan membiarkannya terbuka di bawah daftar lain membuat
    // rincian itu terbaca seolah milik baris yang sekarang tampil.
    setOpened(null)
  }

  function setPage(next: number) {
    setParams(
      (current) => {
        const updated = new URLSearchParams(current)
        if (next <= 1) updated.delete('halaman')
        else updated.set('halaman', String(next))
        return updated
      },
      { replace: true },
    )
  }

  function setSearch(value: string) {
    setParams(
      (current) => {
        const updated = new URLSearchParams(current)
        if (value === '') updated.delete('cari')
        else updated.set('cari', value)

        // Mencari mengembalikan ke halaman pertama. Tanpa itu, pencarian yang cocok
        // dengan tiga baris pada halaman satu akan menampilkan halaman lima yang kosong.
        updated.delete('halaman')
        return updated
      },
      { replace: true },
    )
  }

  if (portal === null) {
    return (
      <Frame>
        <ErrorMessage
          title="Pilih entitas lebih dulu"
          description={
            'Pengajuan salvage milik satu badan hukum, dan aplikasi ini melayani empat. ' +
            'Pilih portal di bilah atas untuk membukanya.'
          }
          tone="gangguan"
        />
      </Frame>
    )
  }

  if (meta.isError) {
    return (
      <Frame>
        <ErrorMessage
          title="Layar tidak dapat dibuka"
          description={messageOf(meta.error)}
          tone="gangguan"
        />
      </Frame>
    )
  }

  return (
    <Frame>
      {saved !== '' && (
        <div
          className="rounded-kartu border border-emerald-200 bg-emerald-50 p-4 text-sm text-emerald-900"
          role="status"
        >
          {saved}
        </div>
      )}

      <StatusSummary
        rows={counts.data?.baris ?? []}
        active={active}
        onSelect={selectTab}
        isLoading={counts.isLoading}
      />

      {creatingFor !== null ? (
        <TambahSalvageForm
          // Kunci memaksa form DIBONGKAR dan dipasang ulang saat berpindah klaim.
          //
          // Tanpanya, isian yang sudah diketik untuk klaim sebelumnya akan tertinggal di
          // form yang sekarang menyebut klaim lain — pengajuan yang tersimpan atas klaim
          // yang salah, tanpa satu pun tanda.
          key={creatingFor.no_klaim}
          statusOptions={meta.data?.pilihan_status_salvage ?? []}
          uploadColumns={meta.data?.kolom_berkas_unggahan ?? []}
          prefill={{
            nomor_klaim: creatingFor.no_klaim,
            nama_object: creatingFor.nama_object,
            nama_coverage: creatingFor.nama_coverage,
          }}
          history={creatingFor.riwayat}
          onClose={() => setOpened(null)}
          onSaved={(message) => {
            setSaved(message)
            setOpened(null)
          }}
        />
      ) : adding ? (
        <TambahSalvageForm
          statusOptions={meta.data?.pilihan_status_salvage ?? []}
          uploadColumns={meta.data?.kolom_berkas_unggahan ?? []}
          onClose={() => setAdding(false)}
          onSaved={setSaved}
        />
      ) : (
        <>
          <SalvageTabs tabs={tabs} active={active} onSelect={selectTab} />

          {tab && (
            <p className="text-sm text-slate-600">
              {tab.keterangan}
            </p>
          )}

          {tab?.catatan_daftar !== undefined && tab.catatan_daftar !== '' && (
            <p className="rounded-kartu border border-amber-200 bg-amber-50 p-3 text-sm text-amber-900">
              {tab.catatan_daftar}
            </p>
          )}

          <DataTable<SalvageRow>
            columns={columnsOf(tab, (key, reference) => setOpened({ key, reference }))}
            rows={list.data?.baris ?? []}
            rowKey={(row) => `${row.referensi}-${row.no_klaim}`}
            label={`Daftar ${tab?.nama ?? 'salvage'}`}
            isLoading={list.isLoading}
            error={
              list.isError ? (
                <ErrorMessage
                  title="Daftar tidak dapat dimuat"
                  description={messageOf(list.error)}
                  tone="gangguan"
                />
              ) : undefined
            }
            emptyMessage={emptyMessageFor(tab, search)}
            searchLabel={tab?.label_pencarian ?? 'Cari'}
            serverSearch={{
              value: search,
              onChange: setSearch,
              matchCount: list.data?.paginasi.total,
            }}
            pagination={{
              page: list.data?.paginasi.halaman ?? 1,
              size: list.data?.paginasi.ukuran ?? 20,
              total: list.data?.paginasi.total ?? 0,
              totalPage: list.data?.paginasi.total_halaman ?? 1,
              onPageChange: setPage,
              isLoading: list.isFetching,
            }}
            actions={
              <div className="flex flex-wrap items-center gap-2">
                <Button type="button" onClick={() => setAdding(true)}>
                  Tambah
                </Button>
                <Button
                  type="button"
                  tone="kedua"
                  onClick={() => void list.refetch()}
                  disabled={list.isFetching}
                >
                  Refresh
                </Button>
                <Button
                  type="button"
                  tone="kedua"
                  disabled={exporting.isPending}
                  onClick={() => exporting.mutate({ tab: active, search })}
                >
                  {exporting.isPending ? 'Menyiapkan…' : 'Export Data'}
                </Button>
              </div>
            }
          />

          {opened !== null && (
            <DetailSalvagePanel
              detailKey={opened.key}
              reference={opened.reference}
              data={detail.data}
              isPending={detail.isPending}
              isError={detail.isError}
              error={detail.error}
              onClose={() => setOpened(null)}
            />
          )}

          {exporting.error != null && (
            <ErrorMessage
              title="Berkas ekspor tidak dapat diambil"
              description={messageOf(exporting.error)}
              tone="gangguan"
            />
          )}
        </>
      )}

      <PlannedDifferences items={meta.data?.selisih_terencana ?? []} />
    </Frame>
  )
}

/** Frame adalah judul layar beserta ruang isinya. */
function Frame({ children }: { children: ReactNode }) {
  return (
    <div className="space-y-5">
      <header>
        <h1 className="text-lg font-semibold text-slate-900">
          Inbox Salvage Asuransi Sinarmas
        </h1>
        <p className="mt-1 text-sm text-slate-600">
          Pengelolaan barang sisa klaim — dari penandaan awal sampai penjualan lewat balai
          lelang.
        </p>
      </header>

      {children}
    </div>
  )
}

/**
 * columnsOf menerjemahkan kolom yang DIKIRIM SERVER menjadi kolom DataTable.
 *
 * # Kenapa kolomnya tidak ditulis di sini
 *
 * Karena tiga belas daftar memakai EMPAT susunan kolom yang berbeda, dan dua di antaranya
 * berbeda hanya satu kolom. Menuliskannya dengan tangan di layar berarti daftar yang sama
 * hidup di dua tempat — dan yang satu akan tertinggal saat yang lain diperbaiki.
 *
 * Yang tetap milik layar adalah cara satu sel DIGAMBAR: tanggal diformat, nilai uang
 * diratakan kanan. Keduanya urusan tampilan, bukan urusan bentuk data.
 */
function columnsOf(
  tab: Tab | undefined,
  onOpenDetail: (key: DetailKey, reference: string) => void,
): Column<SalvageRow>[] {
  if (!tab) return []

  const columns: Column<SalvageRow>[] = tab.kolom.map((column) => ({
    key: column.kunci,
    title: column.judul,
    value: (row) => valueOf(row, column.kunci),
    render: (row) => renderCell(row, column),
    alignRight: column.angka,
  }))

  // Kolom aksi ada di KETIGA BELAS daftar — itu keadaan di layar lama, tempat sebelas
  // tombol `SetDataDetailSalvage_act` tersebar di seluruh grid-nya.
  //
  // Yang berbeda hanyalah KUNCI yang dikirimkannya: ID pengajuan pada tujuh daftar yang
  // barisnya pengajuan, nomor klaim pada enam daftar yang barisnya klaim dan tidak
  // membawa ID pengajuan sama sekali.
  columns.push({
    key: 'aksi',
    title: 'Aksi',
    width: '8rem',
    noSort: true,
    alignRight: true,
    value: () => '',
    render: (row) => {
      const reference =
        tab.kunci_rincian === 'klaim' ? row.no_klaim : row.id_salvage

      if (reference === '') return <span className="text-slate-400">—</span>

      return (
        <Button
          type="button"
          tone="halus"
          onClick={() => onOpenDetail(tab.kunci_rincian, reference)}
        >
          Detail
        </Button>
      )
    },
  })

  return columns
}

/** valueOf mengambil isi satu sel sebagai TEKS — yang dicari dan diurutkan. */
function valueOf(row: SalvageRow, key: string): string {
  const cell = (row as unknown as Record<string, unknown>)[key]
  return typeof cell === 'string' ? cell : ''
}

/**
 * renderCell menggambar satu sel.
 *
 * Tiga perlakuan khusus, dan ketiganya urusan tampilan:
 *
 *   tanggal      diformat mengikuti kebiasaan layar lain
 *   nilai uang   diratakan kanan dengan angka tabular, supaya kolomnya sejajar
 *   sel kosong   digambar sebagai tanda hubung, bukan ruang kosong
 *
 * Yang terakhir penting di layar ini: kolom "Catatan" SELALU kosong, dan sel kosong tanpa
 * tanda tidak dapat dibedakan dari kolom yang gagal dimuat.
 */
function renderCell(row: SalvageRow, column: TabColumn): ReactNode {
  const raw = valueOf(row, column.kunci)
  if (raw === '') return <span className="text-slate-400">—</span>

  if (column.kunci === 'tanggal_input' || column.kunci === 'tanggal_kejadian') {
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
 * Nilainya datang sebagai TEKS dan tetap teks sampai di sini — `D-51` menetapkan nilai
 * uang disimpan presisi penuh dan hanya dibulatkan SAAT DITAMPILKAN. Inilah tempat
 * "saat ditampilkan" itu.
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
 * emptyMessageFor menjelaskan MENGAPA daftarnya kosong, bukan sekadar menyatakan kosong.
 *
 * Pada daftar yang pencariannya cocok persis, kekosongan hampir selalu berarti pengguna
 * mengetik separuh nomor klaim — dan tanpa keterangan itu, ia akan menyimpulkan datanya
 * hilang.
 */
function emptyMessageFor(tab: Tab | undefined, search: string): string {
  if (search !== '' && tab?.pencarian_cocok_persis === true) {
    return (
      `Tidak ada yang cocok dengan "${search}". Pencarian di daftar ini COCOK PERSIS — ` +
      'ketik nomor klaim atau nama PIC selengkapnya, bukan sebagiannya.'
    )
  }
  if (search !== '') {
    return `Tidak ada pengajuan yang cocok dengan "${search}".`
  }
  return 'Belum ada pengajuan pada daftar ini.'
}

/**
 * PlannedDifferences menggambar selisih terhadap Pega yang sudah diputuskan.
 *
 * Ia DIGAMBAR, bukan hanya tercatat di kode, dan alasannya nyata di layar ini: tiga angka
 * pada tabel ringkas memang tidak cocok dengan tabel di bawahnya, dan tanpa keterangan itu
 * akan dilaporkan berulang kali sebagai kerusakan.
 *
 * Dilipat secara bawaan supaya tidak menyaingi isi layar, tetapi TIDAK disembunyikan.
 */
function PlannedDifferences({ items }: { items: string[] }) {
  if (items.length === 0) return null

  return (
    <details className="rounded-kartu border border-slate-200 bg-white p-4">
      <summary className="cursor-pointer text-sm font-medium text-slate-800">
        Perbedaan yang disengaja terhadap layar Pega ({items.length})
      </summary>

      <ul className="mt-3 space-y-2 text-sm text-slate-600">
        {items.map((item) => (
          <li key={item} className="border-l-2 border-slate-200 pl-3">
            {item}
          </li>
        ))}
      </ul>
    </details>
  )
}

function messageOf(error: unknown): string {
  if (error instanceof APIError) return error.message
  if (error instanceof Error && error.message !== '') return error.message
  return 'Coba lagi beberapa saat lagi. Bila terus berulang, hubungi tim teknis.'
}
