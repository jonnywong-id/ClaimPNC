import type { ClauseBusiness } from '@/api/types'
import { BusinessListPicker } from '@/components/shared/BusinessListPicker'

import { MIN_LOOKUP_KEYWORD, useClauseBusinessLookup } from './api'

type Props = {
  /** Lini bisnis yang sudah menempel pada pasal ini. */
  value: ClauseBusiness[]
  onChange: (value: ClauseBusiness[]) => void
  disabled?: boolean
}

/**
 * BusinessPicker adalah isian "Bisnis" — daftar lini bisnis tempat sebuah pasal berlaku.
 *
 * # Bentuknya mengikuti layar lama, bukan diringkas menjadi satu dropdown
 *
 * `Section/BrowsePasalDeatailMaster-Section.xml` memakai **repeating grid** atas
 * `TempPasalCol.BISNISID` dengan ikon tambah/hapus baris (`pzPegaDefaultGridIcons`), dan
 * setiap barisnya adalah **autocomplete** ke `POOLDATA.BUSINESS` lewat
 * `Report Definition/BrowseBusiness_RD-RD.xml`. Satu pasal karena itu dapat menyebut
 * banyak lini bisnis, dan urutannya ditentukan pengguna.
 *
 * Dropdown bertanda centang akan lebih ringkas, tetapi ia mengubah cara kerja pengguna —
 * dan `D-13` menetapkan alur dan tata letak ditiru supaya petugas tidak perlu belajar
 * ulang.
 *
 * # Nama yang diketik bebas DITERIMA, dan itu keputusan yang disengaja
 *
 * Autocomplete layar lama ber-`pyAllowFreeFormInput=true`: nama yang tidak ada di master
 * tetap tersimpan, dengan kode kosong. Modul Master Auto Claim memilih sebaliknya — di
 * sana nilai yang tersimpan SELALU berasal dari baris yang dipilih — tetapi keputusan Work
 * Owner 2026-09-19 untuk modul ini adalah "jalankan as is".
 *
 * Yang ditambahkan di sini hanyalah KEJELASANNYA: butir tanpa kode ditandai terang-terangan
 * di layar, sehingga pengguna tahu butir itu tidak terhubung ke master mana pun. Di Pega,
 * butir semacam itu tampak sama persis dengan yang dipilih dari daftar.
 *
 * # Tampilan dan perilakunya ada di components/shared/BusinessListPicker
 *
 * Markupnya dipakai bersama dengan picker Detail Penyebab Kerugian lewat
 * `components/shared/BusinessListPicker`, karena keduanya identik selain hook lookup dan
 * teks daftar kosong. Berkas ini hanya menentukan master yang dijawab dan teksnya.
 */
export function BusinessPicker({ value, onChange, disabled = false }: Readonly<Props>) {
  return (
    <BusinessListPicker
      value={value}
      onChange={onChange}
      disabled={disabled}
      minKeyword={MIN_LOOKUP_KEYWORD}
      useLookup={useClauseBusinessLookup}
      emptyText="Belum ada lini bisnis. Pasal tetap dapat disimpan tanpa ini."
    />
  )
}
