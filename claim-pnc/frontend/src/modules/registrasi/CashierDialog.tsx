import { useEffect, useRef } from 'react'

import { Button } from '@/components/Button'
import { ErrorMessage } from '@/components/ErrorMessage'

import { useCashierPreview, useTransferCashier, violationsFrom } from './api'

const amountFormatter = new Intl.NumberFormat('id-ID', { minimumFractionDigits: 2, maximumFractionDigits: 2 })

/**
 * Dialog Transfer Kasir. Flow action Pega di baliknya (`ValidasiTransferKasir_dialog`) tidak ada
 * di export; atas keputusan Work Owner, dialog ini menampilkan ringkasan konfirmasi — penerima,
 * rekening, bank, nilai nett, dan nomor akseptasi — lalu Submit menjalankan validasi
 * `TransferToKasir_act` dan mengirim pembayaran ke sistem Kasir.
 */
export function CashierDialog({
  claimID,
  address,
  onClose,
}: {
  claimID: string
  address: { tugas_id: string; objek: number; jaminan: number; adjustment: number }
  onClose: () => void
}) {
  const preview = useCashierPreview(claimID)
  const transfer = useTransferCashier(claimID)

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
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/40 px-4 py-8"
      role="dialog"
      aria-modal="true"
      aria-labelledby="judul-transfer-kasir"
    >
      <div className="max-h-full w-full max-w-lg overflow-y-auto rounded-kartu bg-white p-6 shadow-angkat">
        <h2 id="judul-transfer-kasir" className="text-lg font-semibold text-slate-900">
          Transfer Kasir
        </h2>

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
        {data?.masalah && !done && (
          <p className="mt-4 text-sm text-amber-700" role="status">
            {data.masalah}
          </p>
        )}

        {failure && (
          <div className="mt-4">
            <ErrorMessage
              title="Transfer Kasir belum dapat diproses"
              description={
                violations.length > 0
                  ? violations.map((v) => v.pesan).join(' ')
                  : failure instanceof Error
                    ? failure.message
                    : 'Terjadi kesalahan pada sistem.'
              }
              tone="penolakan"
            />
          </div>
        )}
        {done && (
          <p className="mt-4 text-sm text-emerald-700" role="status">
            Berhasil ditransfer ke Kasir.
          </p>
        )}

        <div className="mt-6 flex justify-end gap-3">
          <Button tone="halus" disabled={busy} onClick={onClose}>
            {done ? 'Tutup' : 'Cancel'}
          </Button>
          {!done && (
            <Button
              tone="utama"
              disabled={busy || !data || data.masalah !== ''}
              onClick={() => {
                transfer.reset()
                transfer.mutate(address)
              }}
            >
              {transfer.isPending ? 'Mengirim…' : 'Submit'}
            </Button>
          )}
        </div>
      </div>
    </div>
  )
}
