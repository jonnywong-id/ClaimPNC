import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { callAPI } from '@/api/client'
import type {
  TravelDocumentChoiceListResponse,
  TravelDocumentDetailInput,
  TravelDocumentDetailListResponse,
  TravelDocumentDetailResponse,
} from '@/api/types'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

const ROUTE = '/api/master/daftar-detail-dokumen-travel'
const ROUTE_DOCUMENT = '/api/master/dokumen-travel-pilihan'

/**
 * listKey menyertakan portal DAN token.
 *
 * Portal ikut di dalam kunci karena itulah yang menentukan basis data mana yang menjawab
 * (`ADR-0030`). Tanpa itu, berpindah entitas akan menampilkan data entitas sebelumnya
 * dari cache — pengguna melihat daftar yang masuk akal, dan tidak ada apa pun di layar
 * yang menandakan daftar itu milik badan hukum lain (`R-20`).
 *
 * Token ikut supaya cache pengguna sebelumnya tidak terwarisi pengguna berikutnya di
 * peramban yang sama, mengikuti pola modul master lainnya.
 */
function listKey(portal: string | null, token: string | null) {
  return ['daftar-detail-dokumen-travel', portal, token] as const
}

function detailKey(id: string, portal: string | null, token: string | null) {
  return ['daftar-detail-dokumen-travel', 'detail', id, portal, token] as const
}

function documentKey(portal: string | null, token: string | null) {
  return ['dokumen-travel-pilihan', portal, token] as const
}

/**
 * Hook daftar Daftar Detail Dokumen Travel.
 *
 * Menggantikan Report Definition `BrowseLstDocTravel_RD` yang mengisi grid layar
 * `ListDocumentTravel_Harness`.
 *
 * Tidak dijalankan sebelum portal dipilih. Backend memang menolak permintaan tanpa portal
 * (TKT-F6-002), tetapi menembaknya lebih dulu hanya untuk menerima penolakan akan
 * menampilkan pesan galat pada layar yang sebenarnya belum siap dibuka — layar yang
 * menuntun pengguna memilih portal lebih berguna daripada pesan galat.
 *
 * Seluruh baris dimuat sekaligus, tanpa paginasi server. Isinya daftar aturan dokumen —
 * puluhan baris, bukan puluhan ribu. Layar yang datanya besar, inbox dan laporan, tidak
 * boleh mengikuti pola ini.
 */
export function useTravelDocumentDetailList() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: listKey(portal, token),
    queryFn: () => callAPI<TravelDocumentDetailListResponse>(ROUTE, { token, portal }),
    enabled: token !== null && portal !== null,
  })
}

/**
 * Hook satu baris, dimuat ulang dari server.
 *
 * # Layar TIDAK memakainya untuk membuka form, dan itu disengaja
 *
 * Daftar sudah membawa SELURUH isi baris — kelima kolomnya adalah seluruh isi tabelnya.
 * Memuat ulang satu baris karena itu tidak menambah satu keterangan pun, dan form dibuka
 * langsung dari baris yang ada di daftar, sama seperti modul Master Dokumen Travel.
 *
 * Hook ini sempat dipakai ketika modul masih memuat pembatasan Plan dan Jaminan: daftar
 * memang tidak membawanya, sehingga form WAJIB memuat ulang. Pembatasan itu dicabut
 * 2026-10-03, dan bersamanya kewajiban itu hilang.
 *
 * Ia dipertahankan karena rutenya memang ada di kontrak API dan berguna untuk memuat satu
 * baris tanpa menarik seluruh daftar. Bila kelak tidak ada pemakainya sama sekali, ia
 * dibuang bersama rutenya — bukan disimpan "untuk berjaga-jaga".
 *
 * `id` null berarti tidak ada baris yang sedang dibuka — hook-nya diam.
 */
export function useTravelDocumentDetail(id: string | null) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: detailKey(id ?? '', portal, token),
    queryFn: () =>
      callAPI<TravelDocumentDetailResponse>(`${ROUTE}/${encodeURIComponent(id ?? '')}`, {
        token,
        portal,
      }),
    enabled: token !== null && portal !== null && id !== null,
  })
}

/**
 * Hook daftar pilihan untuk isian ID Dokumen.
 *
 * Menggantikan autocomplete `BrowseMstDocTravel_RD`. MENUNTUT portal:
 * POOLDATA.M_DOCTRAVEL hidup di basis data setiap entitas.
 *
 * Daftarnya jarang berubah, sehingga tidak dimuat ulang setiap kali form dibuka.
 */
export function useTravelDocumentChoiceList() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: documentKey(portal, token),
    queryFn: () => callAPI<TravelDocumentChoiceListResponse>(ROUTE_DOCUMENT, { token, portal }),
    enabled: token !== null && portal !== null,
    staleTime: 60 * 60 * 1000,
  })
}

/** Hook penambahan detail dokumen travel. */
export function useCreateTravelDocumentDetail() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: (input: TravelDocumentDetailInput) =>
      callAPI<TravelDocumentDetailResponse>(ROUTE, {
        metode: 'POST',
        body: input,
        token,
        portal,
      }),
    // Daftar dimuat ulang dari server, BUKAN ditambahi barisnya di sisi klien. ID baru
    // diterbitkan server dari urutan basis data, dan petugas lain dapat menambah baris
    // pada saat yang sama — daftar yang disusun sendiri di peramban akan berbeda dari isi
    // tabel yang sebenarnya.
    onSuccess: () => {
      client.invalidateQueries({ queryKey: listKey(portal, token) })
    },
  })
}

/** Hook penyuntingan detail dokumen travel. */
export function useUpdateTravelDocumentDetail() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: TravelDocumentDetailInput }) =>
      callAPI<TravelDocumentDetailResponse>(`${ROUTE}/${encodeURIComponent(id)}`, {
        metode: 'PUT',
        body: input,
        token,
        portal,
      }),
    // Daftar DAN baris yang disunting keduanya dibuang dari cache. Tanpa yang kedua,
    // membuka kembali baris yang sama akan menampilkan pembatasan plan sebelum perubahan
    // — dan pengguna akan mengira penyimpanannya gagal.
    onSuccess: (_result, variables) => {
      client.invalidateQueries({ queryKey: listKey(portal, token) })
      client.invalidateQueries({ queryKey: detailKey(variables.id, portal, token) })
    },
  })
}
