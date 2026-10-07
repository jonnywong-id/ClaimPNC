import { useState } from 'react'
import { Link } from 'react-router-dom'

import { APIError } from '@/api/client'
import { ErrorMessage } from '@/components/ErrorMessage'
import { formatDate } from '@/components/format'

import { useCommittee, useDecideCommittee, usePendingCommittees, useTransferCommittee, violationsFrom } from './api'
import { CommitteeDecision, type CommitteeItem, type Settlement } from './types'

/**
 * Komite adjustment klaim PNCN.
 *
 * - Tombol **Transfer Komite** pada baris Adjustment (`ValidationTypePaymentAdj` →
 *   `SetListComiteeClaimPerObjAdj`): baris dibekukan dan jenjang 1 menunggu putusan.
 * - Kolom Akseptasi menampilkan status komite per jenjang.
 * - Daftar **Komite** di Inbox berisi putusan yang menunggu pengguna ini, dengan Setuju/Tolak
 *   (`KomitePost_Adjustment`).
 *
 * Section grid Adjustment Pega (`InputEstimasi`) tidak ada di export; letak tombolnya
 * mengikuti tangkapan layar Work Owner.
 */

function failureText(failure: unknown): string {
  const violations = violationsFrom(failure)
  if (violations.length > 0) return violations.map((v) => v.pesan).join(' ')
  if (failure instanceof APIError || failure instanceof Error) return failure.message
  return 'Terjadi kesalahan pada sistem.'
}

/** Tombol Transfer Komite satu baris Adjustment. Mati bila baris sudah ditransfer. */
export function TransferCommitteeButton({
  claimID,
  taskID,
  object,
  coverage,
  adjustment,
  line,
  lockedReason,
}: Readonly<{
  claimID: string
  taskID: string
  object: number
  coverage: number
  adjustment: number
  line: Settlement
  lockedReason: string | null
}>) {
  const transfer = useTransferCommittee(claimID)
  const done = !!line.komite_id
  const title = done ? 'Sudah ditransfer ke komite.' : (lockedReason ?? undefined)
  return (
    <div>
      <button
        type="button"
        disabled={done || lockedReason !== null || transfer.isPending}
        title={title}
        onClick={() => transfer.mutate({ tugas_id: taskID, objek: object, jaminan: coverage, adjustment })}
        className="rounded bg-orange-500 px-2 py-0.5 text-xs text-white disabled:opacity-60"
      >
        {transfer.isPending ? 'Mentransfer…' : 'Transfer Komite'}
      </button>
      {transfer.isError && (
        <p role="alert" className="mt-1 max-w-48 text-xs text-red-700">
          {failureText(transfer.error)}
        </p>
      )}
    </div>
  )
}

const decisionLabel: Record<string, string> = {
  [CommitteeDecision.pending]: 'menunggu',
  [CommitteeDecision.approve]: 'setuju',
  [CommitteeDecision.reject]: 'tolak',
}

/** Kolom Akseptasi: status komite baris yang sudah ditransfer. */
export function CommitteeStatus({ line }: Readonly<{ line: Settlement }>) {
  const committee = useCommittee(line.komite_id)
  if (!line.komite_id) return <span>Belum ditransfer</span>
  const c = committee.data
  if (!c) return <span>Komite {line.komite_id}</span>

  if (c.status === 'disetujui') {
    return <span>Diakseptasi komite{line.nomor_akseptasi ? ` · ${line.nomor_akseptasi}` : ''} · {c.id}</span>
  }
  if (c.status === 'ditolak') return <span>Ditolak komite · {c.id}</span>

  const current = c.anggota.find((m) => m.keputusan === CommitteeDecision.pending)
  return (
    <span title={c.anggota.map((m) => `${m.jenjang}. ${m.komite}: ${decisionLabel[m.keputusan] ?? m.keputusan}`).join('\n')}>
      Komite {c.id} · jenjang {current?.jenjang ?? '—'}/{c.anggota.length} menunggu {c.menunggu ?? '—'}
    </span>
  )
}

