import type { ReactNode } from 'react'
import type { UseFormRegisterReturn } from 'react-hook-form'
import { z } from 'zod'

import { Button } from '@/components/Button'
import { ComboField } from '@/components/ComboField'

/**
 * Skema satu senarai bisnis berbatas panjang nama.
 *
 * Disimpan sebagai senarai objek, bukan senarai teks, karena useFieldArray menuntut setiap
 * barisnya berupa objek agar dapat memberinya kunci yang stabil.
 */
export function businessListSchema(maxNameLength: number) {
  return z.array(
    z.object({
      nama: z
        .string()
        .trim()
        .max(maxNameLength, `Nama bisnis paling panjang ${maxNameLength} karakter.`),
    }),
  )
}

/** Mengubah pemetaan bisnis baris yang disunting menjadi nilai awal grid. */
export function businessRowsOf(rows: readonly { nama: string }[] | undefined): { nama: string }[] {
  return (rows ?? []).map((b) => ({ nama: b.nama }))
}

type Props = {
  /** Judul grid, mis. "ID Bisnis" atau "Bisnis". */
  legend: string
  /** Teks saat belum ada baris; menyebut jenis barang yang tetap dapat disimpan. */
  emptyText: string
  /** Pemetaan bisnis baris yang disunting masih dimuat. */
  isLoading: boolean
  /** Baris dari useFieldArray; `id`-nya kunci yang stabil. */
  rows: readonly { id: string }[]
  suggestions: string[]
  maxLength: number
  rowError: (index: number) => string | undefined
  rowRegistration: (index: number) => UseFormRegisterReturn
  /** Pelanggaran atas grid secara keseluruhan (server menyebutnya isian "bisnis"). */
  gridError: string | undefined
  onAdd: () => void
  onRemove: (index: number) => void
}

/**
 * BusinessRowsField adalah GRID bisnis, bukan satu isian: satu baris master dapat dipakai
 * banyak bisnis. Di Pega ia repeat grid berkelas ASM-FW-GISFW-Int-BUSINESS.
 *
 * Sel Bisnis memakai ComboField, bukan SelectField: kontrol Pega-nya menerima ketikan
 * bebas, sehingga nama di luar daftar TETAP boleh diketik dan disimpan.
 */
export function BusinessRowsField({
  legend,
  emptyText,
  isLoading,
  rows,
  suggestions,
  maxLength,
  rowError,
  rowRegistration,
  gridError,
  onAdd,
  onRemove,
}: Readonly<Props>) {
  function renderRows(): ReactNode {
    if (isLoading) {
      return <p className="text-sm text-slate-500">Memuat bisnis yang sudah dipilih…</p>
    }
    if (rows.length === 0) {
      return <p className="text-sm text-slate-500">{emptyText}</p>
    }
    return (
      <ul className="space-y-2">
        {rows.map((row, index) => (
          <li key={row.id} className="flex items-end gap-2">
            <div className="grow">
              <ComboField
                id={`bisnis-${index}`}
                label={`Bisnis baris ${index + 1}`}
                options={suggestions}
                maxLength={maxLength}
                error={rowError(index)}
                {...rowRegistration(index)}
              />
            </div>
            <Button
              tone="halus"
              onClick={() => onRemove(index)}
              aria-label={`Hapus bisnis baris ${index + 1}`}
            >
              Hapus
            </Button>
          </li>
        ))}
      </ul>
    )
  }

  return (
    <fieldset className="rounded-kontrol border border-slate-200 p-4">
      <legend className="px-1 text-sm font-medium text-slate-700">{legend}</legend>

      {renderRows()}

      {/* Pelanggaran grid dilaporkan di bawah gridnya, bukan di salah satu barisnya:
          server menyebutnya sebagai satu isian bernama "bisnis". */}
      {gridError && (
        <p className="mt-2 text-sm text-red-600" role="alert">
          {gridError}
        </p>
      )}

      <div className="mt-3">
        <Button tone="kedua" onClick={onAdd}>
          Tambah Bisnis
        </Button>
        {/* Daftar saran yang gagal dimuat TIDAK menghalangi apa pun: namanya memang boleh
            diketik sendiri. Yang hilang hanya kenyamanan memilih. */}
        {suggestions.length === 0 && (
          <p className="mt-2 text-sm text-slate-500">
            Daftar bisnis belum dapat dimuat, jadi tidak ada saran. Nama bisnis tetap dapat
            diketik sendiri.
          </p>
        )}
      </div>
    </fieldset>
  )
}
