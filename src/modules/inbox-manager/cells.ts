import type { Cell, Panel, TabColumn } from "./types";

/**
 * Pembaca sel grid dashboard, dipakai bersama ketiga tampilan dashboard.
 *
 * # Nilai uang tetap TEKS sampai ditampilkan
 *
 * Ia datang sebagai teks presisi penuh dari basis data dan tidak pernah melewati `Number` —
 * mengubahnya menjadi angka JavaScript berarti jumlah rupiah melewati bilangan pecahan biner,
 * yang `I-12` larang. Pemisah ribuan disisipkan pada teksnya.
 */

/** cellText menyusun teks sebuah sel sesuai jenis kolomnya. */
export function cellText(cell: Cell | undefined, money = false): string {
  if (!cell) return "";

  if (money) return formatAmount(cell.nilai ?? "");
  if (cell.teks !== undefined && cell.teks !== "") return cell.teks;
  if (cell.jumlah !== undefined) return formatCount(cell.jumlah);

  return "";
}

/** countOf membaca pencacah sebuah sel; sel kosong dibaca nol. */
export function countOf(cell: Cell | undefined): number {
  return cell?.jumlah ?? 0;
}

/** formatCount menulis bilangan bulat dengan pemisah ribuan titik. */
export function formatCount(value: number): string {
  return formatAmount(String(value));
}

/**
 * formatAmount menyisipkan pemisah ribuan tanpa mengubah nilainya.
 *
 * Ia bekerja pada TEKS: pembulatan yang terjadi saat teks presisi penuh diubah menjadi
 * `number` tidak dapat dibatalkan, dan pada jumlah rupiah yang besar ia mengubah digit
 * terakhir.
 */
export function formatAmount(value: string): string {
  const clean = value.trim();
  if (clean === "") return "";

  const negative = clean.startsWith("-");
  const unsigned = negative ? clean.slice(1) : clean;

  const [whole, fraction] = unsigned.split(".");
  if (whole === undefined || !/^\d+$/.test(whole)) return clean;

  const grouped = whole.replace(/\B(?=(\d{3})+(?!\d))/g, ".");
  const tail = fraction ? `,${fraction}` : "";

  // Tanda minus TIPOGRAFIS, bukan tanda hubung — mengikuti rancangan layar: nilai reject
  // negatif harus terbaca dari tandanya, bukan hanya dari warnanya.
  return `${negative ? "−" : ""}${grouped}${tail}`;
}

/** isNegative menyatakan apakah teks angka bernilai negatif. */
export function isNegative(cell: Cell | undefined): boolean {
  return (cell?.nilai ?? "").trim().startsWith("-");
}

/**
 * rowKeyOf menyusun kunci baris React dari sel dimensinya.
 *
 * Baris dashboard tidak punya kunci dari server — ia memang tidak butuh, karena tidak dapat
 * diputuskan. Gabungan seluruh sel teksnya membuat dua baris yang berbeda tidak pernah berbagi
 * kunci.
 */
export function rowKeyOf(panel: Panel, row: Record<string, Cell>, index: number): string {
  const key = panel.kolom
    .map((column) => row[column.kunci]?.teks ?? "")
    .filter((text) => text !== "")
    .join("|");
  return key || String(index);
}

/**
 * isDimension menyatakan kolom berisi TEKS (COB, penyebab kerugian), bukan pencacah atau nilai
 * uang. Dibaca dari seluruh baris kolom itu — pencacah bernilai nol dikirim sebagai sel
 * kosong, sehingga satu sel saja tidak cukup untuk menentukannya.
 */
export function isDimension(column: TabColumn, rows: Record<string, Cell>[]): boolean {
  if (column.uang) return false;
  return rows.some((row) => (row[column.kunci]?.teks ?? "") !== "");
}
