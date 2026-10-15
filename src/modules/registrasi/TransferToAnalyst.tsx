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
