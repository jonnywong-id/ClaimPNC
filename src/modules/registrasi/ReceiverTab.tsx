import { Fragment, useEffect, useState, type ReactNode } from 'react'

import { APIError } from '@/api/client'
import { Button } from '@/components/Button'
import { ErrorMessage } from '@/components/ErrorMessage'
import { formatDate } from '@/components/format'

import { messagesByField, useBankAccount, useSaveReceiver, violationsFrom } from './api'
import type { Claim, Receiver, Task } from './types'

/**
 * Sub-tab **Penerima Klaim** — grid `.ClaimData.ReceiverClaim` (kolom Nama, Alamat) yang
 * barisnya dibuka menjadi panel isian `Section/InputReceiver_sect.xml`, dengan tombol Tambah.
 *
 * # Susunan InputReceiver (kontainer `.FlagData==''`)
 *
 * Kiri: No Rekening (isian) · Nama Bank · Nama Cabang Bank · Tanggal Approve Kasir · Telepon
 * (isian) · catatan WhatsApp. Kanan: Nama · Alamat · Email* (isian) · Tanggal Approve
 * Komite. Lalu tombol Simpan. Jumlah, Nomor Kontrak, dan Tipe Pembayaran bersyarat `1==2`
 * atau `IsPA` — tidak ditampilkan; kontainer kedua (Master Rekening PA) bersyarat `1==2`.
 *
 * # Yang direkonstruksi
 *
 * Grid dengan tombol Tambah (`ShowReceiver`), activity pengisi No Rekening
 * (`GetDataBankMaster`), dan tombol Simpan (`SetIDReceiver`) tidak ada di export. No Rekening
 * dibaca dari Master Rekening saat isiannya ditinggalkan; Nama, Nama Bank, dan Alamat
 * baca-saja sehingga hanya master yang mengisinya. Email dan Telepon terisi dari master dan
 * dapat diubah, tetapi BELUM tersimpan — T_CLAIM_RECEIVER tidak punya kolomnya.
 */
export function ReceiverTab({
  klaim,
  tugas,
  lockedReason,
}: {
  klaim: Claim
  tugas: Task
  /** Alasan isian dikunci — tugas bukan milik pengguna ini. */
  lockedReason: string | null
}) {
  const receivers = klaim.penerima_klaim ?? []
  // Baris yang sedang dibuka: IDRECEIVER, atau 'baru' untuk baris dari tombol Tambah.
  const [open, setOpen] = useState<string | null>(null)
  const adding = open === NEW

  return (
    <table className="mt-3 w-full border-collapse text-sm">
      <caption className="sr-only">Penerima klaim</caption>
      <thead>
        <tr className="bg-slate-100 text-left text-xs text-slate-700">
          <th className="w-8 p-2" />
          <th className="p-2">Nama</th>
          <th className="p-2">Alamat</th>
          <th className="w-24 p-2">
            <button
              type="button"
              onClick={() => setOpen(NEW)}
              disabled={adding || lockedReason !== null}
              title={lockedReason ?? undefined}
              className="rounded border border-blue-300 px-2 py-0.5 text-xs text-blue-700 hover:bg-blue-50 disabled:opacity-60"
            >
              Tambah
            </button>
          </th>
        </tr>
      </thead>
      <tbody>
        {receivers.length === 0 && !adding && (
          <tr>
            <td colSpan={4} className="p-2 text-xs text-slate-500">
              Data Tidak Ada
            </td>
          </tr>
        )}
        {receivers.map((r, n) => {
          const key = r.id || String(n + 1)
          const expanded = open === key
          return (
            <Fragment key={key}>
              <tr className={['border-b border-slate-100 align-top', expanded ? 'bg-blue-100/70' : ''].join(' ')}>
                <td className="p-2">{n + 1}</td>
                <td className="p-2">
                  <button
                    type="button"
                    aria-expanded={expanded}
                    onClick={() => setOpen(expanded ? null : key)}
                    className="text-left hover:underline"
                  >
                    {r.nama || '—'}
                  </button>
                </td>
                <td className="p-2">{r.alamat || '—'}</td>
                <td className="p-2" />
              </tr>
              {expanded && (
                <tr>
                  <td colSpan={4} className="border border-slate-200 p-3">
                    <ReceiverPane
                      claimID={klaim.id}
                      taskID={tugas.id}
                      receiver={r}
                      lockedReason={lockedReason}
                      onDone={() => setOpen(null)}
                    />
                  </td>
                </tr>
              )}
            </Fragment>
          )
        })}
        {adding && (
          <tr>
            <td colSpan={4} className="border border-slate-200 p-3">
              <ReceiverPane
                claimID={klaim.id}
                taskID={tugas.id}
                receiver={null}
                lockedReason={lockedReason}
                onDone={() => setOpen(null)}
              />
            </td>
          </tr>
        )}
      </tbody>
    </table>
  )
}

const NEW = 'baru'
const EMPTY = '—'

