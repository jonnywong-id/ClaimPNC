import { useEffect, useRef, type FormEventHandler, type ReactNode } from 'react'
import type { UseFormRegisterReturn } from 'react-hook-form'

import { APIError, NetworkError } from '@/api/client'
import { ErrorCode } from '@/api/types'
import { Button } from '@/components/Button'
import { Field } from '@/components/Field'
import { ErrorMessage } from '@/components/ErrorMessage'

import { messageForCode, type CodeMessage, type CodeMessages, type MessageContent } from './saveErrorMessage'

/**
 * Bagian bersama form master bergaya "panel" — form tambah/ubah satu isian yang muncul di
 * atas tabel (Master Tipe Surveyors, Status Klaim, Dominan Factor, Penyebab Kerugian).
 *
 * Panel ini muncul di atas tabel, bukan sebagai dialog melayang: pengguna sering perlu
 * melihat baris lain yang sudah ada, dan dialog yang menutup layar justru menyembunyikan
 * jawabannya. Garis aksen di tepi kiri menandai bahwa panel ini keadaan sementara, bukan
 * bagian tetap halaman.
 */

type FrameProps = {
  onSubmit: FormEventHandler<HTMLFormElement>
  ariaLabel: string
  title: string
  subtitle: string
  children: ReactNode
}

/** PanelFormFrame menggambar bingkai panel: kepala berjudul dan badan berisi isian. */
export function PanelFormFrame({ onSubmit, ariaLabel, title, subtitle, children }: Readonly<FrameProps>) {
  return (
    <form
      onSubmit={onSubmit}
      noValidate
      className="overflow-hidden rounded-kartu border border-slate-200 border-l-4 border-l-blue-500 bg-white shadow-angkat"
      aria-label={ariaLabel}
    >
      <div className="border-b border-slate-100 bg-slate-50/70 px-5 py-4">
        <h3 className="text-base font-semibold text-slate-900">{title}</h3>
        <p className="mt-1 text-sm text-slate-600">{subtitle}</p>
      </div>

      <div className="space-y-5 p-5">{children}</div>
    </form>
  )
}

type CodeBoxProps = {
  /** Label kotaknya, mis. "Kode" atau "ID". */
  label: string
  /** Nilai yang ditampilkan; kosong berarti barisnya belum tersimpan. */
  value: string | number | undefined
}

/**
 * ReadOnlyCodeBox menampilkan kode/ID baris sebagai kotak mati.
 *
 * Digambar sebagai kotak, bukan input ber-`disabled`: input yang dinonaktifkan tetap
 * terlihat seperti isian dan mengundang pengguna mengkliknya; kotak ini jelas bukan tempat
 * mengetik.
 */
export function ReadOnlyCodeBox({ label, value }: Readonly<CodeBoxProps>) {
  return (
    <div>
      <span className="block text-sm font-medium text-slate-700">{label}</span>
      <p className="mt-1.5 flex items-center rounded-kontrol border border-dashed border-slate-300 bg-slate-50 px-3 py-2.5 font-mono text-sm text-slate-500">
        {value ?? 'Dibuat sistem'}
      </p>
      <p className="mt-1.5 text-xs text-slate-500">
        {`${label} tidak dapat disunting, sama seperti di sistem lama.`}
      </p>
    </div>
  )
}

type CodeAndFieldProps = {
  /** Label kotak kode/ID read-only, mis. "Kode" atau "ID". */
  codeLabel: string
  codeValue: string | number | undefined
  id: string
  label: string
  placeholder: string
  maxLength: number
  hint: string
  error: string | undefined
  disabled: boolean
  /** Hasil `register` untuk satu-satunya isian form; ia juga menerima fokus pertama. */
  registration: UseFormRegisterReturn
}

/**
 * PanelCodeAndField menggambar baris isian form panel: kotak kode read-only di kiri dan
 * satu isian teks di kanan, yang menerima fokus saat form terbuka.
 */
