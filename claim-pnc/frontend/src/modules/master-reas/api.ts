import { useQuery } from '@tanstack/react-query'

import { callAPI } from '@/api/client'
import type { ReasMemberListResponse } from '@/api/types'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

const ROUTE = '/api/master/reas'

/**
 * listKey menyertakan portal DAN token.
 *
 * Portal ikut karena itulah yang menentukan basis data mana yang menjawab (`ADR-0030`).
 * Tanpa itu, berpindah entitas akan menampilkan data entitas sebelumnya dari cache —
 * pengguna melihat daftar mitra reasuransi yang masuk akal, dan tidak ada apa pun di layar
 * yang menandakan mereka milik badan hukum lain (`R-20`).
 *
 * TANPA penyaring status di kunci cache: `POOLDATA.T_REINSURER` tidak punya kolom
 * persetujuan, sehingga layar ini tidak bertab dan hanya ada satu daftar per portal.
 */
function listKey(portal: string | null, token: string | null) {
  return ['master-reas', portal, token] as const
}

/**
 * Hook daftar Master Reas.
 *
 * Tidak dijalankan sebelum portal dipilih. Backend memang menolak permintaan tanpa portal
 * (`TKT-F6-002`), tetapi menembaknya lebih dulu hanya untuk menerima penolakan akan
 * menampilkan pesan galat pada layar yang sebenarnya belum siap dibuka.
 *
 * # HANYA hook baca, dan itu bukan kelalaian
 *
 * Tidak ada `useCreate...` maupun `useSave...` di berkas ini. Layar lamanya memang tidak
 * punya jalur tulis: harness `DataMemberReas` memuat satu grid dan satu tombol Refresh, dan
 * satu-satunya penulis `T_REINSURER` di sistem lama adalah alur PLA/DLA lewat
 * `Database/UPDATEREAS.prc` — dipanggil `UpdateDetailPLA2` dan `UpdateDetailDLA2`, bukan
 * layar ini. Endpoint tulisnya pun tidak ada di server.
 *
 * # Penyaring `cari` milik endpoint ini TIDAK dipakai layar
 *
 * Endpoint-nya menerimanya, dan ia memang dibuat sebagai pengganti daftar Pega yang tidak
 * punya kotak pencarian sama sekali. Yang dipakai layar sekarang adalah pencarian bawaan
 * `DataTable`, sama seperti seluruh layar master lain — pencarian kedua di kepala halaman
 * hanya akan membingungkan.
 *
 * Perpindahan ke penyaringan sisi server dilakukan bersama paginasi sisi server
 * (`TKT-U2-001`), bukan sendirian.
 */
export function useReasMemberList() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: listKey(portal, token),
    queryFn: () => callAPI<ReasMemberListResponse>(ROUTE, { token, portal }),
    enabled: token !== null && portal !== null,
  })
}
