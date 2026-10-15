import { describe, expect, it } from 'vitest'

import { mergeReports, parsePolicies } from './api'
import type { Laporan } from './types'

function report(extra: Partial<Laporan>): Laporan {
  return {
    uji_coba: true,
    mulai: '2026-10-09T01:00:00Z',
    selesai: '2026-10-09T01:00:10Z',
    polis: [],
    berhasil: 0,
    tidak_ditemukan: 0,
    gagal: 0,
    total_coverage: 0,
    total_spreading: 0,
    ...extra,
  }
}

describe('konversi coverage per kelompok', () => {
  it('memisah daftar polis seperti backend dan membuang duplikat', () => {
    expect(parsePolicies('A1\nA2, A3;A2\r\n\tA4  ')).toEqual(['A1', 'A2', 'A3', 'A4'])
    expect(parsePolicies('  ')).toEqual([])
  })

  it('menggabungkan laporan kelompok berurutan', () => {
    const first = report({ berhasil: 9, gagal: 1, total_coverage: 20, total_spreading: 30 })
    const second = report({
      mulai: '2026-10-09T01:00:11Z',
      selesai: '2026-10-09T01:00:20Z',
      berhasil: 5,
      tidak_ditemukan: 1,
      total_coverage: 7,
      total_spreading: 8,
    })
    expect(mergeReports(null, first)).toBe(first)
    expect(mergeReports(first, second)).toMatchObject({
      mulai: '2026-10-09T01:00:00Z',
      selesai: '2026-10-09T01:00:20Z',
      berhasil: 14,
      tidak_ditemukan: 1,
      gagal: 1,
      total_coverage: 27,
      total_spreading: 38,
    })
  })
})
