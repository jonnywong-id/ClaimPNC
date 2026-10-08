import { useId, useState, type ReactNode } from 'react'

import { Button } from '@/components/Button'
import { Field } from '@/components/Field'

/** Satu baris master kota yang dapat dipilih. */
export type LookupCity = {
  id: string
  nama: string
}

/** Bentuk minimal hasil kueri lookup kota yang dibaca komponen ini. */
export type CityLookupResult = {
  data?: { kota: LookupCity[] } | undefined
  isError: boolean
  isFetching: boolean
}

type Props = {
  /** Label isian, mis. "Nama kota" atau "Kota". */
  label: string

  /** Panjang minimum kata kunci sebelum pencarian dijalankan. */
  minKeyword: number

  /** Hook kueri lookup kota milik modul pemanggil. */
  useLookup: (keyword: string) => CityLookupResult

  /**
   * Isi tampilan pilihan yang sudah tersimpan, atau null bila belum ada pilihan.
   * Bila bukan null, kotak cari diganti tampilan pilihan dengan tombol "Ganti".
   */
  selectedContent: ReactNode
  onPick: (city: LookupCity) => void
  onClear: () => void

  /** Pesan galat dari server yang menempel pada isian ini. */
  error?: string | undefined

  disabled?: boolean
}

/**
 * CityLookupPicker adalah kotak cari–pilih atas master kota, dipakai bersama oleh
 * `master-bengkel/CityPicker` dan `master-supplier/CityPicker`.
 *
 * Masternya besar (tabel `CITY` berbaris ribuan), sehingga kota dicari lewat kata kunci,
 * bukan dimuat seluruhnya ke dalam dropdown. Alasan rinci per modul tetap dicatat di
 * berkas CityPicker masing-masing modul.
 *
 * Kota SELALU berasal dari sebuah baris yang dipilih, tidak pernah dari yang diketik.
 */
export function CityLookupPicker({
  label,
  minKeyword,
  useLookup,
  selectedContent,
  onPick,
  onClear,
  error,
  disabled = false,
}: Readonly<Props>) {
  const fieldId = useId()
  const [keyword, setKeyword] = useState('')
  const [isOpen, setOpen] = useState(false)

  const lookup = useLookup(keyword)
  const rows = lookup.data?.kota ?? []

  const shortKeyword = keyword.trim().length > 0 && keyword.trim().length < minKeyword

  function pick(city: LookupCity) {
    onPick(city)
    setKeyword('')
    setOpen(false)
  }

  if (selectedContent !== null) {
    return (
      <div>
        <span className="block text-sm font-medium text-slate-700">{label}</span>
        <div className="mt-1.5 flex flex-wrap items-center gap-2 rounded-kontrol border border-slate-300 bg-slate-50 px-3 py-2.5">
          <span className="min-w-0 flex-1 text-sm text-slate-900">{selectedContent}</span>
          {!disabled && (
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
    )
  }

  // Isi daftar hasil pencarian kota menurut keadaan kueri.
  function renderResults() {
    if (lookup.isError) {
      return (
        <p className="px-3 py-3 text-sm text-red-700" role="alert">
          Pencarian kota gagal. Coba beberapa saat lagi.
        </p>
      )
    }
    if (lookup.isFetching) {
      return <p className="px-3 py-3 text-sm text-slate-500">Mencari…</p>
    }
    if (rows.length === 0) {
      return (
        <p className="px-3 py-3 text-sm text-slate-500">
          Tidak ada kota yang cocok dengan “{keyword.trim()}”.
        </p>
      )
    }
    return (
      <ul>
        {rows.map((city) => (
          <li key={city.id}>
            <button
              type="button"
              onClick={() => pick(city)}
              className={[
                'flex w-full items-baseline gap-2 px-3 py-2 text-left text-sm',
                'transition-colors duration-150 ease-halus',
                'hover:bg-blue-50 focus:bg-blue-50 focus:outline-none',
              ].join(' ')}
            >
              <span className="min-w-0 flex-1 text-slate-900">{city.nama}</span>
              <span className="shrink-0 text-xs text-slate-500">{city.id}</span>
            </button>
          </li>
        ))}
      </ul>
    )
  }

  return (
    <div>
      <Field
        id={fieldId}
        label={label}
        type="search"
        value={keyword}
        disabled={disabled}
        autoComplete="off"
        placeholder="Cari kota…"
        error={error}
        hint={`Ketik minimal ${minKeyword} huruf, lalu pilih dari daftar.`}
        onChange={(e) => {
          setKeyword(e.target.value)
          setOpen(true)
        }}
        onFocus={() => setOpen(true)}
      />

      {isOpen && !shortKeyword && keyword.trim() !== '' && (
        <div className="mt-2 max-h-56 overflow-y-auto rounded-kontrol border border-slate-200 bg-white shadow-lembut">
          {renderResults()}
        </div>
      )}
    </div>
  )
}
