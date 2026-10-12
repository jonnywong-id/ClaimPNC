import type { ReactNode } from 'react'
import { Link, useParams } from 'react-router-dom'

import { APIError } from '@/api/client'
import { useSelectedPortal } from '@/app/portal'
import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage } from '@/components/ErrorMessage'
import { formatDate } from '@/components/format'

import { useOutstandingClaimDetail, useOutstandingClaimLayout } from './api'
import type { DetailResponse, Field, Grid, GridRow, Group, LayoutResponse } from './types'

/**
 * Rincian satu klaim treaty proporsional — pengganti Flow Action `OutstandingClaim` beserta
 * `Section/OutstandingClaim-Section.xml`.
 *
 * # Ia tidak punya butir menu, dan memang tidak boleh punya
 *
 * Di Pega layar ini hanya dapat dicapai dengan mengklik nomor klaim di Inbox Claim Treaty
 * Prop (`MENU_ID 54`). Satu-satunya pintunya di sini pun sama.
 *
 * # Susunan layar
 *
 * Sepuluh kelompok, berurutan seperti kontainer di section: Treaty Information, Claim
 * Information, Insured Interest, Deductible, Claim Amount, Result Claim, Estimation, Claim
 * Spreaded, Attachment, dan Suggestion. Judul isian dan judul kolom grid diambil dari
 * `pyLabelFieldValue` apa adanya (`D-13`) — termasuk yang salah ketik ("Geoss Estimate
 * Treaty", "Esstimation ASM") dan yang huruf besarnya tidak konsisten.
 *
 * # Kenapa kelompok, isian, dan grid datang dari server
 *
 * Karena seluruhnya hasil pembacaan export Pega, dan tempat pembacaan itu tercatat adalah
 * backend (`internal/outstandingclaim/section.go`). Menyalinnya ke sini berarti 97 judul
 * isian dan 10 susunan grid hidup di dua tempat.
 *
 * # Layar ini BACA-SAJA, dan itu bukan kelalaian
 *
 * Flow Action aslinya MENULIS: ia menyimpan kembali objek kerja beserta seluruh page list di
 * dalamnya. Selama masa paralel, tabel objek kerja dan `POOLDATA.JSON_KLAIM` dimiliki Pega
 * (`P-1`). Perubahan klaim treaty tetap dilakukan lewat Pega.
 */
export function OutstandingClaimPage() {
  const { no_klaim: claimID = '' } = useParams<{ no_klaim: string }>()
  const portal = useSelectedPortal((state) => state.alias)

  const layout = useOutstandingClaimLayout()
  const detail = useOutstandingClaimDetail(claimID)

  if (portal === null) {
    return (
      <PageFrame claimID={claimID}>
        <ErrorMessage
          title="Pilih entitas lebih dulu"
          description={
            'Klaim treaty milik satu badan hukum, dan aplikasi ini melayani empat. Pilih ' +
            'portal di bilah atas untuk membukanya.'
          }
          tone="gangguan"
        />
      </PageFrame>
    )
  }

  if (layout.isError || detail.isError) {
    const error = layout.error ?? detail.error
    const notFound = error instanceof APIError && error.status === 404

    return (
      <PageFrame claimID={claimID}>
        <ErrorMessage
          title={notFound ? 'Klaim tidak ditemukan' : 'Rincian tidak dapat dimuat'}
          description={messageOf(error)}
          tone="gangguan"
        />
      </PageFrame>
    )
  }

  if (layout.isPending || detail.isPending) {
    return (
      <PageFrame claimID={claimID}>
        <p className="mt-6 text-sm text-slate-600" role="status">
          Memuat rincian klaim…
        </p>
      </PageFrame>
    )
  }

  return (
    <PageFrame claimID={claimID} claim={detail.data}>
      <div className="mt-6 space-y-6">
        {layout.data.kelompok.map((group) => (
          <GroupCard
            key={group.kode}
            group={group}
            claim={detail.data}
            layout={layout.data}
          />
        ))}
      </div>
    </PageFrame>
  )
}

