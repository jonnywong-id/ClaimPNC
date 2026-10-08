import { useQuery } from '@tanstack/react-query'

import { callAPI } from '@/api/client'
import type { BusinessListResponse } from '@/api/types'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

const ROUTE_BUSINESS = '/api/master/bisnis'

/**
 * Kunci cache daftar bisnis — sengaja SAMA untuk setiap layar yang memakainya, sehingga
 * daftar yang sudah dimuat satu layar langsung terpakai di layar lain.
 *
 * Portal ikut di dalam kunci karena itulah yang menentukan basis data mana yang menjawab
 * (ADR-0030), dan token supaya cache pengguna sebelumnya tidak terwarisi.
 */
function businessKey(portal: string | null, token: string | null) {
  return ['master-bisnis', portal, token] as const
}

/**
 * Hook daftar bisnis untuk saran isian Bisnis.
 *
 * MENUNTUT portal: POOLDATA.BUSINESS hidup di basis data setiap entitas, sehingga "bisnis
 * milik siapa" ditentukan portal yang aktif.
 *
 * Daftarnya jarang berubah, sehingga tidak dimuat ulang setiap kali form dibuka.
 */
export function useBusinessList() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: businessKey(portal, token),
    queryFn: () => callAPI<BusinessListResponse>(ROUTE_BUSINESS, { token, portal }),
    enabled: token !== null && portal !== null,
    staleTime: 60 * 60 * 1000,
  })
}
