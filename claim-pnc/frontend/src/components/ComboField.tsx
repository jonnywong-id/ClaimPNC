import { forwardRef, useId, type InputHTMLAttributes } from 'react'

type Props = InputHTMLAttributes<HTMLInputElement> & {
  id: string
  label: string
  /** Saran yang ditawarkan; pengguna TETAP boleh mengetik nilai di luar daftar ini. */
  options: string[]
  /** Pesan kesalahan validasi; bila terisi, kolom ditandai dan pesannya ditampilkan. */
  error?: string | undefined
  /** Keterangan singkat di bawah isian. Disembunyikan saat ada pesan galat. */
  hint?: string | undefined
}

/**
 * ComboField adalah isian teks yang menawarkan saran tetapi TIDAK memaksanya.
 *
 * # Kenapa ia ada, dan kenapa bukan SelectField
 *
 * Isian Bisnis pada layar Master COL Simas Online di Pega adalah autocomplete
 * ber-`pyAllowFreeFormInput=true`
 * (`Section/Online_BrowseCauseOfLoss-Section.xml:6089`): daftarnya membantu, tetapi
 * petugas boleh mengetik nama yang tidak ada di dalamnya. Work Owner menetapkan
 * 2026-09-21 perilaku itu dipertahankan.
 *
 * SelectField tidak dapat menirunya — dropdown menutup nilai di luar daftar, dan itu
 * mengubah perilaku layar. Yang dibutuhkan adalah `<input list>` + `<datalist>`:
 * padanan HTML baku yang paling dekat, tanpa pustaka tambahan dan tanpa menuliskan
 * sendiri penanganan papan ketik yang sudah disediakan peramban.
 *
 * Bentuk, kelas Tailwind, dan penandaan aria-nya dijaga sama dengan Field dan
 * SelectField supaya satu form tidak terlihat seperti dirakit dari dua aplikasi berbeda.
 */
export const ComboField = forwardRef<HTMLInputElement, Props>(function ComboField(
  { id, label, options, error, hint, className, ...rest },
  ref,
) {
  // ID datalist diturunkan dari useId, bukan dari `id` isiannya, supaya dua isian yang
  // kebetulan bernama sama pada satu halaman tidak berbagi satu daftar saran.
  const listID = `${useId()}-daftar`

  const baseClass =
    'mt-1 w-full rounded-kontrol border bg-white px-3 py-2 text-slate-900 shadow-lembut ' +
    'transition-[border-color,box-shadow] duration-150 ease-halus ' +
    'focus:outline-none focus-visible:ring-4 ' +
    (error
      ? 'border-red-400 focus-visible:border-red-500 focus-visible:ring-red-500/25'
      : 'border-slate-300 focus-visible:border-blue-500 focus-visible:ring-blue-500/25')

  return (
    <div>
      <label htmlFor={id} className="block text-sm font-medium text-slate-700">
        {label}
      </label>

      <input
        {...rest}
        id={id}
        ref={ref}
        type="text"
        list={listID}
        // autoComplete dimatikan supaya saran peramban dari riwayat pengisian tidak
        // bercampur dengan daftar bisnis yang sesungguhnya.
        autoComplete="off"
        aria-invalid={error ? true : undefined}
        aria-describedby={error ? `${id}-galat` : hint ? `${id}-petunjuk` : undefined}
        className={[baseClass, className].filter(Boolean).join(' ')}
      />

      <datalist id={listID}>
        {options.map((option) => (
          <option key={option} value={option} />
        ))}
      </datalist>

      {error ? (
        <p id={`${id}-galat`} className="mt-1 text-sm text-red-600" role="alert">
          {error}
        </p>
      ) : hint ? (
        <p id={`${id}-petunjuk`} className="mt-1 text-sm text-slate-500">
          {hint}
        </p>
      ) : null}
    </div>
  )
})
