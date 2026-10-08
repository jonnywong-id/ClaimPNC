import { compareCodeUnits } from '@/lib/sort'

/**
 * Nilai yang sudah terpakai pada kolom-kolom tertentu, dikumpulkan dari baris yang SEDANG
 * TERMUAT — bukan dari daftar yang dikarang.
 *
 * Dipakai sebagai saran nilai untuk kolom yang daftar pilihannya tidak ada di export
 * (R-16): ia jawaban terbaik yang tersedia atas pertanyaan "nilai apa yang sah di kolom
 * ini". Nilai kosong dibuang, dan sisanya diurutkan per unit kode supaya urutannya sama di
 * setiap peramban.
 */
export function collectKnownValues<TColumn extends string>(
  rows: readonly Readonly<Record<TColumn, string>>[],
  columns: readonly TColumn[],
): Record<string, string[]> {
  const collected: Record<string, string[]> = {}
  for (const column of columns) {
    const unique = new Set<string>()
    for (const row of rows) {
      const value = row[column]
      if (value !== '') unique.add(value)
    }
    collected[column] = [...unique].sort(compareCodeUnits)
  }
  return collected
}
