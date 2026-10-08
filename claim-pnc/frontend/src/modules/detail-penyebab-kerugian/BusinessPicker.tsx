import type { CauseOfLossBusiness } from '@/api/types'
import { BusinessListPicker } from '@/components/shared/BusinessListPicker'

import { useCauseOfLossBusinessSearch } from './api'

/** Sama dengan `detailpenyebab.MinLookupKeyword` di backend. */
const MIN_KEYWORD = 2

type Props = {
  /** Lini bisnis yang sudah menempel pada detail ini. */
  value: CauseOfLossBusiness[]
  onChange: (value: CauseOfLossBusiness[]) => void
  disabled?: boolean
}

/**
 * BusinessPicker adalah isian "Bisnis" — daftar lini bisnis tempat sebuah detail penyebab
 * kerugian berlaku.
 *
 * # Bentuknya mengikuti layar lama, bukan diringkas menjadi satu dropdown
 *
 * `Section/BrowseDetailCauseOfLoss-Section.xml:3251` memakai **repeating grid** atas
 * `TempDcol.BISNISID`, dan setiap barisnya adalah **autocomplete** ke `POOLDATA.BUSINESS`
 * lewat `Report Definition/BrowseBusiness_RD-RD.xml` (`:3819`). Satu detail karena itu
 * dapat menyebut banyak lini bisnis, dan urutannya ditentukan pengguna.
 *
 * Dropdown bertanda centang akan lebih ringkas, tetapi ia mengubah cara kerja pengguna —
 * dan `D-13` menetapkan alur dan tata letak ditiru supaya petugas tidak perlu belajar
 * ulang.
 *
 * # Nama yang diketik bebas DITERIMA
 *
 * Autocomplete-nya ber-`pyAllowFreeFormInput=true`
 * (`Section/BrowseDetailCauseOfLoss-Section.xml:3773`): nama yang tidak ada di master tetap
 * tersimpan, dengan kode kosong. Perilakunya ditiru apa adanya (`P-5`), sama seperti Master
 * Pasal Kerugian yang menghadapi isian berbentuk sama.
 *
 * Yang ditambahkan di sini hanyalah KEJELASANNYA: butir tanpa kode ditandai terang-terangan
 * di layar, sehingga pengguna tahu butir itu tidak terhubung ke master mana pun. Di Pega,
 * butir semacam itu tampak sama persis dengan yang dipilih dari daftar.
 *
 * # Tampilan dan perilakunya ada di components/shared/BusinessListPicker
 *
 * Keduanya — picker ini dan milik Master Pasal Kerugian — identik selain hook lookup dan
 * teks daftar kosong, sehingga markupnya dipakai bersama lewat
 * `components/shared/BusinessListPicker`. Berkas ini hanya menentukan master yang dijawab
 * dan teksnya; bila kelak salah satu harus berubah sendiri, ia cukup berhenti memakai
 * komponen bersama itu.
 */
export function BusinessPicker({ value, onChange, disabled = false }: Readonly<Props>) {
  return (
    <BusinessListPicker
      value={value}
      onChange={onChange}
      disabled={disabled}
      minKeyword={MIN_KEYWORD}
      useLookup={useCauseOfLossBusinessSearch}
      emptyText="Belum ada lini bisnis. Detail tetap dapat disimpan tanpa ini."
    />
  )
}
