import { create } from 'zustand'

import type { Pengguna } from '@/api/tipe'

const KUNCI_SIMPANAN = 'claim-pnc.sesi'

export type KeadaanSesi = {
  token: string | null
  pengguna: Pengguna | null
  berlakuSampai: string | null
  masuk: (nilai: { token: string; pengguna: Pengguna; berlakuSampai: string }) => void
  perbaruiBerlaku: (berlakuSampai: string) => void
  bersihkan: () => void
}

/**
 * Simpanan sesi global.
 *
 * Yang disimpan hanyalah token, profil, dan batas berlakunya. Kata sandi TIDAK PERNAH
 * disimpan di peramban dalam bentuk apa pun.
 *
 * CATATAN KEAMANAN YANG DISADARI. Token dikirim sebagai Bearer di header Authorization
 * (keputusan Work Owner, 2026-09-15), sehingga ia harus dapat dibaca JavaScript dan
 * karena itu tidak dapat memakai cookie HttpOnly yang dianjurkan
 * docs/Steering/11-SECURITY.md §2.2. Konsekuensinya: satu kerentanan XSS berarti
 * pencurian sesi. sessionStorage dipilih, bukan localStorage, supaya token hilang saat
 * tab ditutup dan tidak dibagi antar tab.
 */
export const gunakanSesi = create<KeadaanSesi>((set) => ({
  ...muatDariPeramban(),

  masuk: ({ token, pengguna, berlakuSampai }) => {
    simpanKePeramban({ token, pengguna, berlakuSampai })
    set({ token, pengguna, berlakuSampai })
  },

  perbaruiBerlaku: (berlakuSampai) => {
    set((sebelumnya) => {
      if (sebelumnya.token && sebelumnya.pengguna) {
        simpanKePeramban({
          token: sebelumnya.token,
          pengguna: sebelumnya.pengguna,
          berlakuSampai,
        })
      }
      return { berlakuSampai }
    })
  },

  bersihkan: () => {
    hapusDariPeramban()
    set({ token: null, pengguna: null, berlakuSampai: null })
  },
}))

type SesiTersimpan = { token: string; pengguna: Pengguna; berlakuSampai: string }

function muatDariPeramban(): Pick<KeadaanSesi, 'token' | 'pengguna' | 'berlakuSampai'> {
  const kosong = { token: null, pengguna: null, berlakuSampai: null }
  try {
    const mentah = window.sessionStorage.getItem(KUNCI_SIMPANAN)
    if (!mentah) return kosong
    const isi = JSON.parse(mentah) as Partial<SesiTersimpan>
    if (!isi.token || !isi.pengguna || !isi.berlakuSampai) return kosong
    return { token: isi.token, pengguna: isi.pengguna, berlakuSampai: isi.berlakuSampai }
  } catch {
    // sessionStorage dapat ditolak peramban (mode privat, kebijakan perusahaan).
    // Aplikasi tetap harus jalan; pengguna cukup masuk ulang setelah muat ulang.
    return kosong
  }
}

function simpanKePeramban(isi: SesiTersimpan): void {
  try {
    window.sessionStorage.setItem(KUNCI_SIMPANAN, JSON.stringify(isi))
  } catch {
    /* diabaikan dengan sadar: sesi tetap hidup di memori tab ini */
  }
}

function hapusDariPeramban(): void {
  try {
    window.sessionStorage.removeItem(KUNCI_SIMPANAN)
  } catch {
    /* diabaikan dengan sadar */
  }
}
