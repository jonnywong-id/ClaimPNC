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

/** Menjalankan konversi. `ujiCoba` menjalankan lalu membatalkan seluruh tulisan. */
export function useJalankan() {
  const token = useSession((s) => s.token)
  return (polis: string, ujiCoba: boolean): Promise<Laporan> =>
    callAPI<Laporan>(`${PATH}/jalankan`, {
      metode: 'POST',
      token,
      body: { polis, uji_coba: ujiCoba },
    })
}
