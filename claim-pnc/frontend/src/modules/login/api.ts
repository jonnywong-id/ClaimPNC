import { useMutation } from '@tanstack/react-query'

import { callAPI } from '@/api/client'
import type { LoginResponse, ExtendResponse, MeResponse } from '@/api/types'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

export type LoginFormValues = {
  username: string
  password: string
}

/**
 * Hook masuk.
 *
 * Kata sandi hanya hidup selama satu panggilan: ia tidak disimpan di state global,
 * tidak ditulis ke sessionStorage, dan tidak pernah ikut di objek hasil.
 */
export function useLogin() {
  const saveSession = useSession((state) => state.signIn)

  return useMutation({
    mutationFn: (values: LoginFormValues) =>
      callAPI<LoginResponse>('/api/masuk', {
        method: 'POST',
        body: { nama_pengguna: values.username, kata_sandi: values.password },
      }),
    onSuccess: (result) => {
      saveSession({
        token: result.token,
        pengguna: result.pengguna,
        expiresAt: result.berlaku_sampai,
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
  const cleanup = useSession((state) => state.cleanup)
  const clearPortal = useSelectedPortal((state) => state.cleanup)

  return useMutation({
    mutationFn: () => callAPI<void>('/api/keluar', { method: 'POST', token }),
    // Sesi dibersihkan di peramban apa pun hasilnya: bila server tidak dapat dihubungi,
    // menahan pengguna tetap "masuk" di layar justru menyesatkan.
    onSettled: () => {
      cleanup()
      // Pilihan portal ikut dibersihkan: pengguna berikutnya di peramban yang sama
      // tidak boleh mewarisi entitas yang dipilih pengguna sebelumnya.
      clearPortal()
    },
  })
}

/** Hook perpanjang sesi, dipakai peringatan sebelum sesi habis. */
export function useExtendSession() {
  const token = useSession((state) => state.token)
  const refreshExpiry = useSession((state) => state.refreshExpiry)

  return useMutation({
    mutationFn: () =>
      callAPI<ExtendResponse>('/api/sesi/perpanjang', { method: 'POST', token }),
    onSuccess: (result) => refreshExpiry(result.berlaku_sampai),
  })
}

/** fetchMe memuat ulang identitas pemanggil dari server. */
export function fetchMe(token: string): Promise<MeResponse> {
  return callAPI<MeResponse>('/api/saya', { token })
}
