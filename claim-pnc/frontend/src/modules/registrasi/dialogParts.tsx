import type { Dispatch, ReactNode, SetStateAction, TextareaHTMLAttributes } from 'react'

import { ErrorMessage } from '@/components/ErrorMessage'

/** Teks galat dialog: pesan pelanggaran aturan bila ada, selain itu pesan galatnya sendiri. */
export function failureDescription(violations: readonly { pesan: string }[], failure: unknown) {
  if (violations.length > 0) return violations.map((v) => v.pesan).join(' ')
  if (failure instanceof Error) return failure.message
  return 'Terjadi kesalahan pada sistem.'
}

/** DialogFailure menggambar kotak penolakan dialog bila ada galat; selain itu tidak apa-apa. */
export function DialogFailure({
  title,
  failure,
  violations,
}: Readonly<{
  title: string
  failure: unknown
  violations: readonly { pesan: string }[]
}>) {
  if (!failure) return null
  return (
    <div className="mt-4">
      <ErrorMessage title={title} description={failureDescription(violations, failure)} tone="penolakan" />
    </div>
  )
}

/** DialogSuccess menggambar pesan berhasil berwarna hijau bila `show`. */
export function DialogSuccess({ show, children }: Readonly<{ show: boolean; children: ReactNode }>) {
  if (!show) return null
  return <output className="mt-4 block text-sm text-emerald-700">{children}</output>
}

/** Satu baris pemberitahuan kerugian (PLA atau DLA) yang digambar di grid dialog. */
type LossAdviceRow = {
  nomor: string
  penerima: string
  tipe: string
  email: string
}

/**
 * LossAdviceTable adalah grid dialog Print PLA dan Print DLA: NO, REINSURER, TIPE, REMARKS
 * (dapat diubah), EMAIL, lalu kolom tombol per baris. Kedua section layar lama berbentuk
 * sama; yang berbeda — judul kolom, sifat isian REMARKS, dan tombolnya — menjadi prop.
 */
export function LossAdviceTable<R extends LossAdviceRow>({
  kind,
  emailTitle,
  rows,
  notes,
  setNotes,
  remarksProps,
  renderActions,
}: Readonly<{
  /** "PLA" atau "DLA" — dipakai pada keterangan tabel dan judul kolom. */
  kind: string
  emailTitle: string
  rows: R[]
  notes: Record<string, string>
  setNotes: Dispatch<SetStateAction<Record<string, string>>>
  /** Atribut tambahan isian REMARKS per baris (readOnly, disabled, className). */
  remarksProps: (row: R) => TextareaHTMLAttributes<HTMLTextAreaElement>
  renderActions: (row: R) => ReactNode
}>) {
  return (
    <table className="mt-4 w-full border-collapse text-sm">
      <caption className="sr-only">{`Daftar ${kind}`}</caption>
      <thead>
        <tr className="bg-slate-100 text-left text-xs text-slate-700">
          <th className="p-2">{`NO ${kind}`}</th>
          <th className="p-2">{`${kind} REINSURER`}</th>
          <th className="p-2">{`TIPE ${kind}`}</th>
          <th className="p-2">REMARKS</th>
          <th className="p-2">{emailTitle}</th>
          <th className="p-2" />
        </tr>
      </thead>
      <tbody>
        {rows.map((r) => (
          <tr key={r.nomor} className="border-b border-slate-100 align-top">
            <td className="p-2 font-mono text-xs">{r.nomor}</td>
            <td className="p-2">{r.penerima}</td>
            <td className="p-2">{r.tipe}</td>
            <td className="p-2">
              <textarea
                aria-label={`Remarks ${r.nomor}`}
                rows={3}
                value={notes[r.nomor] ?? ''}
                onChange={(e) => setNotes((n) => ({ ...n, [r.nomor]: e.target.value }))}
                {...remarksProps(r)}
              />
            </td>
            <td className="p-2 text-xs text-slate-600">{r.email || '—'}</td>
            <td className="p-2">{renderActions(r)}</td>
          </tr>
        ))}
      </tbody>
    </table>
  )
}
