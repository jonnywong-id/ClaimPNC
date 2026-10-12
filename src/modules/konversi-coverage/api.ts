import { useQuery } from '@tanstack/react-query'

import { callAPI } from '@/api/client'
import { useSession } from '@/app/session'

import type { Keterangan, Laporan } from './types'

const PATH = '/api/konversi-coverage'

/**
 * Keterangan koneksi LIVE/TEST dan daftar polis bawaan.
 *
 * Tanpa header portal: kedua koneksinya ditentukan konfigurasi modul, bukan portal yang
 * dipilih pengguna.
 */
export function useKeterangan() {
  const token = useSession((s) => s.token)
  return useQuery({
    queryKey: ['konversi-coverage', 'keterangan', token],
    queryFn: () => callAPI<Keterangan>(`${PATH}/keterangan`, { token }),
    enabled: token !== null,
  })
}

/**
 * Jumlah polis per permintaan. Satu polis butuh sekitar satu sampai tiga detik; daftar 96
 * polis dalam SATU permintaan berjalan hampir lima menit dan diputus di tengah jalan oleh
 * lapisan di antara peramban dan backend, sehingga layar hanya menerima "Terjadi kesalahan
 * pada sistem." tanpa laporan apa pun. Per 10 polis, setiap permintaan selesai dalam
 * hitungan detik.
 */
export const BATCH_SIZE = 10

/** Pemisah daftar polis — sama dengan `konversicoverage.ParsePolicyList` di backend. */
export function parsePolicies(text: string): string[] {
  return [...new Set(text.split(/[\s,;]+/).map((p) => p.trim()).filter((p) => p !== ''))]
}

/** Galat satu kelompok, membawa laporan polis yang sudah selesai sebelumnya. */
export class BatchError extends Error {
  constructor(
    message: string,
    readonly partial: Laporan | null,
  ) {
    super(message)
  }
}

/** Menggabungkan laporan kelompok berurutan menjadi satu laporan. */
export function mergeReports(a: Laporan | null, b: Laporan): Laporan {
  if (!a) return b
  return {
    uji_coba: b.uji_coba,
    mulai: a.mulai,
    selesai: b.selesai,
    polis: [...a.polis, ...b.polis],
    berhasil: a.berhasil + b.berhasil,
    tidak_ditemukan: a.tidak_ditemukan + b.tidak_ditemukan,
    gagal: a.gagal + b.gagal,
    total_coverage: a.total_coverage + b.total_coverage,
    total_spreading: a.total_spreading + b.total_spreading,
  }
}

/**
 * Menjalankan konversi per BATCH_SIZE polis, berurutan, lalu menggabungkan laporannya.
 * `ujiCoba` menjalankan lalu membatalkan seluruh tulisan. Kelompok yang gagal menghentikan
 * sisanya; polis kelompok sebelumnya sudah selesai dan ikut dikembalikan lewat BatchError.
 */
export function useJalankan() {
  const token = useSession((s) => s.token)
  return async (
    polis: string,
    ujiCoba: boolean,
    onProgress?: (done: number, total: number) => void,
  ): Promise<Laporan> => {
    const list = parsePolicies(polis)
    let report: Laporan | null = null
    onProgress?.(0, list.length)
    for (let i = 0; i < list.length; i += BATCH_SIZE) {
      const batch = list.slice(i, i + BATCH_SIZE)
      try {
        const part = await callAPI<Laporan>(`${PATH}/jalankan`, {
          metode: 'POST',
          token,
          body: { polis: batch.join(','), uji_coba: ujiCoba },
        })
        report = mergeReports(report, part)
      } catch (e) {
        const reason = e instanceof Error ? e.message : String(e)
        throw new BatchError(
          `Polis ke-${i + 1} sampai ${i + batch.length} gagal: ${reason} ` +
            `${i} polis sebelumnya sudah selesai.`,
          report,
        )
      }
      onProgress?.(i + batch.length, list.length)
    }
    if (!report) throw new BatchError('Daftar nomor polis kosong.', null)
    return report
  }
}
