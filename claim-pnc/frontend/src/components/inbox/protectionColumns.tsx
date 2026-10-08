import type { Column } from '@/components/DataTable'
import { formatDate } from '@/components/format'

/** Bentuk minimal satu baris permintaan proteksi (Input Req Protection & akseptasinya). */
export type ProtectionRowLike = {
  nomor_polis: string
  tanggal_proteksi: string
  keterangan: string
  user_create: string
}

/** policyNumberColumn menyusun kolom "No Polis" pada daftar proteksi. */
export function policyNumberColumn<T extends ProtectionRowLike>(): Column<T> {
  return {
    key: 'nomor_polis',
    title: 'No Polis',
    width: '12rem',
    value: (p) => p.nomor_polis,
    render: (p) => (
      <span className="truncate font-mono text-xs text-slate-700">{p.nomor_polis || '—'}</span>
    ),
  }
}

/**
 * protectionTrailingColumns menyusun tiga kolom penutup daftar proteksi: tanggal dibuat,
 * keterangan, dan pembuatnya — sama di layar permintaan maupun layar akseptasinya.
 */
export function protectionTrailingColumns<T extends ProtectionRowLike>(): Column<T>[] {
  return [
    {
      key: 'tanggal_proteksi',
      title: 'Tanggal Proteksi Dibuat',
      width: '10rem',
      value: (p) => p.tanggal_proteksi,
      render: (p) => (
        <span className="tabular-nums">
          {p.tanggal_proteksi ? formatDate(p.tanggal_proteksi) : '—'}
        </span>
      ),
    },
    {
      key: 'keterangan',
      title: 'Keterangan',
      value: (p) => p.keterangan,
      render: (p) => (
        <span className="truncate" title={p.keterangan}>
          {p.keterangan || '—'}
        </span>
      ),
    },
    {
      key: 'user_create',
      title: 'User Create',
      width: '10rem',
      value: (p) => p.user_create,
      render: (p) => <span className="truncate">{p.user_create || '—'}</span>,
    },
  ]
}
