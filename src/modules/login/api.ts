import { useMutation } from '@tanstack/react-query'

import { callAPI } from '@/api/client'
import type { LoginResponse, ExtendResponse, MeResponse } from '@/api/types'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

export type LoginFields = {
  namaPengguna: string
  kataSandi: string
}

/**
 * Hook masuk.
 *
 * Kata sandi hanya hidup selama satu panggilan: ia tidak disimpan di state global,
 * tidak ditulis ke sessionStorage, dan tidak pernah ikut di objek hasil.
 */
export function useLogin() {
  const saveSession = useSession((state) => state.login)

  return useMutation({
    mutationFn: (values: LoginFields) =>
      callAPI<LoginResponse>('/api/masuk', {
        metode: 'POST',
        body: { nama_pengguna: values.namaPengguna, kata_sandi: values.kataSandi },
      }),
    onSuccess: (result) => {
      saveSession({
        token: result.token,
        user: result.pengguna,
        validUntil: result.berlaku_sampai,
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
export function useLogout() {
  const token = useSession((state) => state.token)
  const clear = useSession((state) => state.clear)
  const clearPortal = useSelectedPortal((state) => state.clear)

  return useMutation({
    mutationFn: () => callAPI<void>('/api/keluar', { metode: 'POST', token }),
    // Sesi dibersihkan di peramban apa pun hasilnya: bila server tidak dapat dihubungi,
    // menahan pengguna tetap "masuk" di layar justru menyesatkan.
    onSettled: () => {
      clear()
      // Pilihan portal ikut dibersihkan: pengguna berikutnya di peramban yang sama
      // tidak boleh mewarisi entitas yang dipilih pengguna sebelumnya.
      clearPortal()
    },
  })
}

/** Hook perpanjang sesi, dipakai peringatan sebelum sesi habis. */
export function useExtendSession() {
  const token = useSession((state) => state.token)
  const refreshValidity = useSession((state) => state.refreshValidity)

  return useMutation({
    mutationFn: () =>
      callAPI<ExtendResponse>('/api/sesi/perpanjang', { metode: 'POST', token }),
    onSuccess: (result) => refreshValidity(result.berlaku_sampai),
  })
}

/** ambilSaya memuat ulang identitas pemanggil dari server. */
export function fetchMe(token: string): Promise<MeResponse> {
  return callAPI<MeResponse>('/api/saya', { token })
}
