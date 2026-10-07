import { useEffect, useRef, useState } from 'react'

import { Button } from '@/components/Button'
import { ErrorMessage } from '@/components/ErrorMessage'

import { useCashierPreview, useTransferCashier, violationsFrom } from './api'

const amountFormatter = new Intl.NumberFormat('id-ID', { minimumFractionDigits: 2, maximumFractionDigits: 2 })

/** Teks galat dialog: pesan pelanggaran aturan bila ada, selain itu pesan galatnya sendiri. */
function failureDescription(violations: readonly { pesan: string }[], failure: unknown) {
  if (violations.length > 0) return violations.map((v) => v.pesan).join(' ')
  if (failure instanceof Error) return failure.message
  return 'Terjadi kesalahan pada sistem.'
}

/**
 * Dialog "Transfer Pembayaran" — flow action `ValidasiTransferKasir_dialog` (section
 * `Sec_dialogValidasiTransfer`, pra-proses `Pre_AlertTransferkasir`). Judul, kalimat konfirmasi,
 * dan tombol "Transfer To Kasir" / "Batal" mengikuti Pega; ringkasan penerima, rekening, bank, dan
 * nilai nett ditambahkan atas keputusan Work Owner. Submit menjalankan validasi
 * `TransferToKasir_act` dan mengirim pembayaran ke sistem Kasir.
 *
 * "Tipe Transfer Kasir" (`.JoinPlacement`: Pembayaran Biasa, Join Placement, Fronting). Untuk Join
 * Placement dan Fronting tabel DLA FAC OUT tampil (`GetdataFacoutJoinPlacement`); yang dicentang
 * "Pilih Fac-out Tidak Dibayar" dikirim ke Kasir sebagai baris bernilai negatif.
 */
