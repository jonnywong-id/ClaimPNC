import { useId } from 'react'

/**
 * ChoiceField adalah isian yang nilainya dipilih dari daftar, dengan jalan keluar berupa
 * ketikan bebas.
 *
 * # Kenapa komponen ini ada, dan kenapa ia tinggal di modul ini
 *
 * Empat isian Master Sparepart — Jenis, Satuan, Status Aktif, dan Status Sparepart —
 * dirender `pxRadioButtons` atau `pxDropdown` di layar Pega, tetapi **daftar pilihannya
 * tidak ada di export**: keempatnya memakai `pyListSource=associated`, artinya pilihannya
 * datang dari rule Field Value pada propertinya, dan tidak ada satu pun direktori Property
 * maupun Field Value di export (`R-16`).
 *
 * Work Owner meminta layarnya "seperti aplikasi Pega" (2026-09-20). Itu menuntut dua hal
 * yang tampak bertentangan: bentuknya harus dropdown atau radio, sementara isinya tidak
 * diketahui. Komponen ini yang menyelesaikannya:
 *
 *   - bentuknya dropdown atau radio, persis seperti Pega;
 *   - pilihannya adalah nilai yang SUDAH DIPAKAI baris lain pada entitas ini — bukan
 *     daftar yang dikarang;
 *   - selalu ada pilihan **Lainnya**, yang membuka isian ketik.
 *
 * Pilihan terakhir itu bukan hiasan. Tanpa ia, basis data yang masih kosong akan
 * menghasilkan dropdown tanpa satu pun pilihan — isian yang mustahil diisi — dan baris
 * lama yang nilainya belum pernah dipakai baris lain tidak akan dapat disimpan ulang.
 *
 * Ia tinggal di `modules/master-sparepart/` dan BUKAN di `shared/components/` karena
 * persoalannya khas modul ini: ia jawaban atas daftar pilihan yang hilang, bukan pola
 * antarmuka yang layak dipakai ulang. Memindahkannya ke `shared/` akan mengundang layar
 * lain memakainya di tempat yang daftar pilihannya sebenarnya diketahui.
 */
type Props = {
  /** Label yang dibaca pengguna, mengikuti caption layar Pega. */
  label: string

  /** Nilai yang sedang dipegang isian ini. */
  value: string

  /** Nilai yang sudah dipakai baris lain; menjadi pilihan yang ditawarkan. */
  known: string[]

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
  maxLength: number
  onChange: (value: string) => void
}

/** Sandi pilihan "Lainnya"; sengaja memuat karakter yang tidak mungkin menjadi nilai. */
const OTHER = '\u0000lainnya'

export function ChoiceField({
  label,
  value,
  known,
  variant,
  disabled,
  error,
  hint,
  maxLength,
  onChange,
}: Props) {
  const id = useId()
  const errorId = `${id}-error`

  // Nilai yang sedang dipegang TETAP muncul sebagai pilihan meski tidak ada di daftar —
  // baris lama dapat memuat nilai yang belum pernah dipakai baris lain, dan menyembunyikannya
  // akan mengosongkan isian hanya karena barisnya dibuka.
  const options = [...new Set([...known, ...(value !== '' ? [value] : [])])].sort()
  const isTyping = value !== '' && !known.includes(value)

  function pick(picked: string) {
    // Memilih "Lainnya" mengosongkan isian lebih dulu, supaya kotak ketiknya terbuka kosong
    // alih-alih mewarisi nilai yang barusan ditinggalkan.
    onChange(picked === OTHER ? '' : picked)
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
          value={isTyping ? OTHER : value}
          disabled={disabled}
          aria-invalid={error ? 'true' : 'false'}
          aria-describedby={describedBy}
          onChange={(event) => pick(event.target.value)}
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
            <option key={one} value={one}>
              {one}
            </option>
          ))}
          <option value={OTHER}>Lainnya…</option>
        </select>
      ) : (
        <div
          role="radiogroup"
          aria-label={label}
          aria-describedby={describedBy}
          className="mt-1.5 flex flex-wrap items-center gap-x-4 gap-y-2"
        >
          {options.map((one) => (
            <label key={one} className="inline-flex items-center gap-1.5 text-sm text-slate-800">
              <input
                type="radio"
                name={id}
                value={one}
                checked={value === one}
                disabled={disabled}
                onChange={() => pick(one)}
                className="h-4 w-4 border-slate-300 text-blue-600 focus:ring-blue-500/50"
              />
              {one}
            </label>
          ))}
          <label className="inline-flex items-center gap-1.5 text-sm text-slate-800">
            <input
              type="radio"
              name={id}
              value={OTHER}
              checked={isTyping || value === ''}
              disabled={disabled}
              onChange={() => pick(OTHER)}
              className="h-4 w-4 border-slate-300 text-blue-600 focus:ring-blue-500/50"
            />
            Lainnya…
          </label>
        </div>
      )}

      {/*
        Kotak ketik hanya muncul saat "Lainnya" yang dipilih, atau saat nilai yang tersimpan
        memang belum pernah dipakai baris lain. Menampilkannya selalu akan membuat isian
        punya dua tempat memasukkan nilai yang sama, dan itu membingungkan.
      */}
      {(isTyping || (variant === 'radio' && value === '')) && (
        <input
          type="text"
          value={value}
          maxLength={maxLength}
          disabled={disabled}
          aria-label={`${label} — ketik nilai lain`}
          aria-invalid={error ? 'true' : 'false'}
          aria-describedby={describedBy}
          onChange={(event) => onChange(event.target.value)}
          className={[
            'mt-2 w-full rounded-kontrol border bg-white px-3 py-2.5 text-sm text-slate-900',
            'placeholder:text-slate-400',
            'transition-[border-color,box-shadow] duration-150 ease-halus',
            'focus:outline-none focus:ring-4',
            'disabled:cursor-not-allowed disabled:bg-slate-50 disabled:text-slate-500',
            error
              ? 'border-red-400 focus:border-red-500 focus:ring-red-500/15'
              : 'border-slate-300 hover:border-slate-400 focus:border-blue-500 focus:ring-blue-500/15',
          ].join(' ')}
          placeholder="Ketik nilai lain"
        />
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
