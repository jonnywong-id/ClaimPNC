import { forwardRef, type SelectHTMLAttributes } from 'react'

/** Satu pilihan pada dropdown. */
export type SelectOption = {
  value: string
  label: string
}

type Props = SelectHTMLAttributes<HTMLSelectElement> & {
  id: string
  label: string
  options: SelectOption[]
  /** Pesan kesalahan validasi; bila terisi, kolom ditandai dan pesannya ditampilkan. */
  error?: string | undefined
  /**
   * Teks pilihan kosong di puncak daftar.
   *
   * Ia ada supaya isian yang belum dipilih terlihat sebagai "belum dipilih" — bukan
   * diam-diam terisi pilihan pertama. Dropdown yang langsung menunjuk pilihan pertama
   * membuat pengguna menyimpan nilai yang tidak pernah ia pilih.
   */
  emptyText?: string

  /**
   * Apakah pilihan kosong di puncak daftar ikut digambar. Bawaannya **true**.
   *
   * # Kapan ia dimatikan, dan kenapa itu jarang
   *
   * Hanya ketika layar Pega yang digantikan memang TIDAK punya pilihan kosong, dan
   * nilainya SELALU terisi. Dropdown tanpa pilihan kosong akan menampilkan pilihan
   * pertama sebagai terpilih untuk nilai apa pun yang tidak dikenalnya — termasuk nilai
   * kosong — sehingga baris lama yang kolomnya belum pernah diisi akan **berubah diam-diam**
   * begitu pengguna menekan Simpan.
   *
   * Pemanggil yang mematikannya karena itu wajib memastikan sendiri bahwa nilai kosong
   * tidak mungkin sampai ke sini, atau menyalakannya kembali khusus untuk baris semacam itu
   * (lihat `detail-penyebab-kerugian/DetailForm.tsx`).
   *
   * Bawaannya `true` supaya penambahan prop ini tidak mengubah satu pun dari 43 layar yang
   * sudah memakai SelectField.
   */
  includeEmpty?: boolean
}

/**
 * SelectField adalah satu baris isian berupa dropdown: label, pilihan, dan pesan
 * kesalahannya.
 *
 * Ia pasangan Field untuk isian yang nilainya dipilih dari daftar, dan dibuat untuk
 * dipakai langsung dengan React Hook Form:
 *
 *	<SelectField id="kode_posisi" label="Posisi" options={positionOptions}
 *	             error={errors.kode_posisi?.message} {...register('kode_posisi')} />
 *
 * Bentuk, kelas Tailwind, dan penandaan aria-nya dijaga sama dengan Field supaya sebuah
 * form tidak terlihat seperti dirakit dari dua aplikasi berbeda.
 */
export const SelectField = forwardRef<HTMLSelectElement, Props>(function SelectField(
  { id, label, options, error, emptyText = '— pilih —', includeEmpty = true, className, ...rest },
  ref,
) {
  const baseClass =
    'mt-1 w-full rounded-kontrol border bg-white px-3 py-2 text-slate-900 shadow-lembut ' +
    'transition-[border-color,box-shadow] duration-150 ease-halus ' +
    'focus:outline-none focus-visible:ring-4 ' +
    (error
      ? 'border-red-400 focus:border-red-500 focus-visible:ring-red-500/20'
      : 'border-slate-300 focus:border-blue-500 focus-visible:ring-blue-500/20')

  return (
    <div>
      <label htmlFor={id} className="block text-sm font-medium text-slate-700">
        {label}
      </label>
      <select
        id={id}
        ref={ref}
        aria-invalid={error ? 'true' : 'false'}
        aria-describedby={error ? `${id}-error` : undefined}
        className={className ? `${baseClass} ${className}` : baseClass}
        {...rest}
      >
        {includeEmpty && <option value="">{emptyText}</option>}
        {/*
          Kunci memakai POSISI, bukan nilainya.

          Daftar pilihan boleh memuat nilai kembar — dropdown Posisi pada Master Status
          Progres 1 memuat "All" dua kali, dan itu direplikasi apa adanya dari layar Pega
          (lihat internal/masterstatusprogres/position.go). Mengunci dengan `option.value`
          membuat React menemui dua kunci yang sama dalam satu daftar, dan itu memicu
          peringatan sekaligus penggambaran ulang yang tidak dapat diandalkan.

          Urutan daftar ini ditentukan server dan tidak pernah disusun ulang di layar,
          sehingga kunci berbasis posisi aman di sini — ia tidak aman pada daftar yang
          barisnya dapat dipindah pengguna.
        */}
        {options.map((option, position) => (
          <option key={`${position}-${option.value}`} value={option.value}>
            {option.label}
          </option>
        ))}
      </select>
      {error && (
        <p id={`${id}-error`} className="mt-1 text-sm text-red-700">
          {error}
        </p>
      )}
    </div>
  )
})
