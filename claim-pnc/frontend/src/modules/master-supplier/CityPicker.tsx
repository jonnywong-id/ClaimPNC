import { useId, useState } from 'react'

import type { SupplierCity } from '@/api/types'
import { Button } from '@/components/Button'
import { Field } from '@/components/Field'

import { MIN_LOOKUP_KEYWORD, useSupplierCityLookup } from './api'

type Props = {
  /** Nama kota yang sudah tersimpan, atau kosong bila belum ada. */
  value: string
  onPick: (city: SupplierCity) => void
  onClear: () => void

  /** Pesan galat dari server yang menempel pada isian ini. */
  error?: string | undefined

  disabled?: boolean
}

/**
 * CityPicker adalah kotak cari–pilih untuk isian "Kota".
 *
 * # Kenapa cari–pilih, padahal layar lama memakai dropdown
 *
 * Karena masternya besar. `Section/CreateMasterSupplier_Sec-Section.xml` memang memasang
 * `pxDropdown` atas `BrowseCity_RD`, dan report definition itu **tidak membatasi hasilnya
 * sama sekali** (`pyMaxRecords=0`) — seluruh tabel `CITY` yang berbaris ribuan dimuat ke
 * klipboard, lalu disaring di peramban.
 *
 * Menirunya berarti mengirim ribuan baris untuk satu pilihan, pada setiap kali form
 * dibuka. Yang ditiru karena itu adalah HASILNYA — petugas memilih satu kota dari master
 * kota — bukan cara memuatnya.
 *
 * Cabang, Negara, dan Bank berbeda: ketiganya daftar pendek, dan ketiganya memakai
 * dropdown biasa persis seperti layar lama.
 *
 * # Yang disimpan hanyalah NAMANYA
 *
 * Dokumen supplier tidak punya kunci `KOTA_ID` — `RDB List/GetDataEditMasterSupller-SQL.xml`
 * membaca `KOTA` saja. Kode kota tetap ditampilkan di daftar pilihan sebagai pembeda bila
 * ada dua kota bernama mirip, tetapi ia tidak ikut tersimpan.
 *
 * Akibatnya harus disadari: bila sebuah kota berganti nama di master, baris supplier yang
 * menyebut nama lamanya TIDAK ikut berubah. Itu keadaan sistem lama, dan membetulkannya
 * menuntut kolom yang tabelnya tidak punya. Yang dikerjakan di sini: nama yang tersimpan
 * selalu ditampilkan apa adanya, sehingga menyunting baris lama tidak diam-diam
 * mengosongkan kotanya.
 *
 * # Kenapa ia TIDAK di components/, dan syarat menaikkannya sudah TERPENUHI
 *
 * Aturan susunan frontend melarang satu fitur mengimpor dari fitur lain; kebutuhan bersama
 * naik ke `components/`. `master-bengkel/CityPicker.tsx` mencatat syaratnya: begitu ada
 * modul KETIGA yang membutuhkan kotak cari–pilih, ketiganya dipindahkan sekaligus ke
 * `components/LookupPicker`.
 *
 * **Modul ketiga itu adalah modul ini** — setelah `master-auto-claim/LookupPicker` dan
 * `master-bengkel/CityPicker`. Syaratnya terpenuhi.
 *
 * Yang menahannya bukan lagi syarat itu, melainkan Isolasi Protektif: menaikkannya
 * menuntut menyunting kedua modul master yang sudah dinyatakan selesai. Perubahan itu
 * layak dijadwalkan Work Owner sebagai satu pekerjaan tersendiri, bukan diselipkan ke
 * dalam penambahan modul ini.
 */
export function CityPicker({ value, onPick, onClear, error, disabled = false }: Props) {
  const fieldId = useId()
  const [keyword, setKeyword] = useState('')
  const [isOpen, setOpen] = useState(false)

  const lookup = useSupplierCityLookup(keyword)
  const rows = lookup.data?.kota ?? []

  const shortKeyword = keyword.trim().length > 0 && keyword.trim().length < MIN_LOOKUP_KEYWORD

  function pick(city: SupplierCity) {
    onPick(city)
    setKeyword('')
    setOpen(false)
  }

  if (value !== '') {
    return (
      <div>
        <span className="block text-sm font-medium text-slate-700">Kota</span>
        <div className="mt-1.5 flex flex-wrap items-center gap-2 rounded-kontrol border border-slate-300 bg-slate-50 px-3 py-2.5">
          <span className="min-w-0 flex-1 text-sm text-slate-900">{value}</span>
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
        label="Kota"
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
