import { forwardRef, type TextareaHTMLAttributes } from 'react'

type Props = TextareaHTMLAttributes<HTMLTextAreaElement> & {
  id: string
  label: string
  /** Pesan kesalahan validasi; bila terisi, kolom ditandai dan pesannya ditampilkan. */
  error?: string | undefined
  /** Keterangan singkat di bawah isian. Disembunyikan saat ada pesan galat. */
  hint?: string | undefined
}

/**
 * TextAreaField adalah saudara `Field` untuk isian bertingkat banyak baris.
 *
 * # Kenapa ia komponen tersendiri, bukan sakelar pada Field
 *
 * `<textarea>` dan `<input>` adalah dua elemen berbeda dengan atribut yang berbeda —
 * `rows` di satu sisi, `maxLength` berperilaku beda di sisi lain — dan menyatukannya
 * memaksa tipe props menjadi gabungan yang tidak dapat diperiksa TypeScript dengan benar.
 * Dua komponen yang berbagi tampilan lebih jujur daripada satu komponen yang berbagi
 * kebingungan.
 *
 * # Tampilannya sengaja SAMA PERSIS dengan Field
 *
 * Kelas tepinya, cincin fokusnya, dan cara keadaan salah ditandai diambil dari sana kata
 * per kata. Satu form dapat memuat keduanya berdampingan — form Pelaporan Klaim memuat
 * tiga belas isian pendek dan empat isian panjang — dan bila keduanya terlihat berbeda,
 * formnya terbaca seperti tambal sulam.
 *
 * Nama props-nya TIDAK sama: di sini `error`/`hint` (bahasa Inggris, `D-80`), sementara
 * `Field` memakai `galat`/`petunjuk` karena ia ditulis sebelum `D-80` dan berada di
 * luar lingkup pekerjaan ini. Satu berkas form karenanya memuat keduanya berdampingan.
 *
 * # Keadaan salah ditandai TIGA cara sekaligus
 *
 * Warna tepi, ikon peringatan pada pesannya, dan teks pesan itu sendiri. Warna saja tidak
 * cukup: sekitar satu dari dua belas laki-laki mengalami buta warna merah-hijau, dan bagi
 * mereka tepi merah tidak berbeda dari tepi abu-abu.
 *
 * `aria-invalid` dan `aria-describedby` menyampaikan hal yang sama kepada pembaca layar,
 * sehingga pesannya dibacakan saat kursor masuk ke kolom — bukan hanya terlihat.
 */
export const TextAreaField = forwardRef<HTMLTextAreaElement, Props>(
  function TextAreaField({ id, label, error, hint, className, rows = 3, ...rest }, ref) {
    const fieldClass = [
      'w-full rounded-kontrol border bg-white px-3 py-2.5 text-sm text-slate-900',
      'placeholder:text-slate-400',
      'transition-[border-color,box-shadow,background-color] duration-150 ease-halus',
      'focus:outline-none focus:ring-4',
      'disabled:cursor-not-allowed disabled:bg-slate-50 disabled:text-slate-500',
      // Hanya tinggi yang boleh diubah pengguna. Lebar yang dapat ditarik akan merusak
      // tata letak kolom di sebelahnya, dan tidak ada yang membutuhkannya.
      'resize-y',
      error
        ? 'border-red-400 focus:border-red-500 focus:ring-red-500/15'
        : 'border-slate-300 hover:border-slate-400 focus:border-blue-500 focus:ring-blue-500/15',
    ].join(' ')

    return (
      <div>
        <label htmlFor={id} className="block text-sm font-medium text-slate-700">
          {label}
        </label>

        <textarea
          id={id}
          ref={ref}
          rows={rows}
          aria-invalid={error ? 'true' : 'false'}
          aria-describedby={error ? `${id}-galat` : hint ? `${id}-petunjuk` : undefined}
          className={className ? `${fieldClass} ${className} mt-1.5` : `${fieldClass} mt-1.5`}
          {...rest}
        />

        {error ? (
          <p id={`${id}-galat`} className="mt-1.5 flex items-start gap-1.5 text-sm text-red-700">
            <svg
              viewBox="0 0 16 16"
              aria-hidden="true"
              className="mt-0.5 h-3.5 w-3.5 shrink-0 fill-current"
            >
              <path d="M8 1.5a6.5 6.5 0 1 0 0 13 6.5 6.5 0 0 0 0-13Zm0 3a.8.8 0 0 1 .8.8v3.4a.8.8 0 0 1-1.6 0V5.3a.8.8 0 0 1 .8-.8Zm0 6.2a.9.9 0 1 1 0 1.8.9.9 0 0 1 0-1.8Z" />
            </svg>
            <span>{error}</span>
          </p>
        ) : (
          hint && (
            <p id={`${id}-petunjuk`} className="mt-1.5 text-xs text-slate-500">
              {hint}
            </p>
          )
        )}
      </div>
    )
  },
)
