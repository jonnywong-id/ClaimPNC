import { useState } from 'react'

import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage } from '@/components/ErrorMessage'
import { SelectField } from '@/components/SelectField'

import { useBreakdown, useClaimSummary, useMasters } from './api'
import { messageOf } from './errors'
import type { Breakdown, ClaimSummary, MasterXOL } from './types'

/**
 * Tab "Inbox XOL" — pemilih perjanjian, grid akumulasi klaim, dan rincian per baris.
 *
 * # Susunannya diambil dari mana
 *
 * `Section/InboxClaimXOL-Section.xml`: grid "PILIH MASTER XOL" (Tahun · Kurs · Group
 * Business), grid "DATA XOL BASED ON DOL AND COL" (Date Of Loss · Cause Of Loss · Group
 * Business · OS Value (USD) · Accepted Value (USD)), dan `Sec_Detail_claim_XOL` sebagai
 * rincian yang terbuka di balik satu baris.
 *
 * Judul kolom TIDAK diterjemahkan — `D-13` menetapkan tampilan meniru Pega supaya
 * pengguna tidak perlu belajar ulang, dan itulah teks yang selama ini mereka baca.
 *
 * # Pemilih perjanjian menjadi dropdown, bukan grid
 *
 * Di sistem lama ia grid di dalam modal: pengguna menekan "INSERT DOL DAN COL", memilih
 * satu baris, lalu menekan "Pilih". Bentuk itu ada karena modal itu sekaligus tempat
 * MENAMBAH data DOL dan COL — dan penambahan itulah yang belum dipindahkan.
 *
 * Yang tersisa dari modal tersebut hanyalah pemilihan, dan pemilihan satu nilai dari
 * daftar pendek adalah dropdown. Ketiga kolomnya tetap terbaca: labelnya memuat tahun,
 * kurs, dan group business sekaligus.
 */
export function ClaimPanel() {
  const [masterID, setMasterID] = useState('')
  const [opened, setOpened] = useState<string | null>(null)

  const masters = useMasters()
  const summary = useClaimSummary(masterID)

  const options = (masters.data?.perjanjian ?? []).map((master) => ({
    value: master.id,
    label: masterLabel(master),
  }))

  return (
    <div className="mt-4 space-y-4">
      <div className="rounded-kartu border border-slate-200 bg-white p-4 shadow-lembut">
        <div className="max-w-xl">
          <SelectField
            id="perjanjian-xol"
            label="Perjanjian XOL"
            options={options}
            emptyText={masters.isPending ? '— memuat —' : '— pilih perjanjian —'}
            value={masterID}
            onChange={(event) => {
              setMasterID(event.target.value)
              // Rincian yang terbuka milik perjanjian SEBELUMNYA. Membiarkannya terbuka
              // akan menampilkan angka perjanjian lama di bawah judul perjanjian baru.
              setOpened(null)
            }}
          />
        </div>

        {masters.isError && (
          <div className="mt-3">
            <ErrorMessage
              title="Daftar perjanjian XOL tidak dapat dimuat"
              description={messageOf(masters.error)}
              tone="gangguan"
            />
          </div>
        )}
      </div>

      {masterID === '' ? (
        <p className="rounded-kartu border border-dashed border-slate-300 bg-white px-4 py-8 text-center text-sm text-slate-500">
          Pilih perjanjian XOL untuk melihat akumulasi klaimnya.
        </p>
      ) : (
        <ClaimTable
          masterID={masterID}
          master={summary.data?.perjanjian}
          rows={summary.data?.baris ?? []}
          loading={summary.isPending}
          error={summary.isError ? messageOf(summary.error) : null}
          opened={opened}
          onToggle={setOpened}
        />
      )}
    </div>
  )
}

/**
 * masterLabel merangkai ketiga kolom grid "PILIH MASTER XOL" menjadi satu label.
 *
 * Kursnya ikut disebut karena ia menentukan seluruh angka di grid berikutnya: nilai
 * "USD" yang tidak dapat ditelusuri kursnya tidak dapat diperiksa siapa pun.
 */
function masterLabel(master: MasterXOL): string {
  const groups = master.group_business || 'tanpa group business'
  return `${master.tahun} — ${groups} (kurs ${formatNumber(master.kurs)})`
}

