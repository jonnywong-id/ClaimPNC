import type { WorkshopCity } from '@/api/types'
import { CityLookupPicker } from '@/components/shared/CityLookupPicker'

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
 * # Tampilan dan perilakunya ada di components/shared/CityLookupPicker
 *
 * Kotak cari–pilihnya kini dipakai bersama dengan `master-supplier/CityPicker` lewat
 * `components/shared/CityLookupPicker`, karena keduanya identik selain label dan isi
 * tampilan pilihan. Berkas ini hanya menentukan label, hook lookup, dan isi tampilannya.
 * `master-auto-claim/LookupPicker` tidak ikut dipindahkan.
 *
 * # Yang dijaga di sini
 *
 * ID kota dan namanya SELALU berasal dari sebuah baris yang dipilih, tidak pernah dari
 * yang diketik. Tanpa itu, `CITY_ID` dan `NAMA_KABUPATEN` dapat tersimpan sebagai pasangan
 * yang tidak menunjuk baris mana pun di master kota — dan tidak ada apa pun di layar yang
 * menandakannya.
 */
export function CityPicker({ selected, onPick, onClear, error, disabled = false }: Readonly<Props>) {
  return (
    <CityLookupPicker
      label="Nama kota"
      minKeyword={MIN_LOOKUP_KEYWORD}
      useLookup={useWorkshopCityLookup}
      selectedContent={
        selected ? (
          <>
            {selected.nama}
            <span className="ml-2 text-xs text-slate-500">{selected.id}</span>
          </>
        ) : null
      }
      onPick={onPick}
      onClear={onClear}
      error={error}
      disabled={disabled}
    />
  )
}
