import { useEffect, useState } from 'react'

import { Button } from '@/components/Button'
import { ErrorMessage } from '@/components/ErrorMessage'
import { TextAreaField } from '@/components/TextAreaField'
import { formatDate } from '@/components/format'

import { useProgressRecords, useSendToInputor, violationsFrom } from './api'
import type { Communication } from './types'

/** COMMUNICATE_FROM catatan tombol "Kirim ke Inputor" — parameter `sendToInvest` di Pega. */
export const CHANNEL_SEND_TO_INPUTOR = 'SENDTOINPUTOR'

/** Panjang kolom M_KOMUNIKASI_PNC.MESSAGE. */
const MAX_NOTE = 4000

/**
 * Modal "Kirim ke Inputor" — local action `AnalystRemarks`, section `AnalystRemarks_sect`: satu
 * Text Area (`.ClaimData.AnaylstRemarks`, tidak wajib) dan tombol Kirim. Kirim menutup tugas
 * berjalan dan melompatkan klaim ke Input Register (`setToRegister_ticket`), milik Inputor.
 */
export function SendToInputorDialog({ claimID, taskID, onClose }: { claimID: string; taskID: string; onClose: () => void }) {
  const send = useSendToInputor(claimID)
  const [note, setNote] = useState('')
  const busy = send.isPending

  useEffect(() => {
    function onKey(event: KeyboardEvent) {
      if (event.key === 'Escape' && !busy) onClose()
    }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [onClose, busy])

  const violations = violationsFrom(send.error)
  const tooLong = note.trim().length > MAX_NOTE

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/40 px-4 py-8"
      role="dialog"
      aria-modal="true"
      aria-labelledby="judul-kirim-inputor"
    >
      <div className="max-h-full w-full max-w-lg overflow-y-auto rounded-kartu bg-white p-6 shadow-angkat">
        <h2 id="judul-kirim-inputor" className="text-lg font-semibold text-slate-900">
          Kirim ke Inputor
        </h2>
        <p className="mt-1 text-sm text-slate-600">The claim returns to Input Register, assigned to the Inputor.</p>

        <div className="mt-4">
          <TextAreaField
            id="catatan-analis"
            label="Note"
            rows={5}
            value={note}
            disabled={busy}
            onChange={(e) => setNote(e.target.value)}
            error={tooLong ? `Max ${MAX_NOTE} characters.` : undefined}
          />
        </div>

        {send.isError && (
          <div className="mt-4">
            <ErrorMessage
              tone="penolakan"
              title="Klaim belum terkirim ke Inputor"
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
            Cancel
          </Button>
          <Button
            tone="utama"
            disabled={busy || tooLong}
            onClick={() => {
              send.reset()
              send.mutate({ taskID, catatan: note }, { onSuccess: onClose })
            }}
          >
            {busy ? 'Mengirim…' : 'Kirim'}
          </Button>
        </div>
      </div>
    </div>
  )
}

/** formatMoment menampilkan waktu RFC 3339 (WIB) sebagai `1 Juni 2026 14:05`. */
function formatMoment(iso: string): string {
  const [date = '', rest = ''] = iso.split('T')
  const time = rest.slice(0, 5)
  return time ? `${formatDate(date)} ${time}` : formatDate(date)
}

/** Catatan "Kirim ke Inputor" terbaru, atau undefined. Daftar komunikasi sudah terurut terbaru. */
export function latestAnalystNote(list: Communication[]): Communication | undefined {
  return list.find((c) => c.kanal === CHANNEL_SEND_TO_INPUTOR && c.pesan.trim() !== '')
}

/**
 * "Catatan dari Analyst" pada layar Input Register (`ViewInputRegisterDetail`, format Status
 * failure). Pega menampilkannya selama `StatusAnalystRemarks == 1`; di sini catatan terbaru
 * berkanal SENDTOINPUTOR ditampilkan.
 */
export function AnalystNoteNotice({ claimID }: { claimID: string }) {
  const q = useProgressRecords(claimID)
  const note = latestAnalystNote(q.data?.komunikasi ?? [])
  if (!note) return null
  return (
    <div className="mt-4 rounded border border-red-200 bg-red-50 p-3 text-sm" role="note" aria-label="Catatan dari Analyst">
      <p className="text-xs font-semibold text-red-800">Catatan dari Analyst</p>
      <p className="mt-1 whitespace-pre-wrap text-red-900">{note.pesan}</p>
      <p className="mt-1 text-xs text-red-700">
        {note.pengirim || '—'}
        {note.tanggal ? ` · ${formatMoment(note.tanggal)}` : ''}
      </p>
    </div>
  )
}
