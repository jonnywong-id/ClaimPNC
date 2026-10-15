import { create } from 'zustand'

import type { User } from '@/api/types'

const STORAGE_KEY = 'claim-pnc.sesi'

export type SessionState = {
  token: string | null
  user: User | null
  validUntil: string | null
  login: (value: { token: string; user: User; validUntil: string }) => void
  refreshValidity: (validUntil: string) => void
  clear: () => void
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
export const useSession = create<SessionState>((set) => ({
  ...loadFromBrowser(),

  login: ({ token, user, validUntil }) => {
    saveToBrowser({ token, user, validUntil })
    set({ token, user, validUntil })
  },

  refreshValidity: (validUntil) => {
    set((previous) => {
      if (previous.token && previous.user) {
        saveToBrowser({
          token: previous.token,
          user: previous.user,
          validUntil,
        })
      }
      return { validUntil }
    })
  },

  clear: () => {
    clearFromBrowser()
    set({ token: null, user: null, validUntil: null })
  },
}))

type StoredSession = { token: string; user: User; validUntil: string }

function loadFromBrowser(): Pick<SessionState, 'token' | 'user' | 'validUntil'> {
  const empty = { token: null, user: null, validUntil: null }
  try {
    const raw = window.sessionStorage.getItem(STORAGE_KEY)
    if (!raw) return empty
    const content = JSON.parse(raw) as Partial<StoredSession>
    if (!content.token || !content.user || !content.validUntil) return empty
    return { token: content.token, user: content.user, validUntil: content.validUntil }
  } catch {
    // sessionStorage dapat ditolak peramban (mode privat, kebijakan perusahaan).
    // Aplikasi tetap harus jalan; pengguna cukup masuk ulang setelah muat ulang.
    return empty
  }
}

function saveToBrowser(content: StoredSession): void {
  try {
    window.sessionStorage.setItem(STORAGE_KEY, JSON.stringify(content))
  } catch {
    /* diabaikan dengan sadar: sesi tetap hidup di memori tab ini */
  }
}

function clearFromBrowser(): void {
  try {
    window.sessionStorage.removeItem(STORAGE_KEY)
  } catch {
    /* diabaikan dengan sadar */
  }
}
