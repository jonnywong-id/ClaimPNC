import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { callAPI } from '@/api/client'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import type {
  DaftarResponse,
  DetailRCL,
  KeputusanRCL,
  KeputusanResponse,
  KeteranganResponse,
} from './types'

const PATH = '/api/inbox-rcl'

/** Banyaknya baris per halaman. Backend menolak permintaan di atas 100. */
export const PAGE_SIZE = 25

/**
 * Kunci cache. PORTAL dan TOKEN ikut menjadi bagian kunci: dua portal adalah dua badan
 * hukum (`ADR-0030`, `R-20`), dan antreannya disaring dengan identitas pemanggil — cache
 * yang bertahan melewati pergantian pengguna akan menampilkan tugas dokter sebelumnya.
 */
const keys = {
  keterangan: (portal: string | null, token: string | null) =>
    ['inbox-rcl', 'keterangan', portal, token] as const,

  daftar: (portal: string | null, token: string | null, cari: string, lewati: number) =>
    ['inbox-rcl', 'daftar', portal, token, cari, lewati] as const,

  detail: (portal: string | null, token: string | null, nomor: string) =>
    ['inbox-rcl', 'detail', portal, token, nomor] as const,
}

/**
 * Hook keterangan layar. Judul kolom datang dari server karena ia hasil pembacaan
 * `Harness/RCL_Harness-Harness.xml` yang tercatat di backend.
 */
export function useKeteranganRCL() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: keys.keterangan(portal, token),
    queryFn: () => callAPI<KeteranganResponse>(`${PATH}/keterangan`, { token, portal }),
    enabled: token !== null && portal !== null,
    staleTime: Infinity,
    gcTime: Infinity,
  })
}

/**
 * Hook antrean RCL Dokter milik pengguna yang sedang masuk.
 *
 * Menggantikan `Report Definition/InboxRCLDokter_RD-RD.xml`. Layar tidak mengirim identitas
 * apa pun — server membacanya dari sesi lalu mencari identitas LAMA-nya sendiri.
 */
export function useDaftarRCL(cari: string, lewati: number) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  const params = new URLSearchParams()
  if (cari.trim()) params.set('cari', cari.trim())
  if (lewati > 0) params.set('lewati', String(lewati))
  params.set('batas', String(PAGE_SIZE))

  return useQuery({
    queryKey: keys.daftar(portal, token, cari.trim(), lewati),
    queryFn: () => callAPI<DaftarResponse>(`${PATH}?${params.toString()}`, { token, portal }),
    enabled: token !== null && portal !== null,
    staleTime: 0,
    placeholderData: (previous) => previous,
  })
}

/**
 * Hook layar kerja `RCLDokter` satu klaim — dibuka saat Nomor Case diklik.
 *
 * Server hanya membuka klaim yang ada di antrean pemanggil; klaim lain dijawab 404.
 */
export function useDetailRCL(nomor: string | null) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: keys.detail(portal, token, nomor ?? ''),
    queryFn: () =>
      callAPI<DetailRCL>(`${PATH}/klaim/${encodeURIComponent(nomor ?? '')}`, { token, portal }),
    enabled: token !== null && portal !== null && !!nomor,
    staleTime: 0,
    retry: false,
  })
}

/**
 * Hook keputusan dokter RCL — padanan activity `SendToPUCL`.
 *
 * Sesudah berhasil, antrean dan layar kerja klaim ini dibuang dari cache: klaimnya sudah
 * berpindah ke tahap lain dan tidak boleh tampil lagi di antrean dokter.
 */
export function useKeputusanRCL(nomor: string) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: (input: { keputusan: KeputusanRCL; alasanDokter?: string }) =>
      callAPI<KeputusanResponse>(`${PATH}/klaim/${encodeURIComponent(nomor)}/keputusan`, {
        metode: 'POST',
        body: { keputusan: input.keputusan, alasan_dokter: input.alasanDokter ?? '' },
        token,
        portal,
      }),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: ['inbox-rcl', 'daftar'] })
      client.removeQueries({ queryKey: ['inbox-rcl', 'detail'] })

      // Klaimnya BERPINDAH ANTREAN, bukan sekadar hilang dari antrean dokter: Setuju dan
      // Submit membukanya di workbasket `RCLPUCL` (tahap `rcl-pucl`), Tidak Setuju dan Back
      // mengembalikannya ke PIC Teknik. Tanpa baris ini, cache daftar Inbox RCL/PUCL yang
      // sudah terbuka di sesi yang sama masih memuat daftar LAMA saat penggunanya pindah ke
      // sana — klaim yang baru disetujui terbaca belum sampai.
      void client.invalidateQueries({ queryKey: ['inbox-rcl-pucl'] })
    },
  })
}
