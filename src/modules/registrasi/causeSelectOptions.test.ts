import { describe, expect, it } from 'vitest'

import { causeSelectOptions, coverageShareE4 } from './ClaimPage'

describe('pilihan Penyebab Kerugian', () => {
  const master = [
    { id: '11997', nama: 'FIRE - OPEN FLAME' },
    { id: '12010', nama: 'FIRE - SHORT CIRCUIT' },
  ]

  it('menampilkan nama dari master, dengan D_COL_ID sebagai nilai', () => {
    expect(causeSelectOptions(master, '11997')).toEqual([
      { value: '11997', label: 'FIRE - OPEN FLAME' },
      { value: '12010', label: 'FIRE - SHORT CIRCUIT' },
    ])
  })

  it('nilai tersimpan yang tidak ada di master tetap tampil, tidak dikosongkan', () => {
    expect(causeSelectOptions(master, 'ACCIDENT')[0]).toEqual({ value: 'ACCIDENT', label: 'ACCIDENT' })
    expect(causeSelectOptions(master, '')).toHaveLength(2)
  })
})

describe('total share per jaminan', () => {
  it('97,5 + 2,5 adalah 100%, baris terhapus tidak dihitung', () => {
    const coverage = {
      id: '100819', nama: 'FLEXAS', penyebab_kerugian: '', tsi: '',
      spreading: [
        { jenis_treaty: '10010', nama: 'BPPDAN', share: '2.5', objek_fac_offer: '', dihapus: false },
        { jenis_treaty: '10015', nama: 'FAC-OUT', share: '97,5', objek_fac_offer: '', dihapus: false },
        { jenis_treaty: '10001', nama: 'OR', share: '50', objek_fac_offer: '', dihapus: true },
      ],
    }
    expect(coverageShareE4(coverage)).toBe(1_000_000)
  })
})
