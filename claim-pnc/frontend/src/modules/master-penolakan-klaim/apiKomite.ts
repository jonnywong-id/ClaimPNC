import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { callAPI } from '@/api/client'
import type {
  CommitteeRejectionInput,
  CommitteeRejectionListResponse,
  CommitteeRejectionResponse,
} from '@/api/types'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

const ROUTE = '/api/master/penolakan-komite'

/**
 * Kunci cache tab Penolakan Komite.
 *
 * BERBEDA dari kunci tab sebelah meski keduanya hidup di satu layar, sehingga keduanya
 * tidak pernah saling menimpa cache. Portal ikut dengan alasan yang sama seperti tab
 * sebelah (R-20).
 */
function listKey(portal: string | null, token: string | null) {
  return ['master-penolakan-komite', portal, token] as const
}

/** Hook daftar Penolakan Komite. */
export function useCommitteeRejectionList() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: listKey(portal, token),
    queryFn: () => callAPI<CommitteeRejectionListResponse>(ROUTE, { token, portal }),
    enabled: token !== null && portal !== null,
  })
}

/** Hook penambahan Penolakan Komite. */
export function useCreateCommitteeRejection() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: (input: CommitteeRejectionInput) =>
      callAPI<CommitteeRejectionResponse>(ROUTE, { metode: 'POST', body: input, token, portal }),
    // Daftar dimuat ulang dari server, BUKAN ditambahi barisnya di sisi klien: ID baru
    // diterbitkan server dari isi tabel, dan petugas lain dapat menambah baris pada saat
    // yang sama.
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: listKey(portal, token) })
    },
  })
}

/**
 * Hook pengubahan Penolakan Komite.
 *
 * Inilah tombol "ubah" pada setiap baris grid layar lama. Hanya catatannya yang berubah;
 * IDMASTER adalah kunci dan tidak pernah ikut.
 */
export function useUpdateCommitteeRejection() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: CommitteeRejectionInput }) =>
      callAPI<CommitteeRejectionResponse>(`${ROUTE}/${encodeURIComponent(id)}`, {
        metode: 'PUT',
        body: input,
        token,
        portal,
      }),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: listKey(portal, token) })
    },
  })
}
