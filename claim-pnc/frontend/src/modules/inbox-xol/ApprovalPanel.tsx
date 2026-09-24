import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage } from '@/components/ErrorMessage'

import { useApprovals } from './api'
import { messageOf } from './errors'
import type { ApprovalItem, MasterXOL } from './types'

/**
 * Tab "Inbox XOL Komite" — dua antrean persetujuan yang berdiri sendiri.
 *
 * Kolomnya diambil apa adanya dari `Section/InboxClaimXOL-Section.xml`:
 *
 *	Approval XOL     Date Of Loss · Cause Of Loss · TIPE · Tanggal Insert
 *	DATA MASTER XOL  ID XOL · Nama XOL · Tahun XOL · Kurs Value
 *
 * # Judul kolom pertama MENYESATKAN, dan tetap dipakai
 *
 * Kolom "Date Of Loss" pada grid Approval berisi TAHUN perjanjian, bukan tanggal
 * kejadian — kuerinya `select tahun as "City"`. `D-13` menetapkan tampilan meniru Pega
 * supaya pengguna tidak perlu belajar ulang, jadi judulnya tidak diubah. Yang dibetulkan
 * adalah namanya di kode, dan keterangan di bawah tabel menyebutkan isinya.
 *
 * # Tab ini seharusnya dijaga peran, dan hari ini belum
 *
 * Section lama menampilkannya hanya untuk `GCNMFW:CaseManager`. Tabel peran adalah
 * `TKT-F3-004`, yang dapat dibangun tetapi belum dapat diisi (`11-SECURITY.md` §3.1).
 * Selama modul ini membaca saja, taruhannya terbatas — antrean yang terlihat bukan
 * antrean yang dapat disetujui.
 */
export function ApprovalPanel({ active }: { active: boolean }) {
  const approvals = useApprovals(active)

  const advices = approvals.data?.pemberitahuan ?? []
  const masters = approvals.data?.perjanjian ?? []

  const error = approvals.isError ? (
    <ErrorMessage
      title="Antrean persetujuan tidak dapat dimuat"
      description={messageOf(approvals.error)}
      tone="gangguan"
    />
  ) : undefined

  return (
    <div className="mt-4 space-y-6">
      <div>
        <DataTable<ApprovalItem>
          columns={adviceColumns}
          rows={advices}
          rowKey={(row) => `${row.tipe}|${row.tahun}|${row.sebab_kerugian}`}
          title="Approval XOL"
          description="Satu baris mewakili sekumpulan pemberitahuan — satu tahun, satu penyebab kerugian, satu tipe."
          isLoading={approvals.isPending && active}
          error={error}
          emptyMessage="Tidak ada pemberitahuan yang menunggu persetujuan."
        />
        <p className="mt-2 text-xs text-slate-500">
          Kolom <span className="font-medium">Date Of Loss</span> berisi tahun perjanjian,
          bukan tanggal kejadian. Judulnya dipertahankan seperti di aplikasi lama.
        </p>
      </div>

      <DataTable<MasterXOL>
        columns={masterColumns}
        rows={masters}
        rowKey={(row) => row.id}
        title="DATA MASTER XOL"
        description="Perjanjian XOL yang diajukan dan masih menunggu persetujuan komite."
        isLoading={approvals.isPending && active}
        error={error}
        emptyMessage="Tidak ada perjanjian XOL yang menunggu persetujuan."
      />

      <WriteNotice />
    </div>
  )
}

/**
 * WriteNotice menyatakan mengapa tidak ada tombol menyetujui di layar ini.
 *
 * Tanpa keterangan ini, tab yang menampilkan antrean persetujuan tanpa satu pun tombol
 * akan terbaca sebagai layar yang rusak — dan yang dilaporkan pengguna adalah tombol
 * hilang, bukan keadaan yang sebenarnya.
 */
function WriteNotice() {
  return (
    <p className="rounded-kartu border border-slate-200 bg-slate-50 px-4 py-3 text-sm text-slate-600">
      Menyetujui atau menolak belum tersedia di aplikasi baru. Selama masa paralel,
      perubahan data XOL masih dilakukan lewat aplikasi Pega — daftar di atas hanya
      menampilkan keadaannya.
    </p>
  )
}

const adviceColumns: Column<ApprovalItem>[] = [
  {
    key: 'tahun',
    title: 'Date Of Loss',
    value: (row) => row.tahun,
    width: '9rem',
  },
  {
    key: 'sebab_kerugian',
    title: 'Cause Of Loss',
    value: (row) => row.sebab_kerugian,
  },
  {
    key: 'tipe',
    title: 'TIPE',
    value: (row) => row.tipe,
    width: '6rem',
  },
  {
    key: 'tanggal_insert',
    title: 'Tanggal Insert',
    value: (row) => row.tanggal_insert,
    width: '10rem',
  },
]

const masterColumns: Column<MasterXOL>[] = [
  {
    key: 'id',
    title: 'ID XOL',
    value: (row) => row.id,
    width: '9rem',
  },
  {
    key: 'nama',
    title: 'Nama XOL',
    value: (row) => row.nama,
  },
  {
    key: 'tahun',
    title: 'Tahun XOL',
    value: (row) => row.tahun,
    width: '8rem',
  },
  {
    key: 'kurs',
    title: 'Kurs Value',
    value: (row) => String(row.kurs),
    render: (row) => new Intl.NumberFormat('id-ID').format(row.kurs),
    alignRight: true,
    width: '9rem',
  },
]