const amount = new Intl.NumberFormat('id-ID', { minimumFractionDigits: 2, maximumFractionDigits: 2 })

/** Daftar Komite di Inbox: putusan komite klaim PNCN yang menunggu pengguna ini. */
export function CommitteeInbox() {
  const pending = usePendingCommittees()
  const items = pending.data?.komite ?? []

  return (
    <section className="mt-8" aria-label="Komite">
      <h2 className="text-xs font-medium uppercase tracking-wide text-slate-500">Komite</h2>
      {pending.isPending && <p className="mt-3 text-sm text-slate-500">Memuat putusan komite…</p>}
      {pending.isError && (
        <div className="mt-3">
          <ErrorMessage title="Daftar komite tidak dapat dimuat" description={failureText(pending.error)} tone="gangguan" />
        </div>
      )}
      {pending.data && items.length === 0 && (
        <p className="mt-3 rounded border border-slate-200 bg-slate-50 p-4 text-sm text-slate-600">
          Tidak ada putusan komite yang menunggu Anda.
        </p>
      )}
      {items.length > 0 && (
        <div className="mt-3 space-y-3">
          {items.map((item) => (
            <CommitteeRow key={`${item.komite_id}-${item.jenjang}`} item={item} />
          ))}
        </div>
      )}
    </section>
  )
}

function CommitteeRow({ item }: Readonly<{ item: CommitteeItem }>) {
  const [note, setNote] = useState('')
  const decide = useDecideCommittee()
  const send = (keputusan: string) => decide.mutate({ komiteID: item.komite_id, keputusan, catatan: note })

  return (
    <fieldset aria-label={`Komite ${item.komite_id}`} className="rounded border border-slate-200 p-3 text-sm">
      <div className="flex flex-wrap items-baseline justify-between gap-2">
        <p className="font-medium text-slate-900">
          {item.klaim_id ? (
            <Link to={`/registrasi/klaim/${encodeURIComponent(item.klaim_id)}`} className="underline">
              {item.nomor_klaim}
            </Link>
          ) : (
            item.nomor_klaim
          )}{' '}
          · {item.komite_id} · jenjang {item.jenjang}/{item.jumlah_jenjang || '—'}
        </p>
        <p className="text-xs text-slate-500">Ditransfer {formatDate(item.tanggal_transfer)}</p>
      </div>
      <p className="mt-1 text-slate-700">
        {item.nama_tertanggung || '—'} · {item.nama_objek || '—'} / {item.nama_coverage || '—'} · Adjustment {item.adjustment || '—'}
      </p>
      <p className="text-slate-700">
        {item.nama_tipe_pembayaran} · {item.mata_uang} {amount.format(item.nilai_asm_sen / 100)} · Nilai komite IDR{' '}
        {amount.format(item.nilai_komite_sen / 100)}
      </p>
      <label className="mt-2 block text-xs font-semibold text-slate-800">
        <span>Catatan Komite</span>
        <textarea
          value={note}
          onChange={(e) => setNote(e.target.value)}
          rows={2}
          className="mt-1 block w-full rounded border border-slate-300 px-2 py-1 text-sm font-normal"
        />
      </label>
      <div className="mt-2 flex gap-2">
        <button
          type="button"
          disabled={decide.isPending}
          onClick={() => send(CommitteeDecision.approve)}
          className="rounded bg-slate-900 px-3 py-1 text-sm text-white disabled:opacity-60"
        >
          Setuju
        </button>
        <button
          type="button"
          disabled={decide.isPending}
          onClick={() => send(CommitteeDecision.reject)}
          className="rounded border border-red-300 px-3 py-1 text-sm text-red-700 disabled:opacity-60"
        >
          Tolak
        </button>
      </div>
      {decide.isError && (
        <p role="alert" className="mt-2 text-xs text-red-700">
          {failureText(decide.error)}
        </p>
      )}
    </fieldset>
  )
}