export function CashierDialog({
  claimID,
  address,
  onClose,
}: Readonly<{
  claimID: string
  address: { tugas_id: string; objek: number; jaminan: number; adjustment: number }
  onClose: () => void
}>) {
  const preview = useCashierPreview(claimID)
  const transfer = useTransferCashier(claimID)
  const [kind, setKind] = useState('1')
  const [unpaid, setUnpaid] = useState<string[]>([])

  const opened = useRef(false)
  useEffect(() => {
    if (opened.current) return
    opened.current = true
    preview.mutate(address)
  })

  const busy = preview.isPending || transfer.isPending
  useEffect(() => {
    function onKey(event: KeyboardEvent) {
      if (event.key === 'Escape' && !busy) onClose()
    }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [onClose, busy])

  const data = preview.data
  const failure = transfer.error ?? preview.error
  const violations = violationsFrom(failure)
  const done = transfer.isSuccess

  return (
    <dialog
      open
      className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/40 px-4 py-8"
      aria-modal="true"
      aria-labelledby="judul-transfer-kasir"
    >
      <div className="max-h-full w-full max-w-lg overflow-y-auto rounded-kartu bg-white p-6 shadow-angkat">
        <h2 id="judul-transfer-kasir" className="text-lg font-semibold text-slate-900">
          Transfer Pembayaran
        </h2>
        {data?.konfirmasi && <p className="mt-2 text-sm font-medium text-slate-800">{data.konfirmasi}</p>}

        {preview.isPending && <p className="mt-4 text-sm text-slate-500">Menyiapkan data…</p>}
        {data && (
          <dl className="mt-4 grid grid-cols-[10rem_1fr] gap-x-3 gap-y-1 text-sm">
            <dt className="text-slate-500">No Akseptasi</dt>
            <dd className="font-mono">{data.nomor_akseptasi}</dd>
            <dt className="text-slate-500">Penerima</dt>
            <dd>{data.penerima || '—'}</dd>
            <dt className="text-slate-500">No Rekening</dt>
            <dd>{data.nomor_rekening || '—'}</dd>
            <dt className="text-slate-500">Bank</dt>
            <dd>{data.nama_bank || '—'}</dd>
            <dt className="text-slate-500">Email</dt>
            <dd>{data.email || '—'}</dd>
            <dt className="text-slate-500">Nilai Nett</dt>
            <dd>
              {data.mata_uang} {amountFormatter.format(data.nilai_nett_sen / 100)}
            </dd>
          </dl>
        )}
        {data && !done && (
          <div className="mt-4 space-y-3 text-sm">
            <label className="block">
              <span className="font-medium text-slate-700">Tipe Transfer Kasir</span>
              <select
                value={kind}
                onChange={(e) => setKind(e.target.value)}
                className="mt-1 block w-full rounded border border-slate-300 px-2 py-1 text-sm"
              >
                {(data.tipe_transfer ?? []).map((t) => (
                  <option key={t.id} value={t.id}>
                    {t.nama}
                  </option>
                ))}
              </select>
            </label>
            {(kind === '2' || kind === '3') && (
              <table className="w-full border-collapse border border-slate-200 text-xs">
                <caption className="sr-only">Pilih Fac Fronting/Joinplacement</caption>
                <thead>
                  <tr className="bg-slate-100 text-left text-slate-700">
                    <th className="p-1.5">No DLA</th>
                    <th className="p-1.5">Nama Facout</th>
                    <th className="p-1.5 text-right">Nilai Bayar</th>
                    <th className="p-1.5 text-center">Pilih Fac-out Tidak Dibayar</th>
                  </tr>
                </thead>
                <tbody>
                  {(data.fac_out ?? []).length === 0 ? (
                    <tr>
                      <td colSpan={4} className="p-2 text-center text-slate-500">
                        Tidak ada DLA FAC OUT.
                      </td>
                    </tr>
                  ) : (
                    data.fac_out.map((f) => (
                      <tr key={f.nomor_dla} className="border-t border-slate-200">
                        <td className="p-1.5 font-mono">{f.nomor_dla}</td>
                        <td className="p-1.5">{f.nama_facout}</td>
                        <td className="p-1.5 text-right">
                          {f.mata_uang} {amountFormatter.format(Number(f.nilai_bayar))}
                        </td>
                        <td className="p-1.5 text-center">
                          <input
                            type="checkbox"
                            aria-label={`Fac-out ${f.nomor_dla} tidak dibayar`}
                            checked={unpaid.includes(f.nomor_dla)}
                            onChange={(e) =>
                              setUnpaid((v) => (e.target.checked ? [...v, f.nomor_dla] : v.filter((n) => n !== f.nomor_dla)))
                            }
                          />
                        </td>
                      </tr>
                    ))
                  )}
                </tbody>
              </table>
            )}
          </div>
        )}
        {data?.masalah && !done && (
          <output className="mt-4 block text-sm text-amber-700">
            {data.masalah}
          </output>
        )}

        {failure && (
          <div className="mt-4">
            <ErrorMessage
              title="Transfer Kasir belum dapat diproses"
              description={failureDescription(violations, failure)}
              tone="penolakan"
            />
          </div>
        )}
        {done && (
          <output className="mt-4 block text-sm text-emerald-700">
            Berhasil ditransfer ke Kasir.
          </output>
        )}

        <div className="mt-6 flex justify-end gap-3">
          <Button tone="halus" disabled={busy} onClick={onClose}>
            {done ? 'Tutup' : 'Batal'}
          </Button>
          {!done && (
            <Button
              tone="utama"
              disabled={busy || data?.masalah !== ''}
              onClick={() => {
                transfer.reset()
                const facOut = kind === '2' || kind === '3' ? unpaid : []
                transfer.mutate({ ...address, tipe_transfer: kind, fac_out_tidak_dibayar: facOut })
              }}
            >
              {transfer.isPending ? 'Mengirim…' : 'Transfer To Kasir'}
            </Button>
          )}
        </div>
      </div>
    </dialog>
  )
}
