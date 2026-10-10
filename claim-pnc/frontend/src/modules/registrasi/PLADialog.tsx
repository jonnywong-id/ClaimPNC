import { useEffect, useRef, useState } from 'react'

import { simpanBerkas } from '@/api/client'
import { Button } from '@/components/Button'
import { ErrorMessage } from '@/components/ErrorMessage'

import { usePLAList, usePrintPLA, useSavePLANotes, useSendAllPLA, violationsFrom } from './api'
import type { PLARow, PLASendOutcome } from './types'

/**
 * Dialog Print PLA — padanan layar `Section/PrintPLA_dtl_sect.xml` (flow action lokal
 * `PrintPLA`). Membukanya menerbitkan PLA koasuransi revisi CFS terakhir bila belum ada.
 *
 * Grid-nya mengikuti section itu: NO PLA, PLA REINSURER, TIPE PLA, REMARKS dan Email (keduanya
 * dapat diubah — `.PLARemarks` dan `.pyEmailAddress`, pxTextArea Editable; Email disimpan ke
 * T_PLALIST.EMAILPLA), lalu Print PLA per baris dan Print All PLA. SEND ALL PLA
 * (`DownloadAllDocumentPLA` SendPrint "2") mengirim lewat email setiap PLA yang belum terkirim ke
 * alamat di kolom Email, lalu menandainya terkirim; hasilnya ditampilkan per PLA, dan sebab yang
 * gagal dapat dilampirkan ke IT Support (Work Owner 2026-10-10). Isian "Coverage/Interest AS PER
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
}: {
  claimID: string
  taskID: string
  object: number
  coverage: number
  coverageName: string
  onClose: () => void
}) {
  const list = usePLAList(claimID)
  const save = useSavePLANotes(claimID)
  const print = usePrintPLA(claimID)
  const send = useSendAllPLA(claimID)
  const [sendResult, setSendResult] = useState<PLASendOutcome[] | null>(null)
  const [rows, setRows] = useState<PLARow[]>([])
  const [notes, setNotes] = useState<Record<string, string>>({})
  const [emails, setEmails] = useState<Record<string, string>>({})
  const request = { tugas_id: taskID, objek: object, jaminan: coverage }

  function show(result: { pla: PLARow[] }) {
    setRows(result.pla)
    setNotes(Object.fromEntries(result.pla.map((p) => [p.nomor, p.catatan])))
    setEmails(Object.fromEntries(result.pla.map((p) => [p.nomor, p.email])))
  }

  // Daftar diminta SEKALI saat dialog muncul. Membukanya dapat menerbitkan nomor PLA, dan
  // StrictMode menjalankan efek dua kali — dua permintaan bersamaan akan menerbitkan dua kali.
  const opened = useRef(false)
  useEffect(() => {
    if (opened.current) return
    opened.current = true
    list.mutate(request, { onSuccess: show })
  })

  const busy = list.isPending || save.isPending || print.isPending || send.isPending
  useEffect(() => {
    function onKey(event: KeyboardEvent) {
      if (event.key === 'Escape' && !busy) onClose()
    }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [onClose, busy])

  const notesChanged = rows.filter((r) => (notes[r.nomor] ?? '') !== r.catatan)
  const emailsChanged = rows.filter((r) => (emails[r.nomor] ?? '').trim() !== r.email)
  const changed = [...notesChanged, ...emailsChanged]
  const failure = send.error ?? print.error ?? save.error ?? list.error
  const unsent = rows.filter((r) => !r.terkirim)
  const violations = violationsFrom(failure)

  function download(nomor?: string) {
    print.reset()
    print.mutate(nomor ? { ...request, nomor } : request, { onSuccess: (file) => simpanBerkas(file) })
  }

  function sendAll() {
    const ok = window.confirm(
      `Kirim ${unsent.length} PLA lewat email ke alamat di kolom Email?\n\n` +
        'Email yang sudah terkirim tidak dapat ditarik kembali.',
    )
    if (!ok) return
    send.reset()
    setSendResult(null)
    send.mutate(request, {
      onSuccess: (result) => {
        show(result.daftar)
        setSendResult(result.hasil)
      },
    })
  }

  function saveNotes() {
    const catatan = Object.fromEntries(notesChanged.map((r) => [r.nomor, notes[r.nomor] ?? '']))
    const email = Object.fromEntries(emailsChanged.map((r) => [r.nomor, (emails[r.nomor] ?? '').trim()]))
    save.mutate({ ...request, catatan, email }, { onSuccess: show })
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/40 px-4 py-8"
      role="dialog"
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
          <table className="mt-4 w-full border-collapse text-sm">
            <caption className="sr-only">Daftar PLA</caption>
            <thead>
              <tr className="bg-slate-100 text-left text-xs text-slate-700">
                <th className="p-2">NO PLA</th>
                <th className="p-2">PLA REINSURER</th>
                <th className="p-2">TIPE PLA</th>
                <th className="p-2">REMARKS</th>
                <th className="p-2">Email</th>
                <th className="p-2" />
              </tr>
            </thead>
            <tbody>
              {rows.map((r) => (
                <tr key={r.nomor} className="border-b border-slate-100 align-top">
                  <td className="p-2 font-mono text-xs">
                    {r.nomor}
                    {r.terkirim && (
                      <span className="mt-1 block font-sans text-[11px] font-medium text-emerald-700">Terkirim</span>
                    )}
                  </td>
                  <td className="p-2">{r.penerima}</td>
                  <td className="p-2">{r.tipe}</td>
                  <td className="p-2">
                    <textarea
                      aria-label={`Remarks ${r.nomor}`}
                      rows={3}
                      value={notes[r.nomor] ?? ''}
                      onChange={(e) => setNotes((n) => ({ ...n, [r.nomor]: e.target.value }))}
                      className="w-64 rounded border border-slate-300 px-2 py-1"
                    />
                  </td>
                  <td className="p-2">
                    <textarea
                      aria-label={`Email ${r.nomor}`}
                      rows={3}
                      maxLength={1000}
                      value={emails[r.nomor] ?? ''}
                      onChange={(e) => setEmails((m) => ({ ...m, [r.nomor]: e.target.value }))}
                      className="w-56 rounded border border-slate-300 px-2 py-1 text-xs"
                    />
                  </td>
                  <td className="p-2">
                    <button
                      type="button"
                      disabled={busy || changed.length > 0}
                      title={changed.length > 0 ? 'Simpan remarks dan email lebih dulu.' : undefined}
                      onClick={() => download(r.nomor)}
                      className="rounded bg-blue-800 px-2 py-1 text-xs text-white disabled:opacity-60"
                    >
                      Print PLA
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}

        {failure && (
          <div className="mt-4">
            <ErrorMessage
              title="PLA belum dapat diproses"
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
        {sendResult && <SendResult result={sendResult} />}
        {print.isSuccess && !busy && !failure && (
          <p className="mt-4 text-sm text-emerald-700" role="status">
            PLA diunduh.
          </p>
        )}

        <div className="mt-6 flex flex-wrap justify-between gap-3">
          <Button tone="halus" disabled={busy} onClick={onClose}>
            Tutup
          </Button>
          <div className="flex flex-wrap gap-3">
            <Button tone="kedua" disabled={busy || changed.length === 0} onClick={saveNotes}>
              {save.isPending ? 'Menyimpan…' : 'Simpan Remarks & Email'}
            </Button>
            <Button
              tone="kedua"
              disabled={busy || unsent.length === 0 || changed.length > 0}
              title={
                changed.length > 0
                  ? 'Simpan remarks dan email lebih dulu.'
                  : unsent.length === 0
                    ? 'Seluruh PLA sudah terkirim.'
                    : undefined
              }
              onClick={sendAll}
            >
              {send.isPending ? 'Mengirim…' : 'SEND ALL PLA'}
            </Button>
            <Button tone="utama" disabled={busy || rows.length === 0 || changed.length > 0} onClick={() => download()}>
              {print.isPending ? 'Mencetak…' : 'Print All PLA'}
            </Button>
          </div>
        </div>
      </div>
    </div>
  )
}

/** Hasil SEND ALL PLA per nomor; sebab yang gagal ditulis apa adanya untuk IT Support. */
function SendResult({ result }: { result: PLASendOutcome[] }) {
  const failed = result.filter((o) => o.galat)
  const sent = result.filter((o) => o.terkirim)
  const skipped = result.filter((o) => o.dilewati)
  return (
    <div className="mt-4 space-y-2" role="status">
      {sent.length > 0 && (
        <p className="text-sm text-emerald-700">
          {sent.length} PLA terkirim: {sent.map((o) => o.nomor).join(', ')}.
        </p>
      )}
      {skipped.length > 0 && (
        <p className="text-sm text-slate-600">
          {skipped.length} PLA sudah terkirim sebelumnya dan tidak dikirim ulang.
        </p>
      )}
      {failed.length > 0 && (
        <ErrorMessage
          tone="gangguan"
          title={`${failed.length} PLA gagal dikirim`}
          description={failed.map((o) => `${o.nomor}: ${o.galat}`).join(' | ') + ' — lampirkan rincian ini ke IT Support.'}
        />
      )}
    </div>
  )
}
