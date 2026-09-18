import { useQuery } from '@tanstack/react-query'

import { callAPI } from '@/api/client'
import type { PortalListResponse } from '@/api/types'
import { useSession } from '@/app/session'

/**
 * Hook daftar portal.
 *
 * Daftarnya berada di balik sesi: pemilih portal ada di dalam aplikasi, bukan di layar
 * masuk. ADR-0030 menetapkan berpindah portal **tidak menuntut login ulang**, sehingga
 * portal tidak perlu — dan tidak boleh — menjadi bagian dari alur masuk.
 */
export function usePortalList() {
  const token = useSession((state) => state.token)

  return useQuery({
    queryKey: ['portal', token],
    queryFn: () => callAPI<PortalListResponse>('/api/portal', { token }),
    enabled: token !== null,
    // Daftar entitas nyaris tidak pernah berubah dalam satu sesi kerja; memuatnya
    // ulang setiap kali komponen dipasang hanya membebani basis data portal utama.
    staleTime: 5 * 60 * 1000,
  })
}
