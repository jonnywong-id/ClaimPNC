import { useQuery } from '@tanstack/react-query'

import { callAPI } from '@/api/client'
import { useSession } from '@/app/session'

import type { Info, Report } from './types'

const PATH = '/api/konversi-coins'

/**
 * Keterangan koneksi LIVE/TEST dan daftar polis bawaan.
 *
 * Tanpa header portal: kedua koneksinya ditentukan konfigurasi modul (KONVERSI_LIVE_* dan
 * KONVERSI_TEST_*, sama dengan Konversi Coverage), bukan portal yang dipilih pengguna.
 */
export function useInfo() {
  const token = useSession((s) => s.token)
  return useQuery({
    queryKey: ['konversi-coins', 'keterangan', token],
    queryFn: () => callAPI<Info>(`${PATH}/keterangan`, { token }),
    enabled: token !== null,
  })
}

/** Menjalankan konversi. `ujiCoba` menjalankan lalu membatalkan seluruh tulisan. */
export function useRun() {
  const token = useSession((s) => s.token)
  return (polis: string, ujiCoba: boolean): Promise<Report> =>
    callAPI<Report>(`${PATH}/jalankan`, {
      metode: 'POST',
      token,
      body: { polis, uji_coba: ujiCoba },
    })
}
