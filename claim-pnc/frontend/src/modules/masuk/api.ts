import { useMutation } from '@tanstack/react-query'

import { panggilAPI } from '@/api/klien'
import type { ResponsMasuk, ResponsPerpanjang, ResponsSaya } from '@/api/tipe'
import { gunakanPortalTerpilih } from '@/app/portal'
import { gunakanSesi } from '@/app/sesi'

export type IsianMasuk = {
  namaPengguna: string
  kataSandi: string
}

/**
 * Hook masuk.
 *
 * Kata sandi hanya hidup selama satu panggilan: ia tidak disimpan di state global,
 * tidak ditulis ke sessionStorage, dan tidak pernah ikut di objek hasil.
 */
export function gunakanMasuk() {
  const simpanSesi = gunakanSesi((keadaan) => keadaan.masuk)

  return useMutation({
    mutationFn: (isian: IsianMasuk) =>
      panggilAPI<ResponsMasuk>('/api/masuk', {
        metode: 'POST',
        badan: { nama_pengguna: isian.namaPengguna, kata_sandi: isian.kataSandi },
      }),
    onSuccess: (hasil) => {
      simpanSesi({
        token: hasil.token,
        pengguna: hasil.pengguna,
        berlakuSampai: hasil.berlaku_sampai,
      })
    },
  })
}

/**
 * Hook keluar.
 *
 * Sesi dicabut di server lebih dulu; membersihkan peramban saja tidak cukup karena
 * token yang sudah terlanjur disalin orang lain akan tetap sah.
 */
export function gunakanKeluar() {
  const token = gunakanSesi((keadaan) => keadaan.token)
  const bersihkan = gunakanSesi((keadaan) => keadaan.bersihkan)
  const bersihkanPortal = gunakanPortalTerpilih((keadaan) => keadaan.bersihkan)

  return useMutation({
    mutationFn: () => panggilAPI<void>('/api/keluar', { metode: 'POST', token }),
    // Sesi dibersihkan di peramban apa pun hasilnya: bila server tidak dapat dihubungi,
    // menahan pengguna tetap "masuk" di layar justru menyesatkan.
    onSettled: () => {
      bersihkan()
      // Pilihan portal ikut dibersihkan: pengguna berikutnya di peramban yang sama
      // tidak boleh mewarisi entitas yang dipilih pengguna sebelumnya.
      bersihkanPortal()
    },
  })
}

/** Hook perpanjang sesi, dipakai peringatan sebelum sesi habis. */
export function gunakanPerpanjangSesi() {
  const token = gunakanSesi((keadaan) => keadaan.token)
  const perbaruiBerlaku = gunakanSesi((keadaan) => keadaan.perbaruiBerlaku)

  return useMutation({
    mutationFn: () =>
      panggilAPI<ResponsPerpanjang>('/api/sesi/perpanjang', { metode: 'POST', token }),
    onSuccess: (hasil) => perbaruiBerlaku(hasil.berlaku_sampai),
  })
}

/** ambilSaya memuat ulang identitas pemanggil dari server. */
export function ambilSaya(token: string): Promise<ResponsSaya> {
  return panggilAPI<ResponsSaya>('/api/saya', { token })
}
