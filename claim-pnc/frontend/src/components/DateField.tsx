import { useEffect, useRef, useState } from 'react'

import { Field } from './Field'

type Props = {
  id: string
  label: string
  /** Tanggal ISO `YYYY-MM-DD`; teks kosong berarti tidak diisi. */
  value: string
  /** Dipanggil dengan tanggal ISO saat isian lengkap dan sah, atau '' saat dikosongkan. */
  onChange: (iso: string) => void
  error?: string | undefined
  disabled?: boolean
}

/**
 * DateField adalah isian tanggal berformat DD/MM/YYYY, dengan label dan tombol kalender.
 *
 * # Kenapa bukan `<input type="date">` saja
 *
 * Placeholder isian tanggal bawaan dibuat BROWSER menurut bahasanya, bukan oleh halaman —
 * pada Chrome berbahasa Indonesia tampil "dd/mm/tttt", dan tidak ada atribut yang dapat
 * mengubahnya. Di sini isiannya teks biasa sehingga placeholder-nya tetap DD/MM/YYYY di
 * browser mana pun, sedangkan kalender bawaan tetap tersedia lewat tombol di sisi kanan.
 *
 * Nilai keluar-masuknya tetap ISO, sehingga pemakai cukup mengganti komponen tanpa mengubah
 * bentuk penyaringnya.
 */
export function DateField({ id, label, value, onChange, error, disabled }: Readonly<Props>) {
  const d = useDateText(value, onChange)
  return (
    <div className="relative">
      <Field
        id={id}
        label={label}
        inputMode="numeric"
        placeholder={PLACEHOLDER}
        maxLength={10}
        autoComplete="off"
        value={d.text}
        disabled={disabled}
        error={error ?? (d.invalid ? 'Tanggal tidak sah.' : undefined)}
        className="pr-10"
        onChange={(e) => d.type(e.target.value)}
      />
      <CalendarButton label={label} disabled={disabled} onOpen={d.open} className="h-[42px]" />
      <HiddenPicker d={d} value={value} />
    </div>
  )
}

/**
 * DateInput adalah isian DD/MM/YYYY TANPA label dan pesan galat — untuk form yang menata
 * labelnya sendiri (mis. AcceptationLOD). Perilakunya sama dengan DateField.
 */
export function DateInput({
  value,
  onChange,
  className,
  disabled,
  "aria-label": ariaLabel,
}: Readonly<{
  value: string
  onChange: (iso: string) => void
  className?: string
  disabled?: boolean
  "aria-label"?: string
}>) {
  const d = useDateText(value, onChange)
  return (
    <span className="relative block">
      <input
        type="text"
        inputMode="numeric"
        placeholder={PLACEHOLDER}
        maxLength={10}
        autoComplete="off"
        aria-label={ariaLabel}
        aria-invalid={d.invalid ? 'true' : 'false'}
        value={d.text}
        disabled={disabled}
        onChange={(e) => d.type(e.target.value)}
        className={[className ?? '', 'pr-9', d.invalid ? 'border-red-400' : ''].join(' ')}
      />
      <CalendarButton label={ariaLabel ?? 'tanggal'} disabled={disabled} onOpen={d.open} className="h-full" />
      <HiddenPicker d={d} value={value} />
    </span>
  )
}

const PLACEHOLDER = 'DD/MM/YYYY'

type DateText = ReturnType<typeof useDateText>

/** useDateText menyimpan teks yang diketik dan menerjemahkannya ke tanggal ISO. */
function useDateText(value: string, onChange: (iso: string) => void) {
  const [text, setText] = useState(() => isoToText(value))
  const picker = useRef<HTMLInputElement>(null)

  // Nilai dari luar (mis. tombol Bersihkan) menimpa teks — kecuali teks yang sedang diketik
  // sudah mewakili nilai yang sama.
  useEffect(() => {
    setText((current) => (textToISO(current) === value ? current : isoToText(value)))
  }, [value])

  return {
    text,
    picker,
    invalid: text.replace(/\D/g, '').length === 8 && textToISO(text) === '',
    type(raw: string) {
      const next = mask(raw)
      setText(next)
      const iso = textToISO(next)
      if (iso !== '' || next === '') onChange(iso)
    },
    pick(iso: string) {
      setText(isoToText(iso))
      onChange(iso)
    },
    open() {
      const el = picker.current
      if (!el) return
      if (typeof el.showPicker === 'function') el.showPicker()
      else el.click()
    },
  }
}

function CalendarButton({ label, disabled, onOpen, className }: Readonly<{ label: string; disabled?: boolean | undefined; onOpen: () => void; className: string }>) {
  return (
    <button
      type="button"
      aria-label={`Pilih ${label} dari kalender`}
      disabled={disabled}
      onClick={onOpen}
      className={`absolute bottom-0 right-0 flex w-9 items-center justify-center text-slate-400 hover:text-slate-600 disabled:cursor-not-allowed ${className}`}
    >
      <svg viewBox="0 0 20 20" aria-hidden="true" className="h-4 w-4 fill-current">
        <path d="M6 2a1 1 0 0 1 1 1v1h6V3a1 1 0 1 1 2 0v1h1a2 2 0 0 1 2 2v10a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2h1V3a1 1 0 0 1 1-1Zm10 6H4v8h12V8Z" />
      </svg>
    </button>
  )
}

/** Kalender bawaan, tidak terlihat: hanya sumber pilihan tanggal. */
function HiddenPicker({ d, value }: Readonly<{ d: DateText; value: string }>) {
  return (
    <input
      ref={d.picker}
      type="date"
      tabIndex={-1}
      aria-hidden="true"
      value={value}
      onChange={(e) => d.pick(e.target.value)}
      className="pointer-events-none absolute bottom-0 right-0 h-0 w-0 opacity-0"
    />
  )
}

/** mask menyisipkan garis miring saat mengetik: "01092026" → "01/09/2026". */
function mask(raw: string): string {
  const digits = raw.replace(/\D/g, '').slice(0, 8)
  if (digits.length <= 2) return digits
  if (digits.length <= 4) return `${digits.slice(0, 2)}/${digits.slice(2)}`
  return `${digits.slice(0, 2)}/${digits.slice(2, 4)}/${digits.slice(4)}`
}

/** textToISO mengubah "DD/MM/YYYY" menjadi "YYYY-MM-DD"; '' bila belum lengkap atau tidak sah. */
export function textToISO(text: string): string {
  const m = /^(\d{2})\/(\d{2})\/(\d{4})$/.exec(text)
  if (!m) return ''
  const [, dd, mm, yyyy] = m
  const d = new Date(Date.UTC(Number(yyyy), Number(mm) - 1, Number(dd)))
  if (d.getUTCFullYear() !== Number(yyyy) || d.getUTCMonth() !== Number(mm) - 1 || d.getUTCDate() !== Number(dd)) {
    return ''
  }
  return `${yyyy}-${mm}-${dd}`
}

/** isoToText mengubah "YYYY-MM-DD" menjadi "DD/MM/YYYY". */
export function isoToText(iso: string): string {
  const m = /^(\d{4})-(\d{2})-(\d{2})$/.exec(iso)
  return m ? `${m[3]}/${m[2]}/${m[1]}` : ''
}