type TableProps = {
  masterID: string
  master: MasterXOL | undefined
  rows: ClaimSummary[]
  loading: boolean
  error: string | null
  opened: string | null
  onToggle: (key: string | null) => void
}

function ClaimTable({ masterID, master, rows, loading, error, opened, onToggle }: TableProps) {
  const columns: Column<ClaimSummary>[] = [
    {
      key: 'tanggal_kejadian',
      title: 'Date Of Loss',
      value: (row) => row.tanggal_kejadian,
      width: '9rem',
    },
    {
      key: 'sebab_kerugian',
      title: 'Cause Of Loss',
      value: (row) => row.sebab_kerugian,
    },
    {
      key: 'group_business',
      title: 'Group Business',
      value: (row) => row.group_business,
    },
    {
      key: 'nilai_outstanding',
      title: 'OS Value',
      value: (row) => String(row.nilai_outstanding),
      render: (row) => formatNumber(row.nilai_outstanding),
      alignRight: true,
      width: '10rem',
    },
    {
      key: 'nilai_akseptasi',
      title: 'Accepted Value',
      value: (row) => String(row.nilai_akseptasi),
      render: (row) => formatNumber(row.nilai_akseptasi),
      alignRight: true,
      width: '10rem',
    },
    {
      key: 'aksi',
      title: 'Detail Data',
      noSort: true,
      alignRight: true,
      width: '8rem',
      value: () => '',
      render: (row) => {
        const key = rowKey(row)
        const open = opened === key
        return (
          <button
            type="button"
            className="rounded-kontrol px-2 py-1 text-sm font-medium text-blue-700 underline-offset-2 hover:underline focus-visible:outline-none focus-visible:ring-4"
            aria-expanded={open}
            onClick={() => onToggle(open ? null : key)}
          >
            {open ? 'Tutup rincian' : 'Lihat rincian'}
          </button>
        )
      },
    },
  ]

  const openedRow = rows.find((row) => rowKey(row) === opened) ?? null

  return (
    <div className="space-y-4">
      {master && <MasterSummary master={master} />}

      <DataTable<ClaimSummary>
        columns={columns}
        rows={rows}
        rowKey={rowKey}
        title="DATA XOL BASED ON DOL AND COL"
        description={
          master
            ? `Nilai dibagi kurs perjanjian (${formatNumber(master.kurs)}), mengikuti perhitungan sistem lama.`
            : 'Nilai dibagi kurs perjanjian, mengikuti perhitungan sistem lama.'
        }
        isLoading={loading}
        error={
          error ? (
            <ErrorMessage
              title="Akumulasi klaim tidak dapat dimuat"
              description={error}
              tone="gangguan"
            />
          ) : undefined
        }
        emptyMessage={emptyMessageFor(master)}
      />

      {openedRow && (
        <BreakdownTable
          masterID={masterID}
          lossDate={openedRow.tanggal_kejadian}
          cause={openedRow.sebab_kerugian}
        />
      )}
    </div>
  )
}

/**
 * emptyMessageFor membedakan dua sebab grid kosong yang tampak sama.
 *
 * Perjanjian yang BELUM diisi group business tidak akan pernah punya baris — dan itu
 * bukan "tidak ada klaim", melainkan master yang belum lengkap. Menyatukan keduanya
 * membuat pengguna menunggu data yang tidak akan pernah datang.
 */
function emptyMessageFor(master: MasterXOL | undefined): string {
  if (master && master.jumlah_group_business === 0) {
    return (
      'Perjanjian ini belum punya group business, sehingga tidak ada klaim yang dapat ' +
      'diakumulasi. Lengkapi Master XOL lebih dulu.'
    )
  }
  return 'Belum ada klaim XOL pada perjanjian ini.'
}

/** MasterSummary menampilkan keempat kolom kepala pada `Sec_Detail_claim_XOL`. */
function MasterSummary({ master }: { master: MasterXOL }) {
  const entries: Array<[string, string]> = [
    ['Type Master', master.tipe || '—'],
    ['ID Master', master.id],
    ['Tahun', master.tahun],
    ['Kurs IDR', formatNumber(master.kurs)],
  ]

  return (
    <dl className="grid grid-cols-2 gap-3 rounded-kartu border border-slate-200 bg-white p-4 shadow-lembut sm:grid-cols-4">
      {entries.map(([label, value]) => (
        <div key={label}>
          <dt className="text-xs uppercase tracking-wide text-slate-500">{label}</dt>
          <dd className="mt-0.5 text-sm font-medium text-slate-900">{value}</dd>
        </div>
      ))}
    </dl>
  )
}

