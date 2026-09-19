import { useState } from 'react'
import { Link } from 'react-router-dom'

import { APIError } from '@/api/client'
import { FormField } from '@/components/FormField'
import { ErrorMessage } from '@/components/ErrorMessage'
import { formatDate } from '@/components/format'

import { useClaimTask, useInbox, useStartClaim } from './api'
import { QueueKind, type Task } from './types'

/**
 * Inbox — daftar pekerjaan milik seorang pengguna (`D-79`).
 *
 * Isinya dua kelompok yang sengaja disatukan, persis seperti di sistem lama: tugas
 * Worklist yang sudah menjadi miliknya, dan tugas Workbasket yang belum bertuan pada
 * antrean yang ia berwenang. Yang kedua belum menjadi pekerjaannya — ia tawaran, dan
 * tombolnya berbunyi "Ambil", bukan "Kerjakan".
 */
export function InboxPage() {
  const inbox = useInbox()
  const get = useClaimTask()

  return (
    <div className="mx-auto max-w-5xl px-4 py-8">
      <header className="border-b border-slate-200 pb-4">
        <h1 className="text-xl font-semibold text-slate-900">Registrasi Klaim</h1>
        <p className="mt-1 text-sm text-slate-600">
          Pekerjaan yang menunggu Anda pada alur Register.
        </p>
      </header>

      <section className="mt-6">
        <NewClaimForm />
      </section>

      <section className="mt-8">
        <h2 className="text-xs font-medium uppercase tracking-wide text-slate-500">
          Inbox saya
        </h2>

        {inbox.isPending && <p className="mt-3 text-sm text-slate-500">Memuat pekerjaan…</p>}

        {inbox.isError && (
          <div className="mt-3">
            <ErrorMessage
              title="Daftar pekerjaan tidak dapat dimuat"
              note={errorMessage(inbox.error)}
              tone="gangguan"
            />
          </div>
        )}

        {inbox.data && inbox.data.tugas.length === 0 && (
          <p className="mt-3 rounded border border-slate-200 bg-slate-50 p-4 text-sm text-slate-600">
            Tidak ada pekerjaan yang menunggu. Mulailah dari sebuah nomor polis di atas.
          </p>
        )}

        {inbox.data && inbox.data.tugas.length > 0 && (
          <TaskTable
            tugas={inbox.data.tugas}
            claiming={get.isPending ? get.variables : undefined}
            onClaim={(id) => get.mutate(id)}
          />
        )}

        {get.isError && (
          <div className="mt-3">
            <ErrorMessage
              title="Tugas tidak dapat diambil"
              note={errorMessage(get.error)}
              tone="penolakan"
            />
          </div>
        )}
      </section>
    </div>
  )
}

