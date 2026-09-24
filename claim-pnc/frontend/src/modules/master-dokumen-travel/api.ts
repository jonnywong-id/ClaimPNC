import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { callAPI } from '@/api/client'
import type {
  TravelDocumentInput,
  TravelDocumentListResponse,
  TravelDocumentResponse,
} from '@/api/types'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

const ROUTE = '/api/master/dokumen-travel'

/**
 * listKey menyertakan portal DAN token.
 *
 * Portal ikut di dalam kunci karena itulah yang menentukan basis data mana yang
 * menjawab (`ADR-0030`). Tanpa itu, berpindah entitas akan menampilkan data entitas
 * sebelumnya dari cache — pengguna melihat daftar dokumen yang masuk akal, dan tidak ada
 * apa pun di layar yang menandakan daftar itu milik badan hukum lain (`R-20`).
 *
 * Token ikut supaya cache pengguna sebelumnya tidak terwarisi pengguna berikutnya di
 * peramban yang sama, mengikuti pola modul master lainnya.
 */
function listKey(portal: string | null, token: string | null) {
  return ['master-dokumen-travel', portal, token] as const
}

/**
 * Hook daftar Master Dokumen Travel.
 *
 * Menggantikan Report Definition `BrowseMstDocTravel_RD` yang mengisi grid layar
 * `BrowseMasterDocumentTravel_Harness`.
 *
 * Tidak dijalankan sebelum portal dipilih. Backend memang menolak permintaan tanpa
 * portal, tetapi menembaknya lebih dulu hanya untuk menerima penolakan akan menampilkan
 * pesan galat pada layar yang sebenarnya belum siap dibuka — layar yang menuntun
 * pengguna memilih portal lebih berguna daripada pesan galat.
 *
 * Seluruh baris dimuat sekaligus, tanpa paginasi server. Isinya daftar jenis dokumen —
 * puluhan baris, bukan puluhan ribu — dan Report Definition lama pun memuat seluruhnya
 * dengan batas `pyMaxRecords=500`. Layar yang datanya besar, inbox dan laporan, tidak
 * boleh mengikuti pola ini.
 */
export function useTravelDocumentList() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: listKey(portal, token),
    queryFn: () => callAPI<TravelDocumentListResponse>(ROUTE, { token, portal }),
    enabled: token !== null && portal !== null,
  })
}

/** Hook penambahan dokumen travel. */
export function useCreateTravelDocument() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: (input: TravelDocumentInput) =>
      callAPI<TravelDocumentResponse>(ROUTE, {
        metode: 'POST',
        body: input,
        token,
        portal,
      }),
    // Daftar dimuat ulang dari server, BUKAN ditambahi barisnya di sisi klien. DOCID
    // baru diterbitkan server dari urutan basis data, dan petugas lain dapat menambah
    // baris pada saat yang sama — daftar yang disusun sendiri di peramban akan berbeda
    // dari isi tabel yang sebenarnya.
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: listKey(portal, token) })
    },
  })
}

/** Hook penyuntingan dokumen travel. */
export function useUpdateTravelDocument() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: TravelDocumentInput }) =>
      callAPI<TravelDocumentResponse>(`${ROUTE}/${encodeURIComponent(id)}`, {
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
