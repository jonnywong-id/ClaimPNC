import { forwardRef, type InputHTMLAttributes, type ReactNode } from 'react'

type Props = InputHTMLAttributes<HTMLInputElement> & {
  id: string
  label: string
  /** Pesan kesalahan validasi; bila terisi, kolom ditandai dan pesannya ditampilkan. */
  galat?: string | undefined
  /** Keterangan singkat di bawah isian. Disembunyikan saat ada pesan galat. */
  petunjuk?: string | undefined
  /** Ikon kecil di dalam kolom, sisi kiri. Murni hiasan; label tetap yang menjelaskan. */
  ikon?: ReactNode
}

/**
 * KolomIsian adalah satu baris isian: label, input, dan pesan kesalahannya.
 *
 * Ia dibuat untuk dipakai langsung dengan React Hook Form:
 *
 *	<KolomIsian id="namaPengguna" label="Nama pengguna"
 *	            galat={errors.namaPengguna?.message} {...register('namaPengguna')} />
 *
 * Ref diteruskan supaya `register` dapat memegang elemen inputnya.
 *
 * Alasan keberadaannya: tanpa komponen ini, setiap layar menulis ulang label, kelas
 * Tailwind, dan penandaan aria-invalid sendiri — dan pada 74 layar itu berubah menjadi
 * 74 tafsir berbeda tentang bagaimana sebuah kolom terlihat saat salah.
 *
 * # Keadaan salah ditandai TIGA cara sekaligus
 *
 * Warna tepi, ikon peringatan pada pesannya, dan teks pesan itu sendiri. Warna saja
 * tidak cukup: sekitar satu dari dua belas laki-laki mengalami buta warna merah-hijau,
 * dan bagi mereka tepi merah tidak berbeda dari tepi abu-abu.
 *
 * `aria-invalid` dan `aria-describedby` menyampaikan hal yang sama kepada pembaca layar,
 * sehingga pesannya dibacakan saat kursor masuk ke kolom — bukan hanya terlihat.
 */
export const KolomIsian = forwardRef<HTMLInputElement, Props>(function KolomIsian(
  { id, label, galat, petunjuk, ikon, className, disabled, ...sisa },
  ref,
) {
  const kelasInput = [
    'w-full rounded-kontrol border bg-white py-2.5 text-sm text-slate-900',
    ikon ? 'pl-10 pr-3' : 'px-3',
    'placeholder:text-slate-400',
    'transition-[border-color,box-shadow,background-color] duration-150 ease-halus',
    'focus:outline-none focus:ring-4',
    'disabled:cursor-not-allowed disabled:bg-slate-50 disabled:text-slate-500',
    galat
      ? 'border-red-400 focus:border-red-500 focus:ring-red-500/15'
      : 'border-slate-300 hover:border-slate-400 focus:border-blue-500 focus:ring-blue-500/15',
  ].join(' ')

  return (
    <div>
      <label htmlFor={id} className="block text-sm font-medium text-slate-700">
        {label}
      </label>

      <div className="relative mt-1.5">
        {ikon && (
          <span
            aria-hidden="true"
            className={
              'pointer-events-none absolute inset-y-0 left-0 flex w-10 items-center justify-center ' +
              (galat ? 'text-red-400' : 'text-slate-400')
            }
          >
            {ikon}
          </span>
        )}
        <input
          id={id}
          ref={ref}
          disabled={disabled}
          aria-invalid={galat ? 'true' : 'false'}
          aria-describedby={galat ? `${id}-galat` : petunjuk ? `${id}-petunjuk` : undefined}
          className={className ? `${kelasInput} ${className}` : kelasInput}
          {...sisa}
        />
      </div>

      {galat ? (
        <p id={`${id}-galat`} className="mt-1.5 flex items-start gap-1.5 text-sm text-red-700">
          <svg
            viewBox="0 0 16 16"
            aria-hidden="true"
            className="mt-0.5 h-3.5 w-3.5 shrink-0 fill-current"
          >
            <path d="M8 1.5a6.5 6.5 0 1 0 0 13 6.5 6.5 0 0 0 0-13Zm0 3a.8.8 0 0 1 .8.8v3.4a.8.8 0 0 1-1.6 0V5.3a.8.8 0 0 1 .8-.8Zm0 6.2a.9.9 0 1 1 0 1.8.9.9 0 0 1 0-1.8Z" />
          </svg>
          <span>{galat}</span>
        </p>
      ) : (
        petunjuk && (
          <p id={`${id}-petunjuk`} className="mt-1.5 text-xs text-slate-500">
            {petunjuk}
          </p>
        )
      )}
    </div>
  )
})
