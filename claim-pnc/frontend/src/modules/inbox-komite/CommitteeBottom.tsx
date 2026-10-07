import type {
  KomiteAttachment,
  KomiteBreakdownRow,
  KomiteCommitteeEntry,
  KomiteTransferDetail,
} from '@/api/types'
import { DataTable, type Column } from '@/components/DataTable'
import { formatRupiah } from '@/lib/money'

import { formatTanggal, formatWaktu } from './format'

/**
 * Bagian bawah `Section/ShowTransferDetail`, urutannya sama dengan layar Pega:
 * Claim Adjustment · Dokumen Pendukung / Dokumen Polis · History of Previous Adjustment
 * Committees · Daftar Komite.
 *
 * Seluruh angka dan teks status disusun server (`internal/komite/breakdown.go`); layar hanya
 * memformat dan menggambarnya.
 */
export function CommitteeBottom({ transfer }: Readonly<{ transfer: KomiteTransferDetail }>) {
  return (
    <div className="mt-5 space-y-5">
      {transfer.baris.map((line, i) =>
        line.rincian ? (
          <BreakdownTable
            key={`${line.id_objek}-${line.id_coverage}-${i}`}
            title={line.rincian.judul}
            valueHeader={line.rincian.judul_nilai}
            known={line.rincian.tersedia}
            rows={line.rincian.baris}
            label={
              transfer.baris.length > 1
                ? `Objek ${line.id_objek || '—'} · Coverage ${line.id_coverage || '—'}`
                : undefined
            }
          />
        ) : null,
      )}
      <Documents attachments={transfer.lampiran ?? []} />
      <EntryTable
        title="History of Previous Adjustment Committees"
        withCase
        entries={transfer.riwayat_komite ?? []}
        dateTitle="Tanggal Komite"
        emptyMessage="Belum ada komite adjustment sebelumnya untuk klaim ini."
      />
      <EntryTable
        title="Daftar Komite"
        entries={transfer.daftar_komite ?? []}
        dateTitle="Tanggal Akseptasi Komite"
        emptyMessage="Case ini belum punya anggota komite."
      />
    </div>
  )
}

function money(v: string | undefined): string {
  return v ? formatRupiah(v, { withoutSymbol: true }) : ''
}

function bold(row: KomiteBreakdownRow, text: string) {
  return row.total ? <strong>{text}</strong> : text
}

/**
 * Tabel Claim Adjustment — kolom DESCRIPTION · CURRENCY · ESTIMATION · % · CURRENCY ·
 * ADJUSTMENT (atau CLAIM ACCEPTED pada Travel). Sel kosong digambar kosong, persis Pega.
 */
function BreakdownTable({
  title,
  valueHeader,
  known,
  rows,
  label,
}: Readonly<{
  title: string
  valueHeader: string
  known: boolean
  rows: KomiteBreakdownRow[]
  label?: string | undefined
}>) {
  if (!known) {
    return (
      <p className="rounded-kartu border border-dashed border-slate-300 p-3 text-sm text-slate-600">
        {title}: tabel nilai untuk jenis pembayaran ini belum dibangun.
      </p>
    )
  }
  const columns: Column<KomiteBreakdownRow>[] = [
    {
      key: 'deskripsi',
      title: 'Description',
      value: (r) => r.deskripsi,
      render: (r) => bold(r, r.deskripsi),
      noSort: true,
    },
    { key: 'mu_est', title: 'Currency', value: (r) => r.mata_uang_estimasi ?? '', noSort: true },
    {
      key: 'estimasi',
      title: 'ESTIMATION',
      value: (r) => r.keterangan_estimasi || money(r.estimasi),
      alignRight: true,
      noSort: true,
    },
    {
      key: 'persen',
      title: '%',
      value: (r) => r.persen ?? '',
      render: (r) => bold(r, r.persen ?? ''),
      alignRight: true,
      noSort: true,
    },
    {
      key: 'mu',
      title: 'Currency',
      value: (r) => r.mata_uang ?? '',
      render: (r) => bold(r, r.mata_uang ?? ''),
      noSort: true,
    },
    {
      key: 'nilai',
      title: valueHeader,
      value: (r) => money(r.nilai),
      render: (r) => bold(r, money(r.nilai)),
      alignRight: true,
      noSort: true,
    },
  ]
  return (
    <DataTable
      title={title}
      {...(label ? { description: label } : {})}
      columns={columns}
      rows={rows}
      rowKey={(r) => r.deskripsi + (r.total ? '-total' : '') + (r.persen ?? '') + (r.nilai ?? '')}
      hideSearch
      searchable={false}
    />
  )
}

const documentColumns: Column<KomiteAttachment>[] = [
  { key: 'kategori', title: 'Kategori', value: (r) => r.catatan || r.kategori || '' },
  { key: 'nama', title: 'Nama', value: (r) => r.nama ?? '' },
  { key: 'tanggal', title: 'Tanggal', value: (r) => formatWaktu(r.diunggah_pada) },
]

/**
 * Dokumen Pendukung dan Dokumen Polis.
 *
 * Pega menggambar Dokumen Pendukung lewat `PNCViewAttachmentKomite` — section yang TIDAK ada
 * di export — sehingga isinya diambil dari lampiran klaim (`DATA_ATTACHFILE`). Dokumen Polis
 * dibaca Pega lewat layanan (`GetFilePolisByServiceKlaim`) dan belum dibawa.
 */
function Documents({ attachments }: Readonly<{ attachments: KomiteAttachment[] }>) {
  return (
    <div className="grid gap-5 lg:grid-cols-2">
      <DataTable
        title="Dokumen Pendukung"
        columns={documentColumns}
        rows={attachments}
        rowKey={(r) => r.id}
        emptyMessage="Data Tidak Ada"
        showHeaderWhenEmpty
        hideSearch
        searchable={false}
      />
      <DataTable
        title="Dokumen Polis"
        description="Dokumen polis dibaca Pega dari layanan GetFilePolisByServiceKlaim, yang belum tersedia di sini."
        columns={documentColumns.slice(0, 2)}
        rows={[]}
        rowKey={(r) => r.id}
        emptyMessage="Data Tidak Ada"
        showHeaderWhenEmpty
        hideSearch
        searchable={false}
      />
    </div>
  )
}

function EntryTable({
  title,
  entries,
  withCase = false,
  dateTitle,
  emptyMessage,
}: Readonly<{
  title: string
  entries: KomiteCommitteeEntry[]
  withCase?: boolean
  dateTitle: string
  emptyMessage: string
}>) {
  const columns: Column<KomiteCommitteeEntry>[] = [
    ...(withCase
      ? [{ key: 'case', title: 'Komite ID', value: (r: KomiteCommitteeEntry) => r.nomor_case }]
      : []),
    { key: 'komite', title: 'Komite', value: (r) => r.nama_komite },
    { key: 'status', title: 'Status', value: (r) => r.status },
    { key: 'catatan', title: 'Catatan', value: (r) => r.catatan ?? '' },
    { key: 'tanggal', title: dateTitle, value: (r) => formatTanggal(r.tanggal) },
  ]
  return (
    <DataTable
      title={title}
      columns={columns}
      rows={entries}
      rowKey={(r) => `${r.nomor_case}-${r.jenjang}-${r.nama_komite}`}
      emptyMessage={emptyMessage}
      showHeaderWhenEmpty
      hideSearch
      searchable={false}
    />
  )
}
