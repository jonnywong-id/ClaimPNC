import { describe, expect, it } from 'vitest'

import { formatMoney, formatRupiah, parseMoney, parseRupiah, remainder } from './money'

describe('formatMoney', () => {
  it('memakai pemisah ribuan Indonesia tanpa lambang Rp', () => {
    expect(formatMoney(1234567)).toBe('1.234.567')
    expect(formatMoney(0)).toBe('0')
  })

  // Nilai negatif sengaja ditampilkan apa adanya supaya kelebihan bayar terlihat.
  it('mempertahankan tanda minus', () => {
    expect(formatMoney(-2500)).toBe('-2.500')
  })

  it('membulatkan pecahan ke rupiah utuh', () => {
    expect(formatMoney(1000.6)).toBe('1.001')
  })

  it('mengembalikan 0 untuk nilai yang bukan bilangan terhingga', () => {
    expect(formatMoney(Number.NaN)).toBe('0')
    expect(formatMoney(Number.POSITIVE_INFINITY)).toBe('0')
  })
})

describe('parseMoney', () => {
  it('membuang pemisah ribuan titik maupun koma', () => {
    expect(parseMoney('1.234.567')).toBe(1234567)
    expect(parseMoney('1,234,567')).toBe(1234567)
    expect(parseMoney('  42 ')).toBe(42)
    expect(parseMoney('-500')).toBe(-500)
  })

  it('mengembalikan null untuk isian kosong', () => {
    expect(parseMoney('')).toBeNull()
    expect(parseMoney('   ')).toBeNull()
  })

  it('mengembalikan null untuk yang bukan bilangan bulat', () => {
    expect(parseMoney('1e3')).toBeNull()
    expect(parseMoney('abc')).toBeNull()
    expect(parseMoney('12 34')).toBeNull()
  })

  it('mengembalikan null bila melampaui bilangan bulat aman', () => {
    expect(parseMoney('99999999999999999999')).toBeNull()
  })
})

describe('remainder', () => {
  it('mengurangi pembayaran berjalan bila belum ada pembayaran sebelumnya', () => {
    expect(remainder(1000, 0, 300)).toBe(700)
  })

  // Perilaku yang dipertahankan (P-5): pembayaran batch berjalan tidak ikut mengurangi.
  it('mengurangi pembayaran sebelumnya saja bila ada', () => {
    expect(remainder(1000, 400, 300)).toBe(600)
  })
})

describe('formatRupiah — cabang tambahan', () => {
  it('mengembalikan teks aslinya bila bagian bulatnya bukan angka', () => {
    expect(formatRupiah('abc.00')).toBe('abc.00')
  })

  it('memperlakukan nilai null-ish sebagai kosong', () => {
    expect(formatRupiah(undefined as unknown as string)).toBe('—')
  })

  it('menampilkan nilai tanpa bagian desimal', () => {
    expect(formatRupiah('1500')).toBe('Rp 1.500')
  })
})

describe('parseRupiah — cabang tambahan', () => {
  it('menerima tanda negatif dan positif', () => {
    expect(parseRupiah('-1.000')).toBe('-1000.00')
    expect(parseRupiah('+1.000')).toBe('1000.00')
  })

  it('memperlakukan nilai null-ish sebagai kosong', () => {
    expect(parseRupiah(undefined as unknown as string)).toBeNull()
  })
})
