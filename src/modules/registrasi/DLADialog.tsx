import { useEffect, useRef, useState } from 'react'

import { simpanBerkas } from '@/api/client'
import { Button } from '@/components/Button'
import { ErrorMessage } from '@/components/ErrorMessage'

import { useDLAList, usePrintDLA, violationsFrom } from './api'
import type { DLAListResponse, DLARow } from './types'

/**
 * Dialog Print DLA — padanan layar `Section/PrintDLA-sect.xml` (flow action lokal `PrintDLA`,
 * pra-proses `GenerateDLAListAdjustment`). Membukanya menerbitkan DLA adjustment itu per
 * penerima bila belum ada: koasuransi, Fac Out, Treaty, dan BPPDAN/EQ POOL.
 *
 * Grid-nya mengikuti section itu: NO DLA, DLA REINSURER, TIPE DLA, REMARKS (dapat diubah
 * sampai DLA pertama kali dicetak), EMAIL, lalu PRINT per baris dan Print All DLA. Kirim dan
 * SEND ALL DLA tampil tetapi mati — pengiriman email belum aktif (keputusan Work Owner).
 * Dropdown "Status Klaim DLA" dan grid "File Pendukung" belum dibawa.
 */
export function DLADialog({
  claimID,
  address,
  onClose,
}: {
  claimID: string
  address: { tugas_id: string; objek: number; jaminan: number; adjustment: number }
  onClose: () => void
}) {
  const list = useDLAList(claimID)
  const print = usePrintDLA(claimID)
  const [data, setData] = useState<DLAListResponse | null>(null)
  const [notes, setNotes] = useState<Record<string, string>>({})
  const [asPerPolicy, setAsPerPolicy] = useState(false)

  function show(result: DLAListResponse) {
    setData(result)
    setNotes(Object.fromEntries(result.dla.map((d) => [d.nomor, d.catatan])))
  }

  // Daftar diminta SEKALI saat dialog muncul. Membukanya dapat menerbitkan nomor DLA, dan
  // StrictMode menjalankan efek dua kali — dua permintaan bersamaan akan menerbitkan dua kali.
  const opened = useRef(false)
  useEffect(() => {
    if (opened.current) return
    opened.current = true
    list.mutate(address, { onSuccess: show })
  })

  const busy = list.isPending || print.isPending
  useEffect(() => {
    function onKey(event: KeyboardEvent) {
      if (event.key === 'Escape' && !busy) onClose()
    }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [onClose, busy])

  const rows: DLARow[] = data?.dla ?? []
  const failure = print.error ?? list.error
  const violations = violationsFrom(failure)

  function download(nomor?: string) {
    print.reset()
    const catatan = Object.fromEntries(rows.filter((r) => !r.sudah_cetak).map((r) => [r.nomor, notes[r.nomor] ?? '']))
    print.mutate(
      { ...address, ...(nomor ? { nomor } : {}), catatan, sesuai_polis: asPerPolicy },
      {
        onSuccess: (file) => {
          simpanBerkas(file)
          // PRINT pertama menyimpan REMARKS dan menandai DLA tercetak — muat ulang daftarnya.
          list.mutate(address, { onSuccess: show })
        },
      },
    )
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/40 px-4 py-8"
      role="dialog"
      aria-modal="true"
      aria-labelledby="judul-print-dla"
    >
      <div className="max-h-full w-full max-w-5xl overflow-y-auto rounded-kartu bg-white p-6 shadow-angkat">
        <h2 id="judul-print-dla" className="text-lg font-semibold text-slate-900">
          Print DLA
        </h2>

        <label className="mt-3 flex items-center gap-2 text-sm text-slate-700">
          <input type="checkbox" checked={asPerPolicy} onChange={(e) => setAsPerPolicy(e.target.checked)} />
          Coverage AS PER ORIGINAL POLICY
        </label>

        {list.isPending && !data && <p className="mt-4 text-sm text-slate-500">Menyiapkan DLA…</p>}

        {data?.ex_gratia && (
          <p className="mt-4 text-sm text-slate-600" role="status">
            No DLA is issued for an ex gratia claim.
          </p>
        )}
        {data && !data.ex_gratia && rows.length === 0 && (
          <p className="mt-4 text-sm text-slate-600" role="status">
            No DLA recipient: the policy has no co-insurance led by us, and no Fac Out, Treaty, or BPPDAN spreading
            qualifies.
          </p>
        )}
        {data && data.peringatan.length > 0 && (
          <p className="mt-4 whitespace-pre-line text-sm text-amber-700">{data.peringatan.join('\n')}</p>
        )}

        {rows.length > 0 && (
          <table className="mt-4 w-full border-collapse text-sm">
            <caption className="sr-only">Daftar DLA</caption>
            <thead>
              <tr className="bg-slate-100 text-left text-xs text-slate-700">
                <th className="p-2">NO DLA</th>
                <th className="p-2">DLA REINSURER</th>
                <th className="p-2">TIPE DLA</th>
                <th className="p-2">REMARKS</th>
                <th className="p-2">EMAIL</th>
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
                      readOnly={r.sudah_cetak}
                      disabled={r.sudah_kirim}
                      onChange={(e) => setNotes((n) => ({ ...n, [r.nomor]: e.target.value }))}
                      className="w-64 rounded border border-slate-300 px-2 py-1 read-only:bg-slate-50"
                    />
                  </td>
                  <td className="p-2 text-xs text-slate-600">{r.email || '—'}</td>
                  <td className="p-2">
                    <div className="flex flex-col gap-1">
                      <button
                        type="button"
                        disabled={busy}
                        onClick={() => download(r.nomor)}
                        className="rounded bg-blue-800 px-2 py-1 text-xs text-white disabled:opacity-60"
                      >
                        PRINT
                      </button>
                      {r.sudah_cetak && (
                        <button
                          type="button"
                          disabled
                          title="Sending DLA by email is not active yet."
                          className="rounded border border-slate-300 px-2 py-1 text-xs text-slate-400"
                        >
                          Kirim
                        </button>
                      )}
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}

        {failure && (
          <div className="mt-4">
            <ErrorMessage
              title="DLA belum dapat diproses"
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
        {print.isSuccess && !busy && !failure && (
          <p className="mt-4 text-sm text-emerald-700" role="status">
            DLA diunduh.
          </p>
        )}

        <div className="mt-6 flex flex-wrap justify-between gap-3">
          <Button tone="halus" disabled={busy} onClick={onClose}>
            Tutup
          </Button>
          <div className="flex flex-wrap gap-3">
            <Button tone="kedua" disabled title="Sending DLA by email is not active yet.">
              SEND ALL DLA
            </Button>
            <Button tone="utama" disabled={busy || rows.length === 0} onClick={() => download()}>
              {print.isPending ? 'Mencetak…' : 'Print All DLA'}
            </Button>
          </div>
        </div>
      </div>
    </div>
  )
}