/** Panel InputReceiver satu penerima; `receiver` null untuk baris dari tombol Tambah. */
function ReceiverPane({
  claimID,
  taskID,
  receiver,
  lockedReason,
  onDone,
}: {
  claimID: string
  taskID: string
  receiver: Receiver | null
  lockedReason: string | null
  onDone: () => void
}) {
  const [number, setNumber] = useState(receiver?.nomor_rekening ?? '')
  // Nomor yang dibaca dari master — diperbarui saat isian ditinggalkan, seperti event
  // change pada kontrol Pega (bukan setiap ketikan).
  const [lookup, setLookup] = useState(receiver?.nomor_rekening ?? '')
  const [email, setEmail] = useState('')
  const [phone, setPhone] = useState('')
  const [touched, setTouched] = useState({ email: false, phone: false })

  const account = useBankAccount(lookup)
  const save = useSaveReceiver(claimID)
  const master = account.data

  // Email dan Telepon diisi dari master selama petugas belum mengubahnya.
  useEffect(() => {
    if (!master) return
    if (!touched.email) setEmail(master.email)
    if (!touched.phone) setPhone(master.telepon)
  }, [master, touched.email, touched.phone])

  const locked = lockedReason !== null
  const violations = violationsFrom(save.error)
  const fieldMessages = messagesByField(violations)
  const lookupMessage =
    account.error instanceof APIError ? account.error.message : account.error ? 'Master Rekening cannot be read.' : null

  // Sebelum master terbaca (atau bila nomornya belum diubah), tampilkan yang tersimpan.
  const shown = master ?? {
    nama: receiver?.nama ?? '',
    nama_bank: receiver?.nama_bank ?? '',
    alamat: receiver?.alamat ?? '',
    nama_cabang_bank: '',
    tanggal_approve_kasir: '',
    tanggal_approve_komite: '',
  }

  function submit() {
    save.mutate(
      { tugas_id: taskID, id: receiver?.id ?? '', nomor_rekening: number.trim(), email: email.trim(), telepon: phone.trim() },
      { onSuccess: onDone },
    )
  }

  return (
    <div role="group" aria-label="InputReceiver">
      <div className="grid gap-x-8 gap-y-3 sm:grid-cols-2">
        <div className="space-y-3">
          <Input
            label="No Rekening"
            value={number}
            disabled={locked}
            onChange={setNumber}
            onBlur={() => setLookup(number)}
            message={fieldMessages.nomor_rekening ?? lookupMessage}
          />
          <Display label="Nama Bank">{shown.nama_bank || EMPTY}</Display>
          <Display label="Nama Cabang Bank">{shown.nama_cabang_bank || EMPTY}</Display>
          <Display label="Tanggal Approve Kasir">{dateOrEmpty(shown.tanggal_approve_kasir)}</Display>
          <Input
            label="Telepon"
            value={phone}
            disabled={locked}
            onChange={(v) => {
              setPhone(v)
              setTouched((t) => ({ ...t, phone: true }))
            }}
          />
          <p className="text-sm font-semibold text-slate-800">
            No. Telp agar diisi nomor yang terhubung dengan whatsapp untuk mengirim notfikasi dari kasir jika sudah
            diproses bayar
          </p>
        </div>
        <div className="space-y-3">
          <Display label="Nama">{shown.nama || EMPTY}</Display>
          <Display label="Alamat">{shown.alamat || EMPTY}</Display>
          <Input
            label="Email"
            required
            value={email}
            disabled={locked}
            onChange={(v) => {
              setEmail(v)
              setTouched((t) => ({ ...t, email: true }))
            }}
            message={fieldMessages.email}
          />
          <Display label="Tanggal Approve Komite">{dateOrEmpty(shown.tanggal_approve_komite)}</Display>
        </div>
      </div>

      <p className="mt-3 text-xs text-slate-500">Email and Telephone are not saved yet.</p>

      {save.error && violations.length === 0 && (
        <div className="mt-3">
          <ErrorMessage
            title="Penerima belum dapat disimpan"
            description={save.error instanceof Error ? save.error.message : 'Terjadi kesalahan pada sistem.'}
            tone="penolakan"
          />
        </div>
      )}

      <div className="mt-4 flex gap-3">
        <button
          type="button"
          onClick={submit}
          disabled={locked || save.isPending}
          title={lockedReason ?? undefined}
          className="rounded bg-orange-500 px-3 py-1 text-sm text-white disabled:opacity-60"
        >
          {save.isPending ? 'Menyimpan…' : 'Simpan'}
        </button>
        {receiver === null && (
          <Button tone="halus" disabled={save.isPending} onClick={onDone}>
            Batal
          </Button>
        )}
      </div>
    </div>
  )
}

function dateOrEmpty(value: string): string {
  return value ? formatDate(value) : EMPTY
}

function Display({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div>
      <p className="text-xs font-semibold text-slate-800">{label}</p>
      <p className="text-sm text-slate-700">{children}</p>
    </div>
  )
}

function Input({
  label,
  value,
  onChange,
  onBlur,
  disabled,
  required,
  message,
}: {
  label: string
  value: string
  onChange: (v: string) => void
  onBlur?: () => void
  disabled: boolean
  required?: boolean
  message?: string | null | undefined
}) {
  return (
    <label className="block text-xs font-semibold text-slate-800">
      {label}
      {required && <span className="text-orange-500"> *</span>}
      <input
        value={value}
        disabled={disabled}
        onChange={(e) => onChange(e.target.value)}
        onBlur={onBlur}
        aria-invalid={message ? true : undefined}
        className="mt-1 block w-full rounded border border-slate-300 px-2 py-1 text-sm font-normal disabled:bg-slate-50"
      />
      {message && <span className="mt-1 block font-normal text-red-700">{message}</span>}
    </label>
  )
}
