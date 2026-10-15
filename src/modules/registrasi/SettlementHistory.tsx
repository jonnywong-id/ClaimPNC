import { formatDateTimeWIB } from '@/components/format'

import { useSettlementHistory } from './api'

/** Label kolom Status grid komite — kode STATUSAPPROVE T_CLAIM_KOMITE_LIST. */
const COMMITTEE_DECISION: Record<string, string> = { '0': 'Menunggu', '1': 'Setuju', '2': 'Tolak' }

type Props = { claimID: string; object: number; coverage: number; adjustment: number }

/**
 * SettlementHistory menggambar dua grid di bawah rincian satu baris Adjustment, mengikuti
 * Section/InputAdjustment_sect.xml:
 *
 * - **Status Penerimaan Komite** (.ComiteeClaim): Nama Komite, Status, Tanggal
 *   Approve/Reject, Komentar. Di XML grid ini hanya tampil untuk Bonding Interim; Work Owner
 *   (2026-10-07) menetapkan ia tampil di SEMUA lini.
 * - **Histori Transfer Kasir** (TempDataLogKasir): PIC Teknik, Tanggal Transfer/Reject,
 *   Status Kasir, Komentar — tampil bila ada log (`pxResults(1).KomiteAccepted != ''`).
 *
 * Grid yang tidak berisi tidak digambar, sama seperti kondisi tampil XML. Keduanya baca saja.
 */
export function SettlementHistory({ claimID, object, coverage, adjustment }: Props) {
  const history = useSettlementHistory(claimID, object, coverage, adjustment)
  if (history.isError) {
    return <p className="mt-4 text-xs text-amber-700">Riwayat komite dan transfer kasir tidak dapat dimuat.</p>
  }
  const committee = history.data?.komite ?? []
  const cashier = history.data?.kasir ?? []
  if (committee.length === 0 && cashier.length === 0) return null

  return (
    <div className="mt-5 space-y-5">
      {committee.length > 0 && (
        <HistoryTable
          title="Status Penerimaan Komite"
          headers={['Nama Komite', 'Status', 'Tanggal Approve/Reject', 'Komentar']}
          rows={committee.map((m) => [
            m.nama_komite,
            COMMITTEE_DECISION[m.status] ?? m.status,
            formatDateTimeWIB(m.tanggal),
            m.komentar,
          ])}
        />
      )}
      {cashier.length > 0 && (
        <HistoryTable
          title="Histori Transfer Kasir"
          headers={['PIC Teknik', 'Tanggal Transfer/Reject', 'Status Kasir', 'Komentar']}
          rows={cashier.map((c) => [c.pic_teknik, formatDateTimeWIB(c.tanggal), c.status_kasir, c.komentar])}
        />
      )}
    </div>
  )
}

function HistoryTable({ title, headers, rows }: { title: string; headers: string[]; rows: string[][] }) {
  return (
    <section>
      <h4 className="mb-2 text-sm font-semibold text-slate-800">{title}</h4>
      <table className="w-full border-collapse border border-slate-200 bg-white text-xs">
        <caption className="sr-only">{title}</caption>
        <thead>
          <tr className="bg-slate-100 text-left text-slate-700">
            {headers.map((h) => (
              <th key={h} scope="col" className="border border-slate-200 p-2">
                {h}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {rows.map((row, i) => (
            <tr key={i}>
              {row.map((cell, j) => (
                <td key={j} className="border border-slate-200 p-2">
                  {cell?.trim() || '—'}
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </section>
  )
}
