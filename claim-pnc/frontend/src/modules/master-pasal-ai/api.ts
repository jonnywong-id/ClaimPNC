import { keepPreviousData, useQuery } from '@tanstack/react-query'

import { callAPI } from '@/api/client'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import type { ClauseAIListResponse } from './types'

const ROUTE = '/api/master/pasal-ai'

/**
 * listKey menyertakan portal, token, kata kunci, DAN nomor halaman.
 *
 * Portal ikut karena itulah yang menentukan basis data mana yang menjawab (`ADR-0030`).
 * Tanpa itu, berpindah entitas akan menampilkan data entitas sebelumnya dari cache —
 * pengguna melihat daftar yang masuk akal, dan tidak ada apa pun di layar yang menandakan
 * ia milik badan hukum lain (`R-20`).
 *
 * Kata kunci dan nomor halaman ikut karena **paginasinya dikerjakan server**: setiap
 * kombinasi keduanya adalah permintaan yang berbeda, dengan jawaban yang berbeda. Pada
 * layar master lain keduanya tidak ada di kunci cache, karena di sana seluruh baris memang
 * sudah di tangan dan penyaringnya bekerja di peramban.
 */
function listKey(
  portal: string | null,
  token: string | null,
  keyword: string,
  page: number,
) {
  return ['master-pasal-ai', portal, token, keyword, page] as const
}

/**
 * Hook daftar Master Pasal AI.
 *
 * Tidak dijalankan sebelum portal dipilih. Backend memang menolak permintaan tanpa portal
 * (`TKT-F6-002`), tetapi menembaknya lebih dulu hanya untuk menerima penolakan akan
 * menampilkan pesan galat pada layar yang sebenarnya belum siap dibuka.
 *
 * # `keepPreviousData` — dan kenapa ia penting justru di layar ini
 *
 * Berpindah halaman mengganti kunci cache, sehingga tanpa ini tabelnya **berkedip kosong**
 * di setiap perpindahan: baris hilang, tinggi halaman melompat, lalu baris baru muncul.
 * Dengan ini, baris halaman sebelumnya tetap tergambar sampai halaman berikutnya tiba.
 *
 * Layar master lain tidak membutuhkannya karena perpindahan halamannya tidak menembak
 * server sama sekali.
 */
export function useClauseAIList(keyword: string, page: number) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  const query = new URLSearchParams()
  if (keyword !== '') query.set('cari', keyword)
  if (page > 1) query.set('halaman', String(page))
  const suffix = query.toString() === '' ? '' : `?${query.toString()}`

  return useQuery({
    queryKey: listKey(portal, token, keyword, page),
    queryFn: () => callAPI<ClauseAIListResponse>(`${ROUTE}${suffix}`, { token, portal }),
    enabled: token !== null && portal !== null,
    placeholderData: keepPreviousData,
  })
}
