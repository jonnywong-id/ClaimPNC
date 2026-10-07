import { useState } from 'react'

import { simpanBerkas } from '@/api/client'
import { ErrorMessage } from '@/components/ErrorMessage'

import { usePrintAcceptanceNote, violationsFrom } from './api'
import { CashierDialog } from './CashierDialog'
import { PaymentType, type Settlement } from './types'

/**
 * Nomor Akseptasi beserta tombol PRINT dan Transfer Kasir di detail adjustment
 * (`Section/InputAdjustment_sect.xml`) — bukan di grid, seperti layar Pega.
 *
 * # Kapan tampil — disalin dari section apa adanya
 *
 *   Nomor Akseptasi  tampil bila `.AcceptationStatusLOD == '1' && .AcceptedNo != ''`
 *   PRINT            tampil bila `.AcceptedNo != ''` — `PrintPDFAcceptanceNote`, Draft
 *                    Persetujuan diunduh langsung (keputusan 2026-09-30, sama dengan Print DLA)
 *   Transfer Kasir   tampil bila `.PaymentType != '3' && .AcceptedNo != ''`; mati bila
 *                    `.TransferCashierStatus == '1'` (di sini: sudah pernah ditransfer)
 *
 * Syarat hostname `pega.simaspenjaminan.com` pada Transfer Kasir tidak dibawa (`D-75`): entitas
 * itu bukan salah satu portal aplikasi ini.
 */
export function AcceptanceNumber({
  claimID,
  taskID,
  object,
  coverage,
  adjustment,
  line,
}: Readonly<{
  claimID: string
  taskID: string
  /** Objek, jaminan, dan adjustment berbasis 1. */
  object: number
  coverage: number
  adjustment: number
  line: Settlement
}>) {
  const [cashier, setCashier] = useState(false)
  const print = usePrintAcceptanceNote(claimID)
  const rules = acceptanceNumberRules(line)
  if (!rules.visible) return null

  const address = { tugas_id: taskID, objek: object, jaminan: coverage, adjustment }
  const violations = violationsFrom(print.error)

  return (
    <div>
      <span className="font-semibold text-slate-800">Nomor Akseptasi</span>
      <div className="mt-0.5 flex flex-wrap items-center gap-2">
        <span className="font-mono text-slate-700">{line.nomor_akseptasi}</span>
        <button
          type="button"
          disabled={print.isPending}
          onClick={() => {
            print.reset()
            print.mutate(address, { onSuccess: (file) => simpanBerkas(file) })
          }}
          className="rounded bg-orange-500 px-3 py-1 text-xs font-medium text-white hover:bg-orange-600 disabled:opacity-60"
        >
          {print.isPending ? 'Printing…' : 'Print'}
        </button>
        {rules.cashierVisible && (
          <button
            type="button"
            disabled={rules.cashierDisabled}
            title={rules.cashierDisabled ? 'Already transferred to Kasir.' : undefined}
            onClick={() => setCashier(true)}
            className="rounded border border-blue-400 px-3 py-1 text-xs text-blue-700 hover:bg-blue-50 disabled:border-slate-200 disabled:text-slate-400 disabled:hover:bg-transparent"
          >
            Transfer Kasir
          </button>
        )}
      </div>
      {print.error && (
        <div className="mt-2">
          <ErrorMessage
            title="Draft Persetujuan belum dapat dicetak"
            description={
              violations.length > 0
                ? violations.map((v) => v.pesan).join(' ')
                : print.error instanceof Error
                  ? print.error.message
                  : 'Terjadi kesalahan pada sistem.'
            }
            tone="penolakan"
          />
        </div>
      )}
      {cashier && <CashierDialog claimID={claimID} address={address} onClose={() => setCashier(false)} />}
    </div>
  )
}

export type AcceptanceNumberRules = {
  visible: boolean
  cashierVisible: boolean
  cashierDisabled: boolean
}

/** Aturan tampil/mati — lihat komentar AcceptanceNumber. */
export function acceptanceNumberRules(line: Settlement): AcceptanceNumberRules {
  const accepted = (line.nomor_akseptasi ?? '').trim()
  const lod = (line.status_akseptasi_lod ?? '').trim()
  const type = (line.tipe_pembayaran ?? '').trim()
  return {
    visible: lod === '1' && accepted !== '',
    cashierVisible: type !== PaymentType.Salvage && accepted !== '',
    cashierDisabled: line.sudah_transfer_kasir === true,
  }
}
