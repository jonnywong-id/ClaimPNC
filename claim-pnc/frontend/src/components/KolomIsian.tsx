import { forwardRef, type InputHTMLAttributes } from 'react'

type Props = InputHTMLAttributes<HTMLInputElement> & {
  id: string
  label: string
  /** Pesan kesalahan validasi; bila terisi, kolom ditandai dan pesannya ditampilkan. */
  galat?: string | undefined
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
 */
export const KolomIsian = forwardRef<HTMLInputElement, Props>(function KolomIsian(
  { id, label, galat, className, ...sisa },
  ref,
) {
  const kelasDasar =
    'mt-1 w-full rounded border px-3 py-2 text-slate-900 focus:outline-none ' +
    (galat
      ? 'border-red-400 focus:border-red-500'
      : 'border-slate-300 focus:border-slate-500')

  return (
    <div>
      <label htmlFor={id} className="block text-sm font-medium text-slate-700">
        {label}
      </label>
      <input
        id={id}
        ref={ref}
        aria-invalid={galat ? 'true' : 'false'}
        aria-describedby={galat ? `${id}-galat` : undefined}
        className={className ? `${kelasDasar} ${className}` : kelasDasar}
        {...sisa}
      />
      {galat && (
        <p id={`${id}-galat`} className="mt-1 text-sm text-red-700">
          {galat}
        </p>
      )}
    </div>
  )
})
