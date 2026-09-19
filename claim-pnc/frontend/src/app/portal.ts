import { create } from 'zustand'

const STORAGE_KEY = 'claim-pnc.portal'

type PortalState = {
  /** Alias portal yang sedang dipilih; null berarti belum ditentukan. */
  alias: string | null
  select: (alias: string) => void
  clear: () => void
}

/**
 * Simpanan portal yang sedang aktif.
 *
 * Portal menentukan basis data mana yang dikenai kueri modul bisnis (ADR-0030), karena
 * itu pilihannya bertahan melewati muat ulang halaman. Ia disimpan terpisah dari sesi:
 * berpindah portal TIDAK menuntut login ulang, sehingga keduanya memang berumur
 * berbeda.
 */
export const useSelectedPortal = create<PortalState>((set) => ({
  alias: loadFromBrowser(),

  select: (alias) => {
    saveToBrowser(alias)
    set({ alias })
  },

  clear: () => {
    clearFromBrowser()
    set({ alias: null })
  },
}))

function loadFromBrowser(): string | null {
  try {
    return window.sessionStorage.getItem(STORAGE_KEY)
  } catch {
    // sessionStorage dapat ditolak peramban (mode privat, kebijakan perusahaan).
    // Aplikasi tetap harus jalan; pengguna cukup memilih portalnya lagi.
    return null
  }
}

function saveToBrowser(alias: string): void {
  try {
    window.sessionStorage.setItem(STORAGE_KEY, alias)
  } catch {
    /* diabaikan dengan sadar: pilihan tetap hidup di memori tab ini */
  }
}

function clearFromBrowser(): void {
  try {
    window.sessionStorage.removeItem(STORAGE_KEY)
  } catch {
    /* diabaikan dengan sadar */
  }
}