function BreakdownTable({
  masterID,
  lossDate,
  cause,
}: {
  masterID: string
  lossDate: string
  cause: string
}) {
  const breakdown = useBreakdown(masterID, lossDate, cause)
  const rows = breakdown.data?.baris ?? []

  const columns: Column<Breakdown>[] = [
    {
      key: 'group_business',
      title: 'Group Business',
      value: (row) => row.group_business,
      render: (row) =>
        row.sumber === 'treaty' ? (
          <span>
            {row.group_business}
            <span className="ml-2 rounded-full bg-slate-100 px-2 py-0.5 text-xs font-medium text-slate-600 ring-1 ring-slate-200">
              treaty inward
            </span>
          </span>
        ) : (
          row.group_business
        ),
    },
    {
      key: 'jumlah_klaim',
      title: 'Total Klaim',
      value: (row) => String(row.jumlah_klaim),
      alignRight: true,
      width: '8rem',
    },
    {
      key: 'nilai_outstanding',
      title: 'OS Value',
      value: (row) => String(row.nilai_outstanding),
      render: (row) => valueCell(row, row.nilai_outstanding),
      alignRight: true,
      width: '10rem',
    },
    {
      key: 'nilai_akseptasi',
      title: 'Accept Value',
      value: (row) => String(row.nilai_akseptasi),
      render: (row) => valueCell(row, row.nilai_akseptasi),
      alignRight: true,
      width: '10rem',
    },
  ]

  return (
    <div>
      <DataTable<Breakdown>
        columns={columns}
        rows={rows}
        rowKey={(row) => `${row.sumber}-${row.kode_group_business || row.group_business}`}
        title={`Rincian — ${lossDate} · ${cause}`}
        isLoading={breakdown.isPending}
        error={
          breakdown.isError ? (
            <ErrorMessage
              title="Rincian tidak dapat dimuat"
              description={messageOf(breakdown.error)}
              tone="gangguan"
            />
          ) : undefined
        }
        emptyMessage="Tidak ada rincian untuk tanggal dan penyebab kerugian ini."
      />

      {rows.some((row) => row.kurs_tidak_tersedia) && <MissingRateNotice />}
    </div>
  )
}

/**
 * valueCell menolak menampilkan angka yang kursnya tidak diketahui.
 *
 * Di sistem lama nilai seperti ini tampil sebagai angka biasa, karena fungsi kursnya
 * mengembalikan `1` saat kurs tidak ditemukan. Angka yang salah dan tampak benar jauh
 * lebih berbahaya daripada tanda hubung.
 */
function valueCell(row: Breakdown, value: number) {
  if (row.kurs_tidak_tersedia) {
    return <span className="text-slate-400">—</span>
  }
  return formatNumber(value)
}

function MissingRateNotice() {
  return (
    <p className="mt-2 rounded-kartu border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-900">
      Sebagian baris treaty inward tidak dapat dihitung karena kurs mata uangnya pada
      tanggal kejadian tidak ada di master kurs. Nilainya sengaja tidak ditampilkan —
      lengkapi master kurs lebih dulu.
    </p>
  )
}

/**
 * rowKey menyusun kunci baris dari kedua kolom yang membentuk pengelompokannya.
 *
 * Tanggal saja tidak cukup: satu tanggal dapat punya beberapa penyebab kerugian, dan
 * kunci yang sama pada dua baris membuat React menggambar salah satunya saja.
 */
function rowKey(row: ClaimSummary): string {
  return `${row.tanggal_kejadian}|${row.sebab_kerugian}`
}

/**
 * formatNumber menampilkan angka dengan pemisah ribuan Indonesia.
 *
 * Ia TIDAK memakai formatRupiah dari `shared`: nilai di layar ini bukan rupiah melainkan
 * mata uang perjanjian, dan menempelkan "Rp" padanya akan menyatakan hal yang salah.
 * Desimalnya dibatasi dua — nilai hasil pembagian kurs nyaris selalu berkoma panjang.
 */
function formatNumber(value: number): string {
  return new Intl.NumberFormat('id-ID', { maximumFractionDigits: 2 }).format(value)
}
