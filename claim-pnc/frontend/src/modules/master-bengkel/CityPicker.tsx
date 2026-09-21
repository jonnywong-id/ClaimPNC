import { useId, useState } from 'react'

import type { WorkshopCity } from '@/api/types'
import { Button } from '@/components/Button'
import { Field } from '@/components/Field'

import { MIN_LOOKUP_KEYWORD, useWorkshopCityLookup } from './api'

type Props = {
  /** Kota yang sudah dipilih, atau null bila belum ada. */
  selected: WorkshopCity | null
  onPick: (city: WorkshopCity) => void
  onClear: () => void

  /** Pesan galat dari server yang menempel pada isian ini. */
  error?: string | undefined

  disabled?: boolean
}

/**
 * CityPicker adalah kotak cari–pilih untuk isian "NAMA KOTA".
 *
 * # Kenapa cari–pilih, bukan dropdown biasa
 *
 * Karena masternya besar. Tabel `CITY` berbaris ribuan, dan memuat seluruhnya ke dalam
 * sebuah `<select>` berarti mengirim ribuan baris untuk satu pilihan. Layar lama pun
 * tidak memakai dropdown untuk isian ini — ia memakai `pxAutoComplete` atas
 * `BrowseCity_RD` (`Section/BrowseMasterHEApprove-Section.xml`), dan bentuk itulah yang
 * ditiru di sini.
 *
 * Cabang dan Bank berbeda: keduanya daftar pendek, dan keduanya memakai dropdown biasa.
 *
 * # Kenapa ia TIDAK di components/, dan kenapa ia mirip LookupPicker milik Master Auto Claim
 *
 * Aturan susunan frontend melarang satu fitur mengimpor dari fitur lain; kebutuhan
 * bersama naik ke `components/`. Menaikkannya SEKARANG berarti menyunting
 * `master-auto-claim` yang sudah dinyatakan selesai — dan Isolasi Protektif melarangnya.
 *
 * Syarat menaikkannya kelak sudah jelas dan layak ditulis di sini supaya tidak perlu
 * ditemukan ulang: begitu ada modul KETIGA yang membutuhkan kotak cari–pilih, ketiganya
 * dipindahkan sekaligus ke `components/LookupPicker` dalam satu perubahan — bukan dua
 * modul menunggu satu sama lain.
 *
 * # Yang dijaga di sini
 *
 * ID kota dan namanya SELALU berasal dari sebuah baris yang dipilih, tidak pernah dari
 * yang diketik. Tanpa itu, `CITY_ID` dan `NAMA_KABUPATEN` dapat tersimpan sebagai pasangan
 * yang tidak menunjuk baris mana pun di master kota — dan tidak ada apa pun di layar yang
 * menandakannya.
 */
export function CityPicker({ selected, onPick, onClear, error, disabled = false }: Props) {
  const fieldId = useId()
  const [keyword, setKeyword] = useState('')
  const [isOpen, setOpen] = useState(false)

  const lookup = useWorkshopCityLookup(keyword)
  const rows = lookup.data?.kota ?? []

  const shortKeyword = keyword.trim().length > 0 && keyword.trim().length < MIN_LOOKUP_KEYWORD

  function pick(city: WorkshopCity) {
    onPick(city)
    setKeyword('')
    setOpen(false)
  }

  if (selected) {
    return (
      <div>
        <span className="block text-sm font-medium text-slate-700">Nama kota</span>
        <div className="mt-1.5 flex flex-wrap items-center gap-2 rounded-kontrol border border-slate-300 bg-slate-50 px-3 py-2.5">
          <span className="min-w-0 flex-1 text-sm text-slate-900">
            {selected.nama}
            <span className="ml-2 text-xs text-slate-500">{selected.id}</span>
          </span>
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

  return (
    <div>
      <Field
        id={fieldId}
        label="Nama kota"
        type="search"
        value={keyword}
        disabled={disabled}
        autoComplete="off"
        placeholder="Cari kota…"
        error={error}
        hint={`Ketik minimal ${MIN_LOOKUP_KEYWORD} huruf, lalu pilih dari daftar.`}
        onChange={(e) => {
          setKeyword(e.target.value)
          setOpen(true)
        }}
        onFocus={() => setOpen(true)}
      />

      {isOpen && !shortKeyword && keyword.trim() !== '' && (
        <div className="mt-2 max-h-56 overflow-y-auto rounded-kontrol border border-slate-200 bg-white shadow-lembut">
          {lookup.isError ? (
            <p className="px-3 py-3 text-sm text-red-700" role="alert">
              Pencarian kota gagal. Coba beberapa saat lagi.
            </p>
          ) : lookup.isFetching ? (
            <p className="px-3 py-3 text-sm text-slate-500">Mencari…</p>
          ) : rows.length === 0 ? (
            <p className="px-3 py-3 text-sm text-slate-500">
              Tidak ada kota yang cocok dengan “{keyword.trim()}”.
            </p>
          ) : (
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
          )}
        </div>
      )}
    </div>
  )
}
