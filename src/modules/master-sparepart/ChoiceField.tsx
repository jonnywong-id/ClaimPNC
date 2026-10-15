import { useId } from 'react'
import type { SparepartFlagOption } from '@/api/types'

/**
 * ChoiceField adalah isian yang nilainya dipilih dari daftar pilihan rule Property Pega.
 *
 * # Isinya PERSIS daftar Pega
 *
 * Empat isian Master Sparepart — Jenis, Satuan, Status Aktif, dan Status Sparepart —
 * dirender `pxRadioButtons` atau `pxDropdown` di layar Pega, dan pilihannya ada di
 * `pyPromptTableList` rule Property masing-masing pada kelas
 * `ASM-FW-GCNMFW-Int-SPAREPART_HE` (diterima 2026-10-14).
 *
 * Tidak ada pilihan tambahan, tidak ada ketikan bebas, tidak ada nilai yang dikumpulkan dari
 * baris lain. Pega tidak punya ketiganya, jadi layar ini pun tidak — keputusan Work Owner
 * 2026-10-14, menggantikan rancangan sebelumnya yang menambahkan "Lainnya…".
 *
 * # Satu-satunya hal di luar daftar, dan itu bukan tambahan
 *
 * Bila baris yang sedang dibuka memuat nilai yang TIDAK ada di daftar, nilai itu tetap
 * digambar sebagai pilihan terpilih. Itu bukan pilihan karangan melainkan isi baris itu
 * sendiri; tanpanya form akan menampilkan isian kosong untuk baris yang sebenarnya berisi,
 * lalu menyimpannya sebagai kosong tanpa satu pun tanda.
 *
 * Nilai seperti itu memang ada di data — `SATUAN` hanya punya satu pilihan di rule Pega,
 * sementara baris lama memuat satuan lain.
 *
 * Ia tinggal di `modules/master-sparepart/` dan BUKAN di `shared/components/` karena
 * pilihannya khas modul ini.
 */
type Props = {
  /** Label yang dibaca pengguna, mengikuti caption layar Pega. */
  label: string

  /** Nilai yang sedang dipegang isian ini. */
  value: string

  /**
   * Daftar pilihan dari rule Property Pega, pada urutan rule-nya.
   *
   * TIDAK diurutkan ulang: `REPEATINGINDEX` menentukan urutan yang dilihat petugas pada
   * dropdown hari ini.
   */
  choices: SparepartFlagOption[]

  /**
   * Bentuk kontrolnya, mengikuti layar Pega apa adanya.
   *
   * `radio` untuk Jenis Sparepart (`pxRadioButtons`), `dropdown` untuk Satuan dan Status
   * Sparepart (`pxDropdown`).
   */
  variant: 'dropdown' | 'radio'

  disabled?: boolean
  error?: string | undefined
  hint?: string | undefined
  onChange: (value: string) => void
}

export function ChoiceField({
  label,
  value,
  choices,
  variant,
  disabled,
  error,
  hint,
  onChange,
}: Props) {
  const id = useId()
  const errorId = `${id}-error`

  // Daftar Pega apa adanya, TANPA diurutkan ulang.
  //
  // Label kosong jatuh kembali ke kodenya: `SATUAN` punya pilihan tanpa label di Pega, dan
  // menggambarnya sebagai baris kosong membuat dropdown yang mustahil dipilih dengan sadar.
  const options = choices.map((one) => ({ value: one.nilai, text: one.label || one.nilai }))

  // Nilai baris ini sendiri, bila di luar daftar. Lihat catatan komponen: ini isi barisnya,
  // bukan pilihan tambahan.
  if (value !== '' && !choices.some((one) => one.nilai === value)) {
    options.push({ value, text: value })
  }

  const describedBy = error ? errorId : undefined

  return (
    <div>
      {/*
        Dropdown memakai <label htmlFor>, radio memakai <span> di dalam grup ber-aria-label.
        Keduanya dibacakan pembaca layar dengan benar, tetapi hanya yang pertama yang dapat
        ditautkan ke satu elemen — sekumpulan radio tidak punya satu elemen untuk ditunjuk.
      */}
      {variant === 'dropdown' ? (
        <label htmlFor={id} className="block text-sm font-medium text-slate-700">
          {label}
        </label>
      ) : (
        <span className="block text-sm font-medium text-slate-700">{label}</span>
      )}

      {variant === 'dropdown' ? (
        <select
          id={id}
          value={value}
          disabled={disabled}
          aria-invalid={error ? 'true' : 'false'}
          aria-describedby={describedBy}
          onChange={(event) => onChange(event.target.value)}
          className={[
            'mt-1.5 w-full rounded-kontrol border bg-white px-3 py-2.5 text-sm text-slate-900',
            'shadow-lembut transition-[border-color,box-shadow] duration-150 ease-halus',
            'focus:outline-none focus-visible:ring-4',
            'disabled:cursor-not-allowed disabled:bg-slate-50 disabled:text-slate-500',
            error
              ? 'border-red-400 focus:border-red-500 focus-visible:ring-red-500/15'
              : 'border-slate-300 hover:border-slate-400 focus:border-blue-500 focus-visible:ring-blue-500/15',
          ].join(' ')}
        >
          <option value="">— pilih —</option>
          {options.map((one) => (
            <option key={one.value} value={one.value}>
              {one.text}
            </option>
          ))}
        </select>
      ) : (
        <div
          role="radiogroup"
          aria-label={label}
          aria-describedby={describedBy}
          className="mt-1.5 flex flex-wrap items-center gap-x-4 gap-y-2"
        >
          {options.map((one) => (
            <label
              key={one.value}
              className="inline-flex items-center gap-1.5 text-sm text-slate-800"
            >
              <input
                type="radio"
                name={id}
                value={one.value}
                checked={value === one.value}
                disabled={disabled}
                onChange={() => onChange(one.value)}
                className="h-4 w-4 border-slate-300 text-blue-600 focus:ring-blue-500/50"
              />
              {one.text}
            </label>
          ))}
        </div>
      )}

      {error ? (
        <p id={errorId} className="mt-1 text-sm text-red-700">
          {error}
        </p>
      ) : (
        hint && <p className="mt-1 text-xs text-slate-500">{hint}</p>
      )}
    </div>
  )
}
