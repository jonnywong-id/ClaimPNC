import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { callAPI } from '@/api/client'
import type { CauseOfLossListResponse, CauseOfLossResponse } from '@/api/types'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

const ROUTES = '/api/master/penyebab-kerugian'

/**
 * Kunci cache dikumpulkan di satu tempat supaya invalidasi tidak salah sasaran.
 *
 * Portal ikut di dalam kunci. Tanpa itu, berpindah entitas akan menampilkan data entitas
 * sebelumnya dari cache — pengguna melihat daftar yang masuk akal, dan tidak ada apa pun
 * di layar yang menandakan data itu milik badan hukum lain (`R-20`).
 */
const key = {
  list: (portal: string | null, token: string | null) =>
    ['master-penyebab-kerugian', portal, token] as const,
}

/**
 * Hook daftar Master Penyebab Kerugian.
 *
 * Menggantikan Report Definition `BrowseVMCauseOfLoss_RD` yang mengisi grid layar
 * `CauseOfLossInbox`.
 *
 * Seluruh baris dimuat sekaligus, tanpa paginasi server. Itu bukan kelalaian: layar Pega
 * pun memuat seluruhnya — RD-nya tidak memasang satu pun penyaring maupun pembatas baris.
 * Daftar acuan seperti ini berisi puluhan baris dan bertambah beberapa baris per tahun.
 * Layar yang datanya besar — inbox dan laporan — tidak boleh mengikuti pola ini.
 */
export function useCauseOfLossList() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: key.list(portal, token),
    queryFn: () => callAPI<CauseOfLossListResponse>(ROUTES, { token, portal }),
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
  /** Kosong berarti menambah; terisi berarti mengubah golongan dengan ID itu. */
  id?: string
  deskripsi: string
}

/**
 * Hook simpan — menambah maupun mengubah.
 *
 * Keduanya disatukan karena form-nya memang satu: sistem lama pun memakai satu halaman
 * `TempCauseOfLoss` untuk keduanya, dan membedakannya dengan mengirim `M_COL_ID` berisi
 * `"UnknownID"` saat menambah (`Activity/CNMInsertCauseOfLoss_act-Act.xml`). Di sini
 * penandanya tidak dibawa sama sekali — metode dan jalurnya yang membedakan, dan itu satu
 * baris.
 *
 * ID TIDAK pernah dikirim di badan permintaan. Pada penambahan ia dibuat server; pada
 * pengubahan ia berada di jalur URL. Mengirimnya dari klien akan membuat satu golongan
 * dapat dipindahkan ke nomor lain — dan memutus setiap rincian yang bernaung di bawahnya.
 */
export function useSaveCauseOfLoss() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: ({ id, deskripsi }: SaveFields) =>
      callAPI<CauseOfLossResponse>(id ? `${ROUTES}/${encodeURIComponent(id)}` : ROUTES, {
        metode: id ? 'PUT' : 'POST',
        body: { deskripsi },
        token,
        portal,
      }),
    onSuccess: () => {
      // Daftar dimuat ulang dari server, bukan disunting di cache. Pada penambahan, nomor
      // barunya hanya diketahui server — menebaknya di klien akan menampilkan nomor yang
      // salah sampai muat ulang berikutnya.
      void client.invalidateQueries({ queryKey: key.list(portal, token) })
    },
  })
}
