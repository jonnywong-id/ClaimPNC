import type { ReactNode } from 'react'

/**
 * Bagian tata letak bersama form master yang panjang (Bengkel, Panel, Sparepart, Grouping
 * Sparepart, Supplier).
 */

/**
 * FieldGroup membungkus sekumpulan isian di bawah satu judul.
 *
 * Ia `<fieldset>` dan bukan `<div>` supaya pembaca layar mengumumkan judulnya saat kursor
 * masuk ke salah satu isian di dalamnya.
 */
export function FieldGroup({ title, children }: Readonly<{ title: string; children: ReactNode }>) {
  return (
    <fieldset className="space-y-4 rounded-kontrol border border-slate-200 p-4">
      <legend className="px-1 text-sm font-semibold text-slate-700">{title}</legend>
      {children}
    </fieldset>
  )
}

type InfoProps = { label: string; value: string; hint?: string }

/** ReadOnlyInfo adalah keterangan baca-saja pada kepala form mode ubah. */
export function ReadOnlyInfo({ label, value, hint }: Readonly<InfoProps>) {
  return (
    <div>
      <span className="block text-xs font-medium uppercase tracking-wide text-slate-500">
        {label}
      </span>
      <span className="mt-0.5 block text-sm text-slate-900">{value}</span>
      {hint && <span className="mt-1 block text-xs text-slate-500">{hint}</span>}
    </div>
  )
}

/**
 * WaitingApprovalNote mengingatkan bahwa baris yang disimpan selalu kembali menunggu
 * persetujuan — persetujuan sebelumnya tidak berlaku atas isi yang sudah berubah.
 */
export function WaitingApprovalNote() {
  return (
    <p className="text-xs text-slate-500">
      Baris yang disimpan selalu kembali ke <span className="font-medium">Waiting Approval</span>{' '}
      dan menunggu persetujuan — persetujuan sebelumnya tidak berlaku atas isi yang sudah
      berubah.
    </p>
  )
}

type SuggestionsProps = {
  /** Awalan `id` datalist, mis. "bengkel" atau "panel". */
  prefix: string
  name: string
  values: string[]
}

/**
 * Suggestions menggambar daftar saran untuk sebuah isian.
 *
 * Isinya berasal dari nilai yang SUDAH DIPAKAI baris lain pada entitas yang sedang dibuka
 * — bukan dari daftar yang dikarang. Ia menjawab pertanyaan yang tidak dapat dijawab
 * export Pega ("nilai apa yang sah di kolom ini?") dengan satu-satunya sumber yang
 * tersedia: data itu sendiri.
 *
 * Tidak menggambar apa pun bila belum ada nilai yang terpakai, supaya daftar kosong tidak
 * muncul sebagai kotak saran yang selalu hampa.
 */
export function Suggestions({ prefix, name, values }: Readonly<SuggestionsProps>) {
  if (values.length === 0) return null

  return (
    <datalist id={`${prefix}-${name}`}>
      {values.map((value) => (
        <option key={value} value={value} />
      ))}
    </datalist>
  )
}
