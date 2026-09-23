import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { callAPI } from '@/api/client'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import type {
  ProtectionDetail,
  ProtectionFields,
  ProtectionFilter,
  ProtectionListResponse,
} from './types'

const PATH = '/api/input-req-protection'

/**
 * Banyaknya baris per halaman.
 *
 * Mengikuti `pyPageSize` layar lama, yang bernilai 50 pada
 * `Report Definition/InboxReqOpenProtection_RD-RD.xml`. Backend menolak permintaan di atas
 * 100.
 */
export const PAGE_SIZE = 50

/**
 * Kunci cache dikumpulkan di satu tempat supaya invalidasi tidak salah sasaran.
 *
 * PORTAL ikut menjadi bagian kunci, dan itu bukan kerapian: dua portal adalah dua badan
 * hukum dengan basis data berbeda (`ADR-0030`). Menyimpan keduanya di bawah satu kunci akan
 * membuat perpindahan portal menampilkan proteksi entitas sebelumnya — kebocoran yang
 * tampil sebagai layar normal, persis `R-20`.
 */
const keys = {
  all: ['input-req-protection'] as const,
  list: (portal: string | null, token: string | null, f: ProtectionFilter) =>
    ['input-req-protection', 'daftar', portal, token, f.search ?? '', f.offset ?? 0] as const,
  detail: (portal: string | null, token: string | null, number: string) =>
    ['input-req-protection', 'detail', portal, token, number] as const,
}

function buildPath(f: ProtectionFilter): string {
  const params = new URLSearchParams()
  if (f.search?.trim()) params.set('cari', f.search.trim())
  if (f.offset) params.set('lewati', String(f.offset))
  params.set('batas', String(PAGE_SIZE))

  return `${PATH}?${params.toString()}`
}

/**
 * Hook daftar permintaan proteksi yang belum diakseptasi.
 *
 * Menggantikan `Report Definition/InboxReqOpenProtection_RD-RD.xml`, yang menyaring
 * `.AcceptStatus IS NULL` dan mengurutkan menurun.
 *
 * # Penyaringan dikerjakan SERVER
 *
 * Daftarnya hanya dibatasi "belum diakseptasi", sehingga ia tumbuh tanpa batas seiring
 * waktu. Menyaringnya di peramban berarti hasil pencarian hanya menyentuh halaman yang
 * sedang terbuka — dan itu BOHONG: pengguna diberi tahu sesuatu tidak ada padahal ia ada di
 * halaman lain.
 *
 * # Tidak dijalankan sebelum portal dipilih
 *
 * Backend memang menolak permintaan tanpa portal (`TKT-F6-002`), tetapi menembaknya lebih
 * dulu hanya untuk menerima penolakan adalah perjalanan jaringan yang sia-sia.
 */
export function useProtectionList(filter: ProtectionFilter) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: keys.list(portal, token, filter),
    queryFn: () => callAPI<ProtectionListResponse>(buildPath(filter), { token, portal }),
    enabled: token !== null && portal !== null,

    // Daftar ini berubah setiap kali ada yang membuat atau mengakseptasi proteksi. Data
    // dianggap usang seketika, tetapi tidak ditembak ulang sendiri — pengguna yang menekan
    // Muat ulang.
    staleTime: 0,
  })
}

/** Hook pembacaan satu permintaan proteksi beserta isian formnya. */
export function useProtectionDetail(number: string | null) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: keys.detail(portal, token, number ?? ''),
    queryFn: () =>
      callAPI<ProtectionDetail>(`${PATH}/${encodeURIComponent(number ?? '')}`, {
        token,
        portal,
      }),
    enabled: token !== null && portal !== null && number !== null && number !== '',
  })
}

/**
 * Hook pembuatan permintaan proteksi baru.
 *
 * Nomor proteksi TIDAK dikirim: ia diterbitkan server dari pencacah tahunan, dan dikembalikan
 * lewat respons. Mengirimnya dari sini akan membuat siapa pun dapat menentukan nomornya
 * sendiri.
 */
export function useCreateProtection() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: (values: ProtectionFields) =>
      callAPI<ProtectionDetail>(PATH, { metode: 'POST', body: values, token, portal }),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: keys.all })
    },
  })
}

/**
 * Hook penyuntingan permintaan proteksi yang belum tertaut klaim.
 *
 * Backend menolak penyuntingan proteksi yang sudah tertaut maupun sudah diakseptasi dengan
 * `409`. Layar memeriksanya lebih dulu lewat `dapat_disunting`, tetapi pemeriksaan itu hanya
 * menjaga pengguna dari kesalahan — yang menahan dua permintaan bersamaan adalah backend.
 */
export function useUpdateProtection() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: ({ number, values }: { number: string; values: ProtectionFields }) =>
      callAPI<ProtectionDetail>(`${PATH}/${encodeURIComponent(number)}`, {
        metode: 'PUT',
        body: values,
        token,
        portal,
      }),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: keys.all })
    },
  })
}
