import { create } from 'zustand'

const KUNCI_SIMPANAN = 'claim-pnc.portal'

type KeadaanPortal = {
  /** Alias portal yang sedang dipilih; null berarti belum ditentukan. */
  alias: string | null
  pilih: (alias: string) => void
  bersihkan: () => void
}

/**
 * Simpanan portal yang sedang aktif.
 *
 * Portal menentukan basis data mana yang dikenai kueri modul bisnis (ADR-0030), karena
 * itu pilihannya bertahan melewati muat ulang halaman. Ia disimpan terpisah dari sesi:
 * berpindah portal TIDAK menuntut login ulang, sehingga keduanya memang berumur
 * berbeda.
 */
export const gunakanPortalTerpilih = create<KeadaanPortal>((set) => ({
  alias: muatDariPeramban(),

  pilih: (alias) => {
    simpanKePeramban(alias)
    set({ alias })
  },

  bersihkan: () => {
    hapusDariPeramban()
    set({ alias: null })
  },
}))

function muatDariPeramban(): string | null {
  try {
    return window.sessionStorage.getItem(KUNCI_SIMPANAN)
  } catch {
    // sessionStorage dapat ditolak peramban (mode privat, kebijakan perusahaan).
    // Aplikasi tetap harus jalan; pengguna cukup memilih portalnya lagi.
    return null
  }
}

function simpanKePeramban(alias: string): void {
  try {
    window.sessionStorage.setItem(KUNCI_SIMPANAN, alias)
  } catch {
    /* diabaikan dengan sadar: pilihan tetap hidup di memori tab ini */
  }
}

function hapusDariPeramban(): void {
  try {
    window.sessionStorage.removeItem(KUNCI_SIMPANAN)
  } catch {
    /* diabaikan dengan sadar */
  }
}
