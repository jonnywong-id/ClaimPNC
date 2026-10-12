import { describe, expect, it } from 'vitest'

import { claimDetailPath } from './claim'

describe('claimDetailPath', () => {
  it('menyusun jalur halaman klaim dari NOMORNYA', () => {
    // Bukan dari `pzInsKey`: `D-22` melarang kunci teknis Pega dipakai sebagai alamat.
    expect(claimDetailPath('PNCN.26.0002')).toBe('/registrasi/klaim/PNCN.26.0002')
  })

  it('melayani nomor warisan Pega dengan cara yang sama', () => {
    // Halaman tujuannya membaca `POOLDATA.T_CLAIM_PNC` — tabel klaim warisan — sehingga
    // `PNC-xxxx` bukan bentuk yang perlu ditolak di sini.
    expect(claimDetailPath('PNC-2856')).toBe('/registrasi/klaim/PNC-2856')
  })

  it('menyandikan nomor yang memuat karakter jalur', () => {
    // Nomor yang berlaku tidak pernah memuatnya, tetapi jalur yang tidak disandikan akan
    // memecah rute tanpa satu pun galat yang terbaca.
    expect(claimDetailPath('A/B')).toBe('/registrasi/klaim/A%2FB')
  })
})
