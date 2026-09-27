import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { callAPI } from '@/api/client'
import type {
  CauseOfLossDetailInput,
  CauseOfLossDetailListResponse,
  CauseOfLossDetailResponse,
  CauseOfLossLookupResponse,
  CauseOfLossOptionsResponse,
} from '@/api/types'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

const ROUTE = '/api/master/detail-penyebab'

/**
 * Panjang minimum kata kunci pencarian daftar pilihan.
 *
 * Angkanya sama dengan `detailpenyebab.MinLookupKeyword` di backend. Keduanya diperiksa —
 * bukan hanya salah satu: server menjaga kontraknya, dan layar menghindari perjalanan
 * jaringan yang sudah pasti menjawab daftar kosong pada setiap ketukan huruf pertama.
 */
const MIN_KEYWORD = 2

/**
 * listKey menyertakan portal DAN token.
 *
 * Portal ikut karena itulah yang menentukan basis data mana yang menjawab (ADR-0030).
 * Tanpa itu, berpindah entitas akan menampilkan data entitas sebelumnya dari cache —
 * dan pada modul ini akibatnya lebih dari sekadar baris yang salah tempat: sebab kerugian
 * yang dapat dipilih menentukan bagaimana klaim dinilai pada badan hukum itu (R-20).
 */
function listKey(portal: string | null, token: string | null) {
  return ['detail-penyebab', portal, token] as const
}

/**
 * Hook daftar Detail Penyebab Kerugian.
 *
 * Tidak dijalankan sebelum portal dipilih. Backend memang menolak permintaan tanpa portal
 * (TKT-F6-002), tetapi menembaknya lebih dulu hanya untuk menerima penolakan akan
 * menampilkan pesan galat pada layar yang sebenarnya belum siap dibuka.
 *
 * # Ketiga penyaring milik endpoint ini TIDAK dipakai layar
 *
 * Endpoint-nya menerima `cari`, `id_master`, dan `bisnis`. Yang dipakai layar sekarang
 * adalah pencarian bawaan `DataTable`, sama seperti seluruh layar master lain — pencarian
 * kedua di kepala halaman hanya akan membingungkan.
 *
 * Perpindahan ke penyaringan sisi server dilakukan bersama paginasi sisi server
 * (`TKT-U2-001`), bukan sendirian. Panel "Cari Data" pada layar lama — yang menyaring
 * menurut induk dan lini bisnis — ikut dipertimbangkan di sana.
 */
export function useCauseOfLossDetailList() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: listKey(portal, token),
    queryFn: () => callAPI<CauseOfLossDetailListResponse>(ROUTE, { token, portal }),
    enabled: token !== null && portal !== null,
  })
}

/**
 * Hook pemuat satu baris ke form penyuntingan.
 *
 * Ia TERPISAH dari daftar, dan itu bukan pemborosan: daftar sengaja tidak memuat lini
 * bisnis — grid layar lama pun tidak menampilkannya — sehingga form tidak dapat dibuka
 * dari baris daftar saja.
 *
 * `enabled` menahannya sampai sebuah baris benar-benar dipilih.
 */
export function useCauseOfLossDetail(id: string | null) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: ['detail-penyebab', 'satu', portal, token, id] as const,
    queryFn: () =>
      callAPI<CauseOfLossDetailResponse>(`${ROUTE}/${encodeURIComponent(id ?? '')}`, {
        token,
        portal,
      }),
    enabled: token !== null && portal !== null && id !== null && id !== '',
  })
}

/**
 * Hook daftar pilihan Status Aktif.
 *
 * Isinya konstanta domain, sama di keempat portal — sehingga endpoint-nya pun tidak
 * menuntut portal. Ia tetap dibaca dari server alih-alih ditulis tangan di sini, supaya
 * kedua sisi tidak pernah berbeda pendapat tentang nilai apa yang sah.
 *
 * `staleTime: Infinity` karena isinya tidak akan berubah selama aplikasi hidup.
 */
export function useCauseOfLossOptions() {
  const token = useSession((state) => state.token)

  return useQuery({
    queryKey: ['detail-penyebab', 'pilihan'] as const,
    queryFn: () => callAPI<CauseOfLossOptionsResponse>(`${ROUTE}/pilihan`, { token }),
    enabled: token !== null,
    staleTime: Infinity,
  })
}

/**
 * Hook pencarian Master Penyebab Kerugian — isian "ID Master Kerugian".
 *
 * Kata kunci pendek tidak ditembakkan sama sekali; lihat MIN_KEYWORD.
 */
export function useCauseOfLossMasterSearch(keyword: string) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const clean = keyword.trim()

  return useQuery({
    queryKey: ['detail-penyebab', 'pilihan', 'master', portal, clean] as const,
    queryFn: () =>
      callAPI<CauseOfLossLookupResponse>(
        `${ROUTE}/pilihan/master?cari=${encodeURIComponent(clean)}`,
        { token, portal },
      ),
    enabled: token !== null && portal !== null && clean.length >= MIN_KEYWORD,
  })
}

/** Hook pencarian lini bisnis — grid "Bisnis". */
export function useCauseOfLossBusinessSearch(keyword: string) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const clean = keyword.trim()

  return useQuery({
    queryKey: ['detail-penyebab', 'pilihan', 'bisnis', portal, clean] as const,
    queryFn: () =>
      callAPI<CauseOfLossLookupResponse>(
        `${ROUTE}/pilihan/bisnis?cari=${encodeURIComponent(clean)}`,
        { token, portal },
      ),
    enabled: token !== null && portal !== null && clean.length >= MIN_KEYWORD,
  })
}

/**
 * Hook penambahan.
 *
 * Daftar dibuang dari cache setelah berhasil. Baris baru muncul di daftar yang sama —
 * tidak ada tab yang membedakannya — sehingga satu pembuangan sudah cukup.
 */
export function useCreateCauseOfLossDetail() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: (input: CauseOfLossDetailInput) =>
      callAPI<CauseOfLossDetailResponse>(ROUTE, {
        metode: 'POST',
        body: input,
        token,
        portal,
      }),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: ['detail-penyebab'] })
    },
  })
}

/**
 * Hook penyimpanan.
 *
 * `encodeURIComponent` bukan formalitas: D_COL_ID diterbitkan sequence dan bentuknya
 * angka hari ini, tetapi baris warisan dapat memuat apa saja — kolomnya bertipe teks tanpa
 * constraint yang diketahui (`R-08`). Satu garis miring di dalamnya akan mengubah bentuk
 * jalurnya.
 */
export function useSaveCauseOfLossDetail() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: CauseOfLossDetailInput }) =>
      callAPI<CauseOfLossDetailResponse>(`${ROUTE}/${encodeURIComponent(id)}`, {
        metode: 'PUT',
        body: input,
        token,
        portal,
      }),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: ['detail-penyebab'] })
    },
  })
}