function PageFrame({
  claimID,
  claim,
  children,
}: {
  claimID: string
  claim?: DetailResponse
  children: ReactNode
}) {
  return (
    <div className="mx-auto max-w-[96rem] px-4 py-8">
      <header className="border-b border-slate-200 pb-4">
        {/*
          Tautan kembali digambar SEBELUM judul, bukan sebagai tombol di kanan: layar ini
          hanya dapat dicapai dari antrean, dan jalan pulang adalah hal pertama yang dicari
          pengguna ketika ia membuka klaim yang keliru.
        */}
        <Link
          to="/inbox-claim-treaty-prop"
          className="text-sm text-blue-700 hover:underline focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500/50"
        >
          ← Kembali ke Claim Treatyin In Progress
        </Link>

        {/*
          Judulnya "Outstanding Claim", mengikuti judul di dalam section Pega
          (`<center><b>Outstanding Claim</b></center>`), diikuti nomor klaimnya supaya
          pengguna yang membuka beberapa tab tahu mana yang mana.
        */}
        <h1 className="mt-2 text-xl font-semibold text-slate-900">
          Outstanding Claim — {claimID}
        </h1>

        {claim && (
          <p className="mt-1 text-sm text-slate-600">
            {claim.isian['insured_name'] || 'Tertanggung tidak tercatat'}
            {' · '}
            {claim.status_kerja || 'Status tidak tercatat'}
            {claim.operator_pengubah !== '' && ` · diubah terakhir oleh ${claim.operator_pengubah}`}
          </p>
        )}
      </header>
      {children}
    </div>
  )
}

/**
 * Satu kelompok: isiannya lebih dulu, lalu grid miliknya.
 *
 * Kelompok yang TIDAK punya isian maupun grid tidak digambar sama sekali. Kartu kosong
 * berjudul saja terbaca sebagai bagian yang gagal dimuat.
 */
function GroupCard({
  group,
  claim,
  layout,
}: {
  group: Group
  claim: DetailResponse
  layout: LayoutResponse
}) {
  const grids = group.grid
    .map((code) => layout.grid.find((candidate) => candidate.kode === code))
    .filter((grid): grid is Grid => grid !== undefined)

  if (group.isian.length === 0 && grids.length === 0) return null

  return (
    <section className="rounded-kartu border border-slate-200 bg-white">
      <h2 className="border-b border-slate-200 px-4 py-3 text-sm font-semibold text-slate-800">
        {group.judul}
      </h2>

      {group.isian.length > 0 && (
        /*
          Daftar deskripsi, bukan tabel: isinya pasangan label dan nilai, dan tabel dua
          kolom yang barisnya tidak dapat diurutkan hanya meniru bentuk tabel tanpa
          manfaatnya.
        */
        <dl className="grid grid-cols-1 gap-x-6 gap-y-3 px-4 py-4 sm:grid-cols-2 lg:grid-cols-3">
          {group.isian.map((field) => (
            <FieldCell key={field.kunci} field={field} claim={claim} />
          ))}
        </dl>
      )}

      {grids.length > 0 && (
        <div className="space-y-4 px-4 pt-2 pb-4">
          {grids.map((grid) => (
            <GridBlock key={grid.kode} grid={grid} rows={claim.baris[grid.kode] ?? []} />
          ))}
        </div>
      )}
    </section>
  )
}

/**
 * Satu isian.
 *
 * Isian TERHALANG digambar dengan penanda alih-alih tanda pisah biasa. Tanpa penanda, isian
 * yang tidak punya sumber sama sekali tampak sama dengan isian yang datanya memang belum
 * diisi — dan yang pertama adalah hal yang harus dilaporkan, yang kedua bukan.
 */
