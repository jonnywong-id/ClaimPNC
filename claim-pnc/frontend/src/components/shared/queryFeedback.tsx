import { APIError } from '@/api/client'
import type { ServerPagination } from '@/components/DataTable'
import { ErrorMessage } from '@/components/ErrorMessage'

/**
 * Bantuan bersama untuk menampilkan hasil kueri: pelanggaran per isian, pesan galat,
 * kotak galat gangguan, dan paginasi server.
 *
 * Sebelumnya setiap layar laporan menyalin fungsi yang sama persis. Isinya tidak diubah
 * sedikit pun — hanya dipindahkan ke satu tempat.
 */

/** violationsOf mengambil pelanggaran per isian dari galat server, atau kosong. */
export function violationsOf(error: unknown): Record<string, string> {
  if (error instanceof APIError) return error.violations()
  return {}
}

/** messageOf menyusun pesan galat yang dapat dibaca pengguna. */
export function messageOf(error: unknown): string {
  if (error instanceof APIError) return error.message
  if (error instanceof Error && error.message !== '') return error.message
  return 'Coba lagi beberapa saat lagi. Bila terus berulang, hubungi tim teknis.'
}

/** failureNotice menggambar kotak galat bernada gangguan untuk sebuah kueri yang gagal. */
export function failureNotice(title: string, error: unknown) {
  return <ErrorMessage title={title} description={messageOf(error)} tone="gangguan" />
}

/** Bentuk paginasi yang dikirim server pada jawaban berhalaman. */
export type ServerPageInfo = {
  halaman: number
  ukuran: number
  total: number
  total_halaman: number
}

/**
 * pageControl menerjemahkan paginasi server ke bentuk yang dibaca DataTable. Bawaannya
 * (halaman 1, ukuran 50, nol baris) dipakai selama jawaban pertama belum tiba.
 */
export function pageControl(
  info: ServerPageInfo | undefined,
  onPageChange: (page: number) => void,
  isLoading: boolean,
): ServerPagination {
  return {
    page: info?.halaman ?? 1,
    size: info?.ukuran ?? 50,
    total: info?.total ?? 0,
    totalPage: info?.total_halaman ?? 0,
    onPageChange,
    isLoading,
  }
}
