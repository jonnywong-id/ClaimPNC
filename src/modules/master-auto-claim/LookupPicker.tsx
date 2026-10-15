import { useId, useState } from 'react'

import { Button } from '@/components/Button'
import { Field } from '@/components/Field'

import { MIN_LOOKUP_KEYWORD } from './api'

/** Satu baris hasil pencarian. Kedua lookup memakai bentuk yang sama. */
export type LookupRow = { id: string; nama: string }

type Props = {
  /** Judul kotak pencarian, misalnya "Sumber Bisnis". */
  label: string

  /** Yang sudah dipilih, atau null bila belum ada. */
  selected: LookupRow | null

  /** Hasil pencarian yang sedang ditampilkan. */
  rows: LookupRow[]
  isSearching: boolean
  isError: boolean

  /** Kata kunci yang sedang berlaku, dipegang pemanggil supaya hook-nya ikut berubah. */
  keyword: string
  onKeywordChange: (keyword: string) => void

  onPick: (row: LookupRow) => void
  onClear?: (() => void) | undefined

  /** Pesan galat dari server yang menempel pada isian ini. */
  error?: string | undefined

  /** Keterangan singkat di bawah kotak pencarian. */
  hint?: string | undefined

  disabled?: boolean
}

/**
 * LookupPicker adalah kotak cari–pilih untuk Sumber Bisnis dan Client.
 *
 * # Kenapa cari–pilih, bukan dropdown biasa
 *
 * Karena kedua masternya besar. `POOLDATA.AGENT` dan `POOLDATA.CLIENT` bukan daftar
 * pendek seperti master bank: memuat seluruhnya ke dalam sebuah `<select>` berarti
 * mengirim ribuan baris untuk satu pilihan. Layar lama pun tidak memakai dropdown — ia
 * memakai kotak pencarian dengan grid hasil di bawahnya
 * (`Section/BrowseAutoKlaim-Section.xml`), dan bentuk itulah yang ditiru di sini.
 *
 * # Kenapa ia TIDAK di components/
 *
 * Hanya modul ini yang memakainya. Aturan susunan frontend menaikkan kebutuhan bersama
 * ke `components/`, dan yang belum bersama tetap di modulnya — menaikkannya lebih dulu
 * berarti menebak bentuk yang dibutuhkan modul lain sebelum modul itu ada.
 *
 * # Yang dijaga di sini
 *
 * Nilai yang tersimpan SELALU berasal dari sebuah baris yang dipilih, tidak pernah dari
 * yang diketik. Itu padanan langsung dua penolakan sistem lama — "Nama penerima klaim
 * tidak ditemukan. Jangan diketik manual." dan "Nama bank jangan diketik manual." —
 * dan di sini ia ditegakkan oleh BENTUK kontrolnya, bukan oleh pemeriksaan sesudahnya.
 */
export function LookupPicker({
  label,
  selected,
  rows,
  isSearching,
  isError,
  keyword,
  onKeywordChange,
  onPick,
  onClear,
  error,
  hint,
  disabled = false,
}: Props) {
  const fieldId = useId()
  const [isOpen, setOpen] = useState(false)

  const shortKeyword = keyword.trim().length > 0 && keyword.trim().length < MIN_LOOKUP_KEYWORD

  function pick(row: LookupRow) {
    onPick(row)
    onKeywordChange('')
    setOpen(false)
  }

  return (
    <div>
      {selected ? (
        <div>
          <span className="block text-sm font-medium text-slate-700">{label}</span>
          <div className="mt-1.5 flex flex-wrap items-center gap-2 rounded-kontrol border border-slate-300 bg-slate-50 px-3 py-2.5">
            <span className="min-w-0 flex-1 text-sm text-slate-900">
              {selected.nama}
              <span className="ml-2 text-xs text-slate-500">{selected.id}</span>
            </span>
            {onClear && !disabled && (
              <Button tone="halus" onClick={onClear}>
                Ganti
              </Button>
            )}
          </div>
          {/*
            Galat tetap ditampilkan meski sudah ada yang dipilih: server dapat menolak
            pilihan yang sudah tidak ada lagi di masternya.
          */}
          {error && (
            <p className="mt-1.5 text-sm text-red-700" role="alert">
              {error}
            </p>
          )}
        </div>
      ) : (
        <div>
          <Field
            id={fieldId}
            label={label}
            type="search"
            value={keyword}
            disabled={disabled}
            autoComplete="off"
            placeholder={`Cari ${label.toLowerCase()}…`}
            error={error}
            hint={hint ?? `Ketik minimal ${MIN_LOOKUP_KEYWORD} huruf, lalu pilih dari daftar.`}
            onChange={(e) => {
              onKeywordChange(e.target.value)
              setOpen(true)
            }}
            onFocus={() => setOpen(true)}
          />

          {isOpen && !shortKeyword && keyword.trim() !== '' && (
            <div className="mt-2 max-h-56 overflow-y-auto rounded-kontrol border border-slate-200 bg-white shadow-lembut">
              {isError ? (
                <p className="px-3 py-3 text-sm text-red-700" role="alert">
                  Pencarian gagal. Coba beberapa saat lagi.
                </p>
              ) : isSearching ? (
                <p className="px-3 py-3 text-sm text-slate-500">Mencari…</p>
              ) : rows.length === 0 ? (
                <p className="px-3 py-3 text-sm text-slate-500">
                  Tidak ada yang cocok dengan “{keyword.trim()}”.
                </p>
              ) : (
                <ul>
                  {rows.map((row) => (
                    <li key={row.id}>
                      <button
                        type="button"
                        onClick={() => pick(row)}
                        className={[
                          'flex w-full items-baseline gap-2 px-3 py-2 text-left text-sm',
                          'transition-colors duration-150 ease-halus',
                          'hover:bg-blue-50 focus:bg-blue-50 focus:outline-none',
                        ].join(' ')}
                      >
                        <span className="min-w-0 flex-1 text-slate-900">{row.nama}</span>
                        <span className="shrink-0 text-xs text-slate-500">{row.id}</span>
                      </button>
                    </li>
                  ))}
                </ul>
              )}
            </div>
          )}
        </div>
      )}
    </div>
  )
}