function FieldCell({ field, claim }: { field: Field; claim: DetailResponse }) {
  return (
    <div className="min-w-0">
      <dt className="text-xs font-medium tracking-wide text-slate-500">{field.judul}</dt>
      <dd className="mt-0.5 text-sm break-words text-slate-900">
        {field.terhalang ? (
          <span
            className="text-xs text-amber-800"
            title="Isian ini belum punya sumber yang dapat dibaca di sistem baru."
          >
            belum tersedia
          </span>
        ) : (
          cellText(claim.isian[field.kunci])
        )}
      </dd>
    </div>
  )
}

/**
 * Satu grid.
 *
 * Grid terhalang menggambar ALASAN dan pemiliknya, bukan tabel kosong — tabel kosong terbaca
 * sebagai "tidak ada isinya", padahal yang benar adalah "belum dapat dibaca".
 */
function GridBlock({ grid, rows }: { grid: Grid; rows: GridRow[] }) {
  if (grid.terhalang) {
    return (
      <div className="rounded-kartu border border-amber-200 bg-amber-50 px-4 py-3">
        <h3 className="text-sm font-semibold text-amber-900">
          {grid.judul} belum tersedia
        </h3>
        <p className="mt-1 text-sm text-slate-700">{grid.alasan_terhalang}</p>
        {grid.pemilik_penghalang && (
          <p className="mt-1 text-xs text-slate-600">
            <span className="font-medium">Menunggu:</span> {grid.pemilik_penghalang}
          </p>
        )}
      </div>
    )
  }

  /*
    Barisnya tidak punya kunci alami: dokumen klaim tidak membawa id per baris, dan dua
    baris spreading yang nilainya sama persis memang mungkin ada. Kunci berbasis isi karena
    itu tidak cukup — ia membuat React menemui dua kunci yang sama dalam satu daftar.

    Posisinya ditempelkan di sini, dan itu aman pada layar ini: urutan baris ditentukan
    server dan tidak pernah disusun ulang maupun disaring di layar — `hideSearch` dan
    `noSort` pada setiap kolom menjaganya tetap begitu.

    Namanya berawalan garis bawah ganda supaya tidak dapat bertabrakan dengan kunci kolom
    yang datang dari server.
  */
  const keyed = rows.map((row, position) => ({ ...row, __baris: String(position) }))

  return (
    <DataTable<GridRow>
      columns={columnsFor(grid)}
      rows={keyed}
      rowKey={(row) => `${grid.kode}-${row['__baris']}`}
      title={grid.judul}
      label={grid.judul}
      hideSearch
      showHeaderWhenEmpty
      emptyMessage="Tidak ada baris."
    />
  )
}

/**
 * columnsFor menyusun kolom tabel dari susunan yang ditetapkan server.
 *
 * Tidak ada kolom aksi: layar ini baca-saja.
 */
function columnsFor(grid: Grid): Column<GridRow>[] {
  return grid.kolom.map((column) => ({
    key: column.kunci,
    title: column.judul,
    value: (row) => cellText(row[column.kunci]),
    noSort: true,
  }))
}

/**
 * cellText menyusun teks satu sel.
 *
 * Tanggal diformat HANYA bila bentuknya memang `YYYY-MM-DD`. Nilainya dibaca dari dokumen
 * JSON yang bentuknya tidak dapat diperiksa (`R-08`), sehingga memaksakan pemformatan akan
 * mengubah nilai yang tidak dikenali menjadi teks yang salah — lebih buruk daripada
 * menampilkannya apa adanya.
 *
 * Nilai kosong menjadi tanda pisah, bukan sel kosong yang tidak dapat dibedakan dari kolom
 * yang gagal dimuat.
 */
function cellText(value: string | undefined): string {
  if (value === undefined || value === '') return '—'
  return isDate(value) ? formatDate(value) : value
}

/** isDate mengenali bentuk `YYYY-MM-DD`. */
function isDate(text: string): boolean {
  return /^\d{4}-\d{2}-\d{2}$/.test(text)
}

/** messageOf mengambil pesan yang layak dibaca pengguna dari sebuah galat. */
function messageOf(error: unknown): string {
  if (error instanceof APIError) return error.message
  return 'Coba lagi beberapa saat lagi. Bila terus berulang, hubungi tim teknis.'
}
