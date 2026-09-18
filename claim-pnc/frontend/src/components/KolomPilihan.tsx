import { forwardRef, type SelectHTMLAttributes } from 'react'

/** Satu pilihan pada dropdown. */
export type Pilihan = {
  nilai: string
  label: string
}

type Props = SelectHTMLAttributes<HTMLSelectElement> & {
  id: string
  label: string
  pilihan: Pilihan[]
  /** Pesan kesalahan validasi; bila terisi, kolom ditandai dan pesannya ditampilkan. */
  galat?: string | undefined
  /**
   * Teks pilihan kosong di puncak daftar.
   *
   * Ia ada supaya isian yang belum dipilih terlihat sebagai "belum dipilih" — bukan
   * diam-diam terisi pilihan pertama. Dropdown yang langsung menunjuk pilihan pertama
   * membuat pengguna menyimpan nilai yang tidak pernah ia pilih.
   */
  teksKosong?: string
}

/**
 * KolomPilihan adalah satu baris isian berupa dropdown: label, pilihan, dan pesan
 * kesalahannya.
 *
 * Ia pasangan KolomIsian untuk isian yang nilainya dipilih dari daftar, dan dibuat
 * untuk dipakai langsung dengan React Hook Form:
 *
 *	<KolomPilihan id="kode_posisi" label="Posisi" pilihan={pilihanPosisi}
 *	              galat={errors.kode_posisi?.message} {...register('kode_posisi')} />
 *
 * Bentuk, kelas Tailwind, dan penandaan aria-nya dijaga sama dengan KolomIsian supaya
 * sebuah form tidak terlihat seperti dirakit dari dua aplikasi berbeda.
 */
export const KolomPilihan = forwardRef<HTMLSelectElement, Props>(function KolomPilihan(
  { id, label, pilihan, galat, teksKosong = '— pilih —', className, ...sisa },
  ref,
) {
  const kelasDasar =
    'mt-1 w-full rounded border bg-white px-3 py-2 text-slate-900 focus:outline-none ' +
    (galat ? 'border-red-400 focus:border-red-500' : 'border-slate-300 focus:border-slate-500')

  return (
    <div>
      <label htmlFor={id} className="block text-sm font-medium text-slate-700">
        {label}
      </label>
      <select
        id={id}
        ref={ref}
        aria-invalid={galat ? 'true' : 'false'}
        aria-describedby={galat ? `${id}-galat` : undefined}
        className={className ? `${kelasDasar} ${className}` : kelasDasar}
        {...sisa}
      >
        <option value="">{teksKosong}</option>
        {pilihan.map((p) => (
          <option key={p.nilai} value={p.nilai}>
            {p.label}
          </option>
        ))}
      </select>
      {galat && (
        <p id={`${id}-galat`} className="mt-1 text-sm text-red-700">
          {galat}
        </p>
      )}
    </div>
  )
})