export function PanelCodeAndField({ codeLabel, codeValue, registration, ...field }: Readonly<CodeAndFieldProps>) {
  const focusTarget = useFocusFirstField()
  return (
    <div className="grid gap-5 sm:grid-cols-2">
      <ReadOnlyCodeBox label={codeLabel} value={codeValue} />
      <Field
        id={field.id}
        label={field.label}
        placeholder={field.placeholder}
        maxLength={field.maxLength}
        autoComplete="off"
        hint={field.hint}
        error={field.error}
        disabled={field.disabled}
        {...focusTarget(registration)}
      />
    </div>
  )
}

/** Peringatan bahwa keterangan ganda diterima sistem — pengganti pemeriksaan yang sengaja tidak ada. */
export function DuplicateAllowedNote({ noun }: Readonly<{ noun: string }>) {
  return (
    <p className="rounded-kontrol border border-amber-200 bg-amber-50 px-3 py-2.5 text-xs leading-relaxed text-amber-900">
      {`${noun} yang sama boleh terdaftar lebih dari satu kali — sistem tidak menolaknya. ` +
        'Periksa daftar di bawah lebih dulu agar tidak terdaftar ganda.'}
    </p>
  )
}

type ActionsProps = {
  isPending: boolean
  onCancel: () => void
}

/** PanelFormActions menggambar tombol Simpan (dengan pemutar saat menyimpan) dan Batal. */
export function PanelFormActions({ isPending, onCancel }: Readonly<ActionsProps>) {
  return (
    <div className="flex flex-wrap gap-2 border-t border-slate-100 pt-5">
      <Button type="submit" tone="utama" disabled={isPending}>
        {isPending && <Spinner />}
        {isPending ? 'Menyimpan…' : 'Simpan'}
      </Button>
      <Button tone="halus" onClick={onCancel} disabled={isPending}>
        Batal
      </Button>
    </div>
  )
}

/** Pemutar kecil pada tombol yang sedang bekerja. */
export function Spinner() {
  return (
    <svg viewBox="0 0 16 16" aria-hidden="true" className="h-4 w-4 animate-spin">
      <circle cx="8" cy="8" r="6.5" fill="none" stroke="currentColor" strokeOpacity="0.3" strokeWidth="2" />
      <path
        d="M8 1.5a6.5 6.5 0 0 1 6.5 6.5"
        fill="none"
        stroke="currentColor"
        strokeWidth="2"
        strokeLinecap="round"
      />
    </svg>
  )
}

/**
 * useFocusFirstField memindahkan fokus ke isian pertama saat form terbuka.
 *
 * Tanpa ini, pengguna papan ketik harus menekan Tab berkali-kali dari awal halaman untuk
 * mencapainya. Kembaliannya membungkus hasil `register` supaya ref isian itu juga dipegang
 * di sini.
 */
export function useFocusFirstField() {
  const firstField = useRef<HTMLInputElement | null>(null)

  useEffect(() => {
    firstField.current?.focus()
  }, [])

  return function focusTarget(registration: UseFormRegisterReturn) {
    const { ref, ...remaining } = registration
    return {
      ...remaining,
      ref: (element: HTMLInputElement | null) => {
        ref(element)
        firstField.current = element
      },
    }
  }
}

/**
 * Galat validasi: pesan dari server dipakai apa adanya — ia yang tahu aturan mana yang
 * dilanggar, dan menerjemahkannya ulang di sini akan membuat keduanya dapat berbeda.
 */
export function panelValidationMessage(error: APIError): MessageContent {
  return {
    title: 'Isian belum benar',
    description: error.detail.map((d) => d.pesan).join(' ') || error.message,
    tone: 'penolakan',
  }
}

const panelPortalNotChosen: MessageContent = {
  title: 'Portal entitas belum dipilih',
  description: 'Pilih portal entitas di bilah atas halaman, lalu simpan lagi.',
  tone: 'penolakan',
}

