/**
 * compareCodeUnits membandingkan dua teks menurut code unit UTF-16 — urutan yang SAMA
 * PERSIS dengan `Array.prototype.sort()` tanpa pembanding.
 *
 * # Kenapa bukan `localeCompare`
 *
 * `localeCompare` mengurutkan menurut aturan bahasa: huruf besar dan kecil berdampingan,
 * dan aksen diabaikan. Layar yang memakai `.sort()` polos selama ini menampilkan huruf
 * besar lebih dulu (`"B"` sebelum `"a"`). Mengganti ke `localeCompare` akan mengubah urutan
 * pilihan yang dilihat pengguna — perubahan perilaku, bukan perbaikan penulisan.
 *
 * Pembanding ini ada supaya urutannya dinyatakan EKSPLISIT di kode, tanpa mengubahnya.
 */
export function compareCodeUnits(a: string, b: string): number {
  if (a < b) return -1
  if (a > b) return 1
  return 0
}
