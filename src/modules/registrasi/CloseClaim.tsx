import { useEffect, useState } from 'react'

import { Button } from '@/components/Button'
import { ErrorMessage } from '@/components/ErrorMessage'
import { TextAreaField } from '@/components/TextAreaField'

import { useCloseClaim, violationsFrom } from './api'

/** Panjang kolom T_CLAIM_PNC: CLOSECLAIMNOTE 4000; USULAN, EFFORT_CLOSE, KENDALA_CLOSE 100. */
const MAX_NOTE = 4000
const MAX_SHORT = 100

const NOT_STORED = 'No options and no storage column for this field in the Pega export.'

/**
 * Dialog "Prevent Close Claim" — tombol Tutup Klaim (`Section/ClaimSurvey_sect.xml` → local action
 * `PreventRejectClaim`, `Section/PreventRejectClaim-sect.xml`). Urutan isian mengikuti section:
 * pertanyaan, Catatan (wajib), Tutup Sementara, Alasan Keterlambatan, No Reff Broker, Banding,
 * Survey Kepuasan, Manual/Paperless, Effort Sebelum Close, Kendala Sebelum Close, Usulan; lalu Ya / Tidak.
 *
 * Empat dropdown dan No Reff Broker tampil tetapi nonaktif: tidak punya daftar pilihan maupun kolom.
 */
export function CloseClaimDialog({ claimID, taskID, onClose }: { claimID: string; taskID: string; onClose: () => void }) {
  const close = useCloseClaim(claimID)
  const [note, setNote] = useState('')
  const [temporary, setTemporary] = useState(false)
  const [effort, setEffort] = useState('')
  const [obstacle, setObstacle] = useState('')
  const [proposal, setProposal] = useState('')
  const busy = close.isPending

  useEffect(() => {
    function onKey(event: KeyboardEvent) {
      if (event.key === 'Escape' && !busy) onClose()
    }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [onClose, busy])

  const violations = violationsFrom(close.error)
  const fieldError = (field: string) => violations.find((v) => v.field === field)?.pesan
  const tooLong = (value: string, max: number) => (value.trim().length > max ? `Max ${max} characters.` : undefined)
  const invalid = Boolean(
    tooLong(note, MAX_NOTE) || tooLong(effort, MAX_SHORT) || tooLong(obstacle, MAX_SHORT) || tooLong(proposal, MAX_SHORT),
  )

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/40 px-4 py-8"
      role="dialog"
      aria-modal="true"
      aria-labelledby="judul-tutup-klaim"
    >
      <div className="max-h-full w-full max-w-2xl overflow-y-auto rounded-kartu bg-white p-6 shadow-angkat">
        <h2 id="judul-tutup-klaim" className="text-lg font-semibold text-slate-900">
          Prevent Close Claim
        </h2>
        <p className="mt-1 text-sm font-semibold text-slate-800">Apakah anda yakin ingin menutup klaim ini?</p>

        <div className="mt-4 space-y-4">
          <TextAreaField
            id="catatan-tutup"
            label="Catatan *"
            rows={4}
            value={note}
            disabled={busy}
            onChange={(e) => setNote(e.target.value)}
            error={tooLong(note, MAX_NOTE) ?? fieldError('catatan_tutup')}
          />
          <label className="flex items-center gap-2 text-sm text-slate-800">
            <input type="checkbox" checked={temporary} disabled={busy} onChange={(e) => setTemporary(e.target.checked)} />
            Tutup Sementara
          </label>

          <div className="grid gap-4 sm:grid-cols-2">
            <DisabledSelect id="alasan-terlambat" label="Alasan Keterlambatan" />
            <label className="block text-xs font-semibold text-slate-800">
              No Reff Broker
              <input
                id="no-reff-broker"
                type="text"
                disabled
                title={NOT_STORED}
                className="mt-1 block w-full rounded border border-slate-300 px-2 py-1 text-sm disabled:bg-slate-100"
              />
            </label>
            <DisabledSelect id="banding" label="Banding" />
            <DisabledSelect id="survey-kepuasan" label="Survey Kepuasan" />
            <DisabledSelect id="paperless" label="Manual/Paperless" />
          </div>

          <TextAreaField
            id="effort-tutup"
            label="Effort Sebelum Close"
            rows={2}
            value={effort}
            disabled={busy}
            onChange={(e) => setEffort(e.target.value)}
            error={tooLong(effort, MAX_SHORT) ?? fieldError('effort_tutup')}
          />
          <TextAreaField
            id="kendala-tutup"
            label="Kendala Sebelum Close"
            rows={2}
            value={obstacle}
            disabled={busy}
            onChange={(e) => setObstacle(e.target.value)}
            error={tooLong(obstacle, MAX_SHORT) ?? fieldError('kendala_tutup')}
          />
          <TextAreaField
            id="usulan"
            label="Usulan"
            rows={2}
            value={proposal}
            disabled={busy}
            onChange={(e) => setProposal(e.target.value)}
            error={tooLong(proposal, MAX_SHORT) ?? fieldError('usulan')}
          />
        </div>

        {close.isError && violations.length === 0 && (
          <div className="mt-4">
            <ErrorMessage
              tone="penolakan"
              title="Klaim belum ditutup"
              description={close.error instanceof Error ? close.error.message : 'Terjadi kesalahan.'}
            />
          </div>
        )}

        <div className="mt-6 flex justify-end gap-3">
          <Button tone="halus" disabled={busy} onClick={onClose}>
            Tidak
          </Button>
          <Button
            tone="utama"
            disabled={busy || invalid}
            onClick={() => {
              close.reset()
              close.mutate(
                {
                  taskID,
                  body: {
                    catatan_tutup: note,
                    usulan: proposal,
                    effort_tutup: effort,
                    kendala_tutup: obstacle,
                    tutup_sementara: temporary,
                  },
                },
                { onSuccess: onClose },
              )
            }}
          >
            {busy ? 'Menutup…' : 'Ya'}
          </Button>
        </div>
      </div>
    </div>
  )
}

function DisabledSelect({ id, label }: { id: string; label: string }) {
  return (
    <label className="block text-xs font-semibold text-slate-800">
      {label}
      <select
        id={id}
        disabled
        title={NOT_STORED}
        className="mt-1 block w-full rounded border border-slate-300 px-2 py-1 text-sm disabled:bg-slate-100"
      >
        <option value="">-- Pilih --</option>
      </select>
    </label>
  )
}