/** Pesan galat portal entitas pada form panel. */
export const panelPortalMessages: CodeMessages = {
  [ErrorCode.portalNotStated]: panelPortalNotChosen,
  [ErrorCode.portalUnknown]: panelPortalNotChosen,
  [ErrorCode.portalNotReady]: {
    title: 'Basis data entitas ini belum tersedia',
    description:
      'Mengulang tidak akan menolong. Hubungi administrator Claim PNC untuk melengkapi kredensial basis datanya.',
    tone: 'gangguan',
  },
}

/** Kode yang tidak dikenal: pesan server ditampilkan apa adanya. */
export function panelFallbackMessage(error: APIError): MessageContent {
  return { title: 'Gagal menyimpan', description: error.message, tone: 'gangguan' }
}

/** Nomor yang dibuat sistem bentrok dengan penyimpanan lain. */
export const panelIDTakenMessage: MessageContent = {
  title: 'Nomor bentrok',
  description: 'Nomor yang dibuat sistem sedang dipakai penyimpanan lain. Coba simpan sekali lagi.',
  tone: 'gangguan',
}

/** Kode yang dibuat sistem sudah dipakai. */
export const panelCodeTakenMessage: MessageContent = {
  title: 'Kode bentrok',
  description: 'Kode yang dibuat sistem sudah dipakai. Coba simpan sekali lagi.',
  tone: 'gangguan',
}

/** Baris yang disunting tidak ditemukan lagi; `title` menyebut jenis barisnya. */
export function panelNotFoundMessage(title: string): MessageContent {
  return {
    title,
    description: 'Baris ini mungkin sudah diubah orang lain. Muat ulang daftarnya.',
    tone: 'penolakan',
  }
}

type SaveErrorProps = {
  error: unknown
  /** Pesan per kode galat API khusus form ini. */
  messages: CodeMessages
  fallback?: CodeMessage | undefined
}

/**
 * PanelSaveErrorMessage membedakan galat simpan menurut KODE-nya, bukan teks pesannya.
 *
 * Jenisnya menuntut tindak lanjut berbeda: nama yang bentrok dapat diperbaiki pengguna,
 * portal yang belum dipilih diperbaiki di bilah atas, dan gangguan sistem tidak dapat
 * ditolong dengan mencoba ulang berkali-kali.
 */
export function PanelSaveErrorMessage({ error, messages, fallback = panelFallbackMessage }: Readonly<SaveErrorProps>) {
  return <SaveErrorBox error={error} parse={(apiError) => messageForCode(apiError, messages, fallback)} />
}

type SaveErrorBoxProps = {
  error: unknown
  /** Memilih pesan untuk galat API; null berarti tidak ada kotak pesan. */
  parse: (error: APIError) => MessageContent | null
  /** Keterangan kotak galat jaringan; bawaannya untuk perubahan satu baris. */
  networkDescription?: string | undefined
}

/**
 * SaveErrorBox menampilkan galat simpan: galat jaringan dan galat tak terduga memakai pesan
 * tetap, galat API dipilih `parse` menurut kodenya.
 */
export function SaveErrorBox({
  error,
  parse,
  networkDescription = 'Perubahan belum tersimpan. Periksa koneksi lalu coba lagi.',
}: Readonly<SaveErrorBoxProps>) {
  if (error instanceof NetworkError) {
    return <ErrorMessage title="Tidak dapat menghubungi server" description={networkDescription} tone="gangguan" />
  }

  if (!(error instanceof APIError)) {
    return (
      <ErrorMessage
        title="Gagal menyimpan"
        description="Terjadi kesalahan yang tidak terduga. Coba beberapa saat lagi."
        tone="gangguan"
      />
    )
  }

  const message = parse(error)
  if (message === null) return null
  return <ErrorMessage title={message.title} description={message.description} tone={message.tone} />
}