function TaskTable({
  tugas,
  claiming,
  onClaim,
}: {
  tugas: Task[]
  claiming: string | undefined
  onClaim: (taskID: string) => void
}) {
  return (
    <div className="mt-3 overflow-x-auto">
      <table className="w-full border-collapse text-sm">
        <caption className="sr-only">Daftar pekerjaan yang menunggu</caption>
        <thead>
          <tr className="border-b border-slate-200 text-left text-xs uppercase tracking-wide text-slate-500">
            <th scope="col" className="py-2 pr-4 font-medium">Nomor klaim</th>
            <th scope="col" className="py-2 pr-4 font-medium">Tahap</th>
            <th scope="col" className="py-2 pr-4 font-medium">Antrean</th>
            <th scope="col" className="py-2 pr-4 font-medium">Masuk</th>
            <th scope="col" className="py-2 font-medium">Tindakan</th>
          </tr>
        </thead>
        <tbody>
          {tugas.map((t) => (
            <tr key={t.id} className="border-b border-slate-100">
              <td className="py-2 pr-4 text-slate-900">
                {t.nomor_klaim || <span className="text-slate-400">belum bernomor</span>}
              </td>
              <td className="py-2 pr-4 text-slate-700">{t.nama_tahap || t.tahap}</td>
              <td className="py-2 pr-4 text-slate-700">
                {t.antrean === QueueKind.workbasket ? (
                  <span title="Antrean bersama">Workbasket · {t.workbasket}</span>
                ) : (
                  <span title="Tugas milik satu orang">Worklist</span>
                )}
              </td>
              <td className="py-2 pr-4 text-slate-600">{formatDate(t.dibuat_pada.slice(0, 10))}</td>
              <td className="py-2">
                {t.dapat_diambil ? (
                  <button
                    type="button"
                    onClick={() => onClaim(t.id)}
                    disabled={claiming === t.id}
                    className="rounded border border-slate-300 px-3 py-1 text-sm font-medium text-slate-700 hover:bg-slate-100 disabled:cursor-not-allowed disabled:opacity-60"
                  >
                    {claiming === t.id ? 'Mengambil…' : 'Ambil'}
                  </button>
                ) : (
                  <Link
                    to={`/registrasi/klaim/${t.klaim_id}`}
                    className="rounded bg-slate-900 px-3 py-1 text-sm font-medium text-white hover:bg-slate-700"
                  >
                    Kerjakan
                  </Link>
                )}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

/**
 * Membuka klaim baru dari sebuah nomor polis.
 *
 * Pencarian polis yang sesungguhnya adalah `B-1`, yang belum ada. Sampai ia ada, nomor
 * polis diketik langsung — dan polis yang tidak dikenali ditolak server, bukan diterima
 * lalu gagal belakangan.
 */
function NewClaimForm() {
  const [policyNumber, setPolicyNumber] = useState('')
  const mulai = useStartClaim()

  return (
    <form
      className="rounded border border-slate-200 p-4"
      onSubmit={(e) => {
        e.preventDefault()
        if (policyNumber.trim() === '') return
        mulai.mutate({ nomor_polis: policyNumber.trim(), portal: '' })
      }}
    >
      <h2 className="text-xs font-medium uppercase tracking-wide text-slate-500">
        Mulai klaim baru
      </h2>

      <div className="mt-3 flex flex-wrap items-end gap-3">
        <div className="min-w-64 flex-1">
          <FormField
            id="nomorPolis"
            label="Nomor polis"
            value={policyNumber}
            onChange={(e) => setPolicyNumber(e.target.value)}
            placeholder="POL-FIRE-0001"
            autoComplete="off"
          />
        </div>
        <button
          type="submit"
          disabled={mulai.isPending || policyNumber.trim() === ''}
          className="rounded bg-slate-900 px-4 py-2 text-sm font-medium text-white hover:bg-slate-700 disabled:cursor-not-allowed disabled:opacity-60"
        >
          {mulai.isPending ? 'Membuka…' : 'Buka klaim'}
        </button>
      </div>

      {mulai.isError && (
        <div className="mt-3">
          <ErrorMessage
            title="Klaim tidak dapat dibuka"
            note={errorMessage(mulai.error)}
            tone="penolakan"
          />
        </div>
      )}

      {mulai.isSuccess && (
        <p className="mt-3 text-sm text-slate-700">
          Klaim dibuka pada tahap View Polis.{' '}
          <Link
            to={`/registrasi/klaim/${mulai.data.klaim.id}`}
            className="font-medium text-slate-900 underline"
          >
            Lanjutkan
          </Link>
        </p>
      )}

      <p className="mt-3 text-xs text-slate-500">
        Nomor klaim belum terbit di tahap ini. Ia terbit setelah Input Register lolos
        validasi, dan setelah terbit tidak dapat ditarik kembali.
      </p>
    </form>
  )
}

function errorMessage(failure: unknown): string {
  if (failure instanceof APIError) return failure.message
  if (failure instanceof Error) return failure.message
  return 'Terjadi kesalahan pada sistem.'
}
