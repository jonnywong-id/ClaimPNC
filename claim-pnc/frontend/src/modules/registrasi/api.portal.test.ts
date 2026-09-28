import { readFileSync } from 'node:fs'
import { join } from 'node:path'

import { describe, expect, it } from 'vitest'

/**
 * Setiap panggilan API modul ini WAJIB membawa portal entitas.
 *
 * # Kenapa uji ini membaca kode, bukan menjalankan layar
 *
 * Modul ini punya tujuh pemanggilan API di tujuh hook. Menguji masing-masing lewat layarnya
 * menuntut tujuh berkas uji dengan tujuh perakitan halaman, dan yang dijaga hanya SATU
 * hal — satu kata pada satu objek. Pemindaian sumber menjaga ketujuhnya sekaligus, dan
 * ikut menjaga panggilan kedelapan yang ditulis besok.
 *
 * # Kegagalan yang ia cegah
 *
 * Modul ini ditulis sebelum portal ada. Saat ia dipasang di belakang middleware portal
 * (2026-09-24), SELURUH layarnya menjawab "Portal entitas belum dipilih" — termasuk layar
 * klaim yang terbuka tepat setelah tombol Register Klaim berhasil. Klaimnya benar-benar
 * terbuat; yang gagal hanya menampilkannya, dan dari layar keduanya tampak sama saja.
 *
 * Portal menentukan basis data mana yang dibaca, yakni MILIK BADAN HUKUM MANA (`D-75`).
 * Panggilan tanpa portal ditolak, bukan jatuh ke portal bawaan — itulah yang mencegah
 * `R-20`, kebocoran data antar badan hukum yang tidak terlihat sebagai galat.
 */
describe('setiap panggilan API registrasi membawa portal', () => {
  const source = readFileSync(join(__dirname, 'api.ts'), 'utf8')

  it('tidak ada callAPI tanpa portal', () => {
    // Setiap pemanggilan dipotong sampai kurung tutup argumennya, lalu diperiksa.
    const calls = source.match(/callAPI<[^>]*>\([\s\S]*?\}\)/g) ?? []

    expect(calls.length).toBeGreaterThan(0)

    const tanpaPortal = calls.filter((call) => !/\bportal\b/.test(call))
    expect(tanpaPortal, 'panggilan berikut tidak menyebut portal').toEqual([])
  })

  it('setiap hook membaca portal aktif dari store', () => {
    const jumlahHook = (source.match(/useSession\(\(state\) => state\.token\)/g) ?? []).length
    const jumlahPortal = (source.match(/useSelectedPortal\(\(state\) => state\.alias\)/g) ?? [])
      .length

    expect(jumlahHook).toBeGreaterThan(0)
    expect(jumlahPortal).toBe(jumlahHook)
  })
})
