import { useEffect, useRef, useState } from 'react'

import { simpanBerkas } from '@/api/client'
import { Button } from '@/components/Button'
import { useEscapeToClose } from '@/components/shared/useEscapeToClose'

import { usePLAList, usePrintPLA, useSavePLANotes, violationsFrom } from './api'
import { DialogFailure, DialogSuccess, LossAdviceTable } from './dialogParts'
import type { PLARow } from './types'

/**
 * Dialog Print PLA — padanan layar `Section/PrintPLA_dtl_sect.xml` (flow action lokal
 * `PrintPLA`). Membukanya menerbitkan PLA koasuransi revisi CFS terakhir bila belum ada.
 *
 * Grid-nya mengikuti section itu: NO PLA, PLA REINSURER, TIPE PLA, REMARKS (dapat diubah),
 * Email, lalu Print PLA per baris dan Print All PLA. SEND ALL PLA tampil tetapi mati —
 * pengiriman email tidak dibawa (keputusan Work Owner). Isian "Coverage/Interest AS PER
 * ORIGINAL POLICY" dan Remarks Reserve belum dibawa: akibatnya pada dokumen ditentukan
 * activity cetak `DownloadFireLossAdvice_act`, yang tidak ada di export.
 */
export function PLADialog({
  claimID,
  taskID,
  object,
  coverage,
  coverageName,
  onClose,
}: Readonly<{
  claimID: string
  taskID: string
  object: number
  coverage: number
  coverageName: string
  onClose: () => void
}>) {
  const list = usePLAList(claimID)
  const save = useSavePLANotes(claimID)
  const print = usePrintPLA(claimID)
  const [rows, setRows] = useState<PLARow[]>([])
  const [notes, setNotes] = useState<Record<string, string>>({})
  const request = { tugas_id: taskID, objek: object, jaminan: coverage }

  function show(result: { pla: PLARow[] }) {
    setRows(result.pla)
    setNotes(Object.fromEntries(result.pla.map((p) => [p.nomor, p.catatan])))
  }

  // Daftar diminta SEKALI saat dialog muncul. Membukanya dapat menerbitkan nomor PLA, dan
  // StrictMode menjalankan efek dua kali — dua permintaan bersamaan akan menerbitkan dua kali.
  const opened = useRef(false)
  useEffect(() => {
    if (opened.current) return
    opened.current = true
    list.mutate(request, { onSuccess: show })
  })

  const busy = list.isPending || save.isPending || print.isPending
  useEscapeToClose(onClose, busy)

  const changed = rows.filter((r) => (notes[r.nomor] ?? '') !== r.catatan)
  const failure = print.error ?? save.error ?? list.error
  const violations = violationsFrom(failure)

  function download(nomor?: string) {
    print.reset()
    print.mutate(nomor ? { ...request, nomor } : request, { onSuccess: (file) => simpanBerkas(file) })
  }

  function saveNotes() {
    const catatan = Object.fromEntries(changed.map((r) => [r.nomor, notes[r.nomor] ?? '']))
    save.mutate({ ...request, catatan }, { onSuccess: show })
  }

  return (
    <dialog
      open
      className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/40 px-4 py-8"
      aria-modal="true"
      aria-labelledby="judul-print-pla"
    >
      <div className="max-h-full w-full max-w-4xl overflow-y-auto rounded-kartu bg-white p-6 shadow-angkat">
        <h2 id="judul-print-pla" className="text-lg font-semibold text-slate-900">
          Print PLA
        </h2>
        <p className="mt-1 text-sm text-slate-600">{coverageName}</p>

        {list.isPending && <p className="mt-4 text-sm text-slate-500">Menyiapkan PLA…</p>}

        {rows.length > 0 && (
          <LossAdviceTable
            kind="PLA"
            emailTitle="Email"
            rows={rows}
            notes={notes}
            setNotes={setNotes}
            remarksProps={() => ({ className: 'w-64 rounded border border-slate-300 px-2 py-1' })}
            renderActions={(r) => (
              <button
                type="button"
                disabled={busy || changed.length > 0}
                title={changed.length > 0 ? 'Simpan remarks lebih dulu.' : undefined}
                onClick={() => download(r.nomor)}
                className="rounded bg-blue-800 px-2 py-1 text-xs text-white disabled:opacity-60"
              >
                Print PLA
              </button>
            )}
          />
        )}

        <DialogFailure title="PLA belum dapat diproses" failure={failure} violations={violations} />
        <DialogSuccess show={print.isSuccess && !busy && !failure}>PLA diunduh.</DialogSuccess>

        <div className="mt-6 flex flex-wrap justify-between gap-3">
          <Button tone="halus" disabled={busy} onClick={onClose}>
            Tutup
          </Button>
          <div className="flex flex-wrap gap-3">
            <Button tone="kedua" disabled={busy || changed.length === 0} onClick={saveNotes}>
              {save.isPending ? 'Menyimpan…' : 'Simpan Remarks'}
            </Button>
            <Button tone="kedua" disabled title="Pengiriman email PLA belum dibawa.">
              SEND ALL PLA
            </Button>
            <Button tone="utama" disabled={busy || rows.length === 0 || changed.length > 0} onClick={() => download()}>
              {print.isPending ? 'Mencetak…' : 'Print All PLA'}
            </Button>
          </div>
        </div>
      </div>
    </dialog>
  )
}
