import { describe, expect, it } from 'vitest'

import { formatRupiah, parseRupiah } from './money'

describe('formatRupiah', () => {
  it('mengubah bentuk canonical menjadi teks yang enak dibaca', () => {
    expect(formatRupiah('0.00')).toBe('Rp 0')
    expect(formatRupiah('50000001.00')).toBe('Rp 50.000.001')
    expect(formatRupiah('100000000000.00')).toBe('Rp 100.000.000.000')
    expect(formatRupiah('1000.00')).toBe('Rp 1.000')
  })

  // Sen nol dibuang karena seluruh isi master berupa rupiah bulat, dan menampilkan
  // ",00" di setiap baris hanya menambah keramaian. Sen yang TIDAK nol tetap tampil —
  // membuangnya akan menyembunyikan selisih yang justru menentukan.
  it('membuang sen yang nol tetapi mempertahankan yang tidak nol', () => {
    expect(formatRupiah('12.00')).toBe('Rp 12')
    expect(formatRupiah('12.50')).toBe('Rp 12,50')
    expect(formatRupiah('12.05')).toBe('Rp 12,05')
  })

  it('menangani nilai negatif dan kosong tanpa menampilkan NaN', () => {
    expect(formatRupiah('-7.25')).toBe('-Rp 7,25')
    expect(formatRupiah('')).toBe('—')
  })

  it('dapat menampilkan tanpa lambang untuk dipakai di dalam kalimat', () => {
    expect(formatRupiah('50000001.00', { withoutSymbol: true })).toBe('50.000.001')
  })
})

describe('parseRupiah', () => {
  // Pengguna boleh mengetik dengan cara apa pun yang lazim dipakai orang menulis rupiah.
  it('menerima bentuk yang biasa diketik orang', () => {
    expect(parseRupiah('50000000')).toBe('50000000.00')
    expect(parseRupiah('50.000.000')).toBe('50000000.00')
    expect(parseRupiah('Rp 50.000.000')).toBe('50000000.00')
    expect(parseRupiah('rp50.000.000')).toBe('50000000.00')
    expect(parseRupiah('  50 000 000 ')).toBe('50000000.00')
    expect(parseRupiah('50.000.000,50')).toBe('50000000.50')
    expect(parseRupiah('12,5')).toBe('12.50')
  })

  it('menolak yang bukan nilai uang', () => {
    expect(parseRupiah('')).toBeNull()
    expect(parseRupiah('lima juta')).toBeNull()
    expect(parseRupiah('12,345')).toBeNull()
    expect(parseRupiah('1e8')).toBeNull()
  })

  // Sifat inilah yang menjaga nilai tidak bergeser saat bolak-balik antara layar dan
  // server: yang ditampilkan lalu diketik ulang harus menghasilkan nilai yang sama.
  it('membalik formatRupiah tanpa menggeser nilainya', () => {
    for (const canonical of ['0.00', '50000001.00', '100000000000.00', '12.50']) {
      expect(parseRupiah(formatRupiah(canonical))).toBe(canonical)
    }
  })
})
