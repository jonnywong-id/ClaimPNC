import { useEffect } from 'react'

import { Button } from '@/components/Button'
import { ErrorMessage } from '@/components/ErrorMessage'

import { useTransferToAnalyst, violationsFrom } from './api'
import type { Claim, InsuredItem } from './types'

/** Group Panel PA — When `IsPA`. */
const PANEL_PA = '002'

/** Tahap Estimation PA (Assignment4) — layar tempat grid ObjectCoverageAdj berada. */
const STAGE_ESTIMATE_PA = 'estimasi-pa'

/** Kode jaminan When `IsPHK` (`.CoverageOldID` 10010, 10023, 10018). */
const PHK_COVERAGES = ['10010', '10023', '10018']

/** When `IsPHK` pada baris jaminan. */
export function isPHKCoverage(coverageID: string): boolean {
  return PHK_COVERAGES.includes(coverageID)
}

/**
 * Kondisi tombol "Transfer ke Analyst" pada `Section/TrfKomiteButton` (kontainer `IsPA`):
 * `.IsAnalisTransfer != '1' && !IsPHK && pyWorkPage.ClaimData.PNCStatus != '5'`.
 *
 * `.IsAnalisTransfer` milik jaminan itu (ISANALISTRANSFER, `sudah_transfer_analis` jaminan). Tombol tampil
 * pada setiap jaminan; `setTicketToAnalyst` hanya memindahkan klaim bila jaminan yang ditekan adalah
 * `ObjectCoverageList(<last>)` — jaminan lain hanya ditandai sehingga tombolnya hilang.
 */
export function showTransferToAnalyst(klaim: Claim, stage: string, item: InsuredItem, index: number): boolean {
  const coverage = item.coverage[index]
  return (
    klaim.polis.lini === PANEL_PA &&
    stage === STAGE_ESTIMATE_PA &&
    coverage !== undefined &&
    !coverage.sudah_transfer_analis &&
    !PHK_COVERAGES.includes(coverage.id)
  )
}

/**
 * Modal yang dibuka tombol "Transfer ke Analyst" — local action `ClaimComitee_OC` (judul Pega
 * "Transfer Claim ke Komite"). Dari isian modal, yang dibangun baru tombol Kirim Analyst
 * (`setTicketToAnalyst`: lompat ke Send To Analis, StatusClaim 1151) dan Batal.
 */
export function TransferToAnalystDialog({
  claimID,
  taskID,
  objekID,
  coverageID,
  coverageName,
  onClose,
}: Readonly<{
  claimID: string
  taskID: string
  objekID: string
  coverageID: string
  coverageName: string
  onClose: () => void
}>) {
  const send = useTransferToAnalyst(claimID)
  const busy = send.isPending

  useEffect(() => {
    function onKey(event: KeyboardEvent) {
      if (event.key === 'Escape' && !busy) onClose()
    }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [onClose, busy])

  const violations = violationsFrom(send.error)

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/40 px-4 py-8"
      role="dialog"
      aria-modal="true"
      aria-labelledby="judul-transfer-analis"
    >
      <div className="max-h-full w-full max-w-lg overflow-y-auto rounded-kartu bg-white p-6 shadow-angkat">
        <h2 id="judul-transfer-analis" className="text-lg font-semibold text-slate-900">
          Transfer Claim ke Komite
        </h2>
        <p className="mt-1 text-sm text-slate-600">
          Coverage: {coverageName || coverageID}.
        </p>

        {send.isError && (
          <div className="mt-4">
            <ErrorMessage
              tone="penolakan"
              title="Klaim belum terkirim ke Analyst"
              description={
                violations.length > 0
                  ? violations.map((v) => v.pesan).join(' ')
                  : send.error instanceof Error
                    ? send.error.message
                    : 'Terjadi kesalahan.'
              }
            />
          </div>
        )}

        <div className="mt-6 flex justify-end gap-3">
          <Button tone="halus" disabled={busy} onClick={onClose}>
            Batal
          </Button>
          <Button
            tone="utama"
            disabled={busy}
            onClick={() => {
              send.reset()
              send.mutate({ taskID, objekID, coverageID }, { onSuccess: onClose })
            }}
          >
            {busy ? 'Mengirim…' : 'Kirim Analyst'}
          </Button>
        </div>
      </div>
    </div>
  )
}
