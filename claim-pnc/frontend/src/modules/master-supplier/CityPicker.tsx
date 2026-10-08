import type { SupplierCity } from '@/api/types'
import { CityLookupPicker } from '@/components/shared/CityLookupPicker'

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
 * # Tampilan dan perilakunya ada di components/shared/CityLookupPicker
 *
 * Kotak cari–pilihnya kini dipakai bersama dengan `master-bengkel/CityPicker` lewat
 * `components/shared/CityLookupPicker`, karena keduanya identik selain label dan isi
 * tampilan pilihan. Berkas ini hanya menentukan label, hook lookup, dan isi tampilannya.
 * `master-auto-claim/LookupPicker` tidak ikut dipindahkan.
 */
export function CityPicker({ value, onPick, onClear, error, disabled = false }: Readonly<Props>) {
  return (
    <CityLookupPicker
      label="Kota"
      minKeyword={MIN_LOOKUP_KEYWORD}
      useLookup={useSupplierCityLookup}
      selectedContent={value === '' ? null : value}
      onPick={onPick}
      onClear={onClear}
      error={error}
      disabled={disabled}
    />
  )
}
