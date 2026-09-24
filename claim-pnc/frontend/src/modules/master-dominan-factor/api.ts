import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { callAPI } from '@/api/client'
import type { DominantFactorListResponse, DominantFactorResponse } from '@/api/types'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

const ROUTES = '/api/master/dominan-factor'

/**
 * Kunci cache dikumpulkan di satu tempat supaya invalidasi tidak salah sasaran.
 *
 * Portal ikut di dalam kunci. Tanpa itu, berpindah entitas akan menampilkan data entitas
 * sebelumnya dari cache — pengguna melihat daftar yang masuk akal, dan tidak ada apa pun
 * di layar yang menandakan data itu milik badan hukum lain (`R-20`).
 */
const key = {
  list: (portal: string | null, token: string | null) =>
    ['master-dominan-factor', portal, token] as const,
}

/**
 * Hook daftar Master Dominan Factor.
 *
 * Menggantikan `RDB List/GetDataDominanFactor-SQL.xml` yang mengisi grid harness
 * `DetailDominanFactor`.
 *
 * Seluruh baris dimuat sekaligus, tanpa paginasi server — sama seperti kueri lamanya,
 * yang memang tidak punya satu pun pembatas baris. Layar yang datanya besar — inbox dan
 * laporan — tidak boleh mengikuti pola ini.
 */
export function useDominantFactorList() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: key.list(portal, token),
    queryFn: () => callAPI<DominantFactorListResponse>(ROUTES, { token, portal }),
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
  /** Kosong berarti menambah; terisi berarti mengubah faktor dengan ID itu. */
  id?: string
  nama: string
}

/**
 * Hook simpan — menambah maupun mengubah.
 *
 * Keduanya disatukan karena form-nya memang satu: sistem lama pun memakai satu halaman
 * `TempFactor` untuk keduanya, dan membedakannya lewat `TempFactor.Status` yang diisi
 * `"Insert"` atau `"Update"` oleh dua data transform. Di sini pembedanya metode HTTP,
 * dan itu satu baris.
 *
 * ID TIDAK pernah dikirim di badan permintaan. Pada penambahan ia dibuat server; pada
 * pengubahan ia berada di jalur URL.
 */
export function useSaveDominantFactor() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: ({ id, nama }: SaveFields) =>
      callAPI<DominantFactorResponse>(id ? `${ROUTES}/${encodeURIComponent(id)}` : ROUTES, {
        metode: id ? 'PUT' : 'POST',
        body: { nama },
        token,
        portal,
      }),
    onSuccess: () => {
      // Daftar dimuat ulang dari server, bukan disunting di cache. Pada penambahan,
      // nomornya hanya diketahui server — menebaknya di klien akan menampilkan nomor
      // yang salah sampai muat ulang berikutnya.
      void client.invalidateQueries({ queryKey: key.list(portal, token) })
    },
  })
}
