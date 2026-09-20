import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { callAPI } from '@/api/client'
import type { ClaimStatusListResponse, ClaimStatusResponse } from '@/api/types'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

const ROUTES = '/api/master/status-klaim'

/**
 * Kunci cache dikumpulkan di satu tempat supaya invalidasi tidak salah sasaran.
 *
 * Portal ikut di dalam kunci sejak modul ini menjadi per portal (2026-09-19). Tanpa itu,
 * berpindah entitas akan menampilkan data entitas sebelumnya dari cache — pengguna
 * melihat daftar yang masuk akal, dan tidak ada apa pun di layar yang menandakan data itu
 * milik badan hukum lain (`R-20`).
 */
const key = {
  list: (portal: string | null, token: string | null) =>
    ['master-status-klaim', portal, token] as const,
}

/**
 * Hook daftar Master Status Klaim.
 *
 * Menggantikan Report Definition `BrowseVStsClaim_RD` yang mengisi grid layar
 * `StatusClaimInbox`.
 *
 * Seluruh baris dimuat sekaligus, tanpa paginasi server. Itu keputusan yang diambil
 * dengan angka: isinya 33 baris dan bertambah beberapa baris per tahun. Layar yang
 * datanya besar — inbox dan laporan — tidak boleh mengikuti pola ini.
 */
export function useClaimStatusList() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: key.list(portal, token),
    queryFn: () => callAPI<ClaimStatusListResponse>(ROUTES, { token, portal }),
    // Tidak dijalankan sebelum portal dipilih: backend memang menolak permintaan tanpa
    // portal, tetapi menembaknya hanya untuk menerima penolakan akan menampilkan pesan
    // galat pada layar yang sebenarnya belum siap dibuka.
    enabled: token !== null && portal !== null,
    // Master nyaris tidak pernah berubah dalam satu sesi kerja. Lima menit menahan
    // pemuatan ulang yang tidak perlu, sementara tombol Refresh tetap tersedia bagi
    // pengguna yang tahu datanya baru saja diubah orang lain.
    staleTime: 5 * 60 * 1000,
  })
}

type SaveFields = {
  /** Kosong berarti menambah; terisi berarti mengubah status dengan kode itu. */
  kode?: string
  label: string
}

/**
 * Hook simpan — menambah maupun mengubah.
 *
 * Keduanya disatukan karena form-nya memang satu: sistem lama pun memakai satu halaman
 * `TempStsClaim` untuk keduanya, dan membedakannya dengan ada-tidaknya `LSC_ID`.
 * Perbedaannya hanya pada metode dan jalur, dan itu satu baris.
 *
 * Kode TIDAK pernah dikirim di badan permintaan. Pada penambahan ia dibuat server; pada
 * pengubahan ia berada di jalur URL.
 */
export function useSaveClaimStatus() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: ({ kode, label }: SaveFields) =>
      callAPI<ClaimStatusResponse>(kode ? `${ROUTES}/${encodeURIComponent(kode)}` : ROUTES, {
        metode: kode ? 'PUT' : 'POST',
        body: { label },
        token,
        portal,
      }),
    onSuccess: () => {
      // Daftar dimuat ulang dari server, bukan disunting di cache. Pada penambahan,
      // kode barunya hanya diketahui server — menebaknya di klien akan menampilkan
      // kode yang salah sampai muat ulang berikutnya.
      void client.invalidateQueries({ queryKey: key.list(portal, token) })
    },
  })
}
