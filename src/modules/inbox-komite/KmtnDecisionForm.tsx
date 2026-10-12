import { useState } from 'react'
import { Link } from 'react-router-dom'

import { APIError } from '@/api/client'
import { useSession } from '@/app/session'
import { Button } from '@/components/Button'

import { isKmtnCase, useDecideKmtn, useKmtnCommittee } from './api'

/**
 * Form keputusan komite — bagian akhir `Section/ShowTransfer`:
 * "Apakah anda yakin untuk akseptasi ini?", Keputusan Komite (Approved / Rejected),
 * Catatan Komite, Back dan Submit.
 *
 * # Hanya untuk case KMTN
 *
 * Diputuskan Work Owner 2026-09-29, membatasi pencabutan tombol keputusan pada hari yang
 * sama: case KMTN dibentuk aplikasi ini dan keputusannya ditulis modul registrasi
 * (`DecideCommittee` — pengganti `KomitePost_Adjustment`). Case Pega tetap hanya-baca,
 * karena tabel dan alurnya masih milik Pega (`P-1`).
 *
 * Form hanya tergambar bagi anggota yang sedang DITUNGGU. Penyembunyian itu kenyamanan
 * tampilan; server tetap menolak anggota yang bukan gilirannya.
 *
 * Teks tambahan yang tidak ada di Pega berbahasa Inggris (D-80); label form mengikuti Pega.
 *
 * # Yang tidak dibawa
 *
 * `.RejectedCode` (pasal penolakan) pada ShowTransfer tidak ditampilkan: sumber daftarnya
 * (`GetDataPenolakanKlaimMas`) tidak ada di export.
 */
export function KmtnDecisionForm({ caseID }: { caseID: string }) {
  const login = useSession((state) => state.user?.login ?? '')
  const committee = useKmtnCommittee(caseID)
  const decide = useDecideKmtn(caseID)
  const [decision, setDecision] = useState<'' | '1' | '2'>('')
  const [note, setNote] = useState('')
  const [missing, setMissing] = useState<string[]>([])

  if (!isKmtnCase(caseID)) return null

  const box = 'mt-5 rounded-kartu border border-slate-200 bg-white p-5 shadow-sm'

  if (committee.isPending) {
    return (
      <p className="mt-5 text-sm text-slate-600" role="status">
        Loading committee status…
      </p>
    )
  }
  if (committee.isError || !committee.data) {
    return (
      <p className={`${box} text-sm text-rose-700`} role="alert">
        The committee status could not be loaded, so no decision can be given yet.
      </p>
    )
  }

  const c = committee.data
  if (c.status !== 'berjalan') {
    return (
      <p className={`${box} text-sm text-slate-700`} role="status">
        This committee is closed — {c.status === 'disetujui' ? 'approved' : 'rejected'}.
      </p>
    )
  }
  const mine = (c.menunggu ?? '').trim().toUpperCase() === login.trim().toUpperCase()
  if (!mine) {
    return (
      <p className={`${box} text-sm text-slate-700`} role="status">
        Waiting for the decision of {c.menunggu || 'the next committee member'}.
      </p>
    )
  }

  function submit() {
    const kurang: string[] = []
    if (!decision) kurang.push('Keputusan Komite')
    if (!note.trim()) kurang.push('Catatan Komite')
    setMissing(kurang)
    if (kurang.length > 0 || !decision) return
    decide.mutate({ decision, note: note.trim() })
  }

  const failure =
    decide.error instanceof APIError
      ? decide.error.message
      : decide.error
        ? 'The decision could not be saved. Try again.'
        : ''

  return (
    <section aria-label="Keputusan komite" className={box}>
      <p className="text-sm text-slate-900">Apakah anda yakin untuk akseptasi ini?</p>

      <fieldset className="mt-4">
        <legend className="text-xs font-semibold text-slate-700">
          Keputusan Komite <span className="text-amber-600">*</span>
        </legend>
        <div className="mt-2 flex flex-wrap gap-8 text-sm text-slate-800">
          {(
            [
              ['1', 'Approved'],
              ['2', 'Rejected'],
            ] as const
          ).map(([value, label]) => (
            <label key={value} className="inline-flex items-center gap-2">
              <input
                type="radio"
                name="keputusan-komite"
                value={value}
                checked={decision === value}
                onChange={() => setDecision(value)}
                disabled={decide.isPending}
              />
              {label}
            </label>
          ))}
        </div>
      </fieldset>

      <label htmlFor="catatan-komite" className="mt-4 block text-xs font-semibold text-slate-700">
        Catatan Komite <span className="text-amber-600">*</span>
      </label>
      <textarea
        id="catatan-komite"
        value={note}
        onChange={(e) => setNote(e.target.value)}
        rows={3}
        disabled={decide.isPending}
        className="mt-1 w-full rounded-kontrol border border-slate-300 p-2 text-sm"
      />

      {missing.length > 0 && (
        <p className="mt-2 text-sm text-rose-700" role="alert">
          {missing.join(' and ')} {missing.length > 1 ? 'are' : 'is'} required.
        </p>
      )}
      {failure && (
        <p className="mt-2 text-sm text-rose-700" role="alert">
          {failure}
        </p>
      )}
      {decide.isSuccess && (
        <p className="mt-2 text-sm text-emerald-700" role="status">
          Decision saved.
        </p>
      )}

      <div className="mt-4 flex justify-end gap-2">
        <Link
          to="/komite/inbox"
          className="inline-flex items-center rounded-kartu border border-slate-200 px-3 py-1.5 text-sm font-medium text-slate-700 hover:bg-slate-50"
        >
          Back
        </Link>
        <Button tone="utama" onClick={submit} disabled={decide.isPending}>
          {decide.isPending ? 'Saving…' : 'Submit'}
        </Button>
      </div>
    </section>
  )
}
