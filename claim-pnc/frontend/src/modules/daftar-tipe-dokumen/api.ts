import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { callAPI } from '@/api/client'
import type {
  DocumentTypeInput,
  DocumentTypeListResponse,
  DocumentTypeResponse,
} from '@/api/types'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

const ROUTE = '/api/master/tipe-dokumen'

/**
 * listKey menyertakan portal DAN token.
 *
 * Portal ikut di dalam kunci karena itulah yang menentukan basis data mana yang menjawab
 * (`ADR-0030`). Tanpa itu, berpindah entitas akan menampilkan data entitas sebelumnya dari
 * cache — pengguna melihat daftar tipe dokumen yang masuk akal, dan tidak ada apa pun di
 * layar yang menandakan daftar itu milik badan hukum lain (`R-20`).
 *
 * Token ikut supaya cache pengguna sebelumnya tidak terwarisi pengguna berikutnya di
 * peramban yang sama, mengikuti pola modul master lainnya.
 */
function listKey(portal: string | null, token: string | null) {
  return ['daftar-tipe-dokumen', portal, token] as const
}

/**
 * Hook daftar Tipe Dokumen.
 *
 * Menggantikan Report Definition `BrowseLstDocType_RD` yang mengisi grid layar
 * `ListDocumentTypeInbox`.
 *
 * Tidak dijalankan sebelum portal dipilih. Backend memang menolak permintaan tanpa portal,
 * tetapi menembaknya lebih dulu hanya untuk menerima penolakan akan menampilkan pesan galat
 * pada layar yang sebenarnya belum siap dibuka — layar yang menuntun pengguna memilih
 * portal lebih berguna daripada pesan galat.
 *
 * Seluruh baris dimuat sekaligus, tanpa paginasi server. Isinya daftar kategori dokumen —
 * puluhan baris, bukan puluhan ribu — dan Report Definition lama pun memuat seluruhnya
 * dengan batas `pyMaxRecords=500`. Layar yang datanya besar, inbox dan laporan, tidak boleh
 * mengikuti pola ini.
 */
export function useDocumentTypeList() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: listKey(portal, token),
    queryFn: () => callAPI<DocumentTypeListResponse>(ROUTE, { token, portal }),
    enabled: token !== null && portal !== null,
  })
}

/** Hook penambahan tipe dokumen. */
export function useCreateDocumentType() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: (input: DocumentTypeInput) =>
      callAPI<DocumentTypeResponse>(ROUTE, {
        metode: 'POST',
        body: input,
        token,
        portal,
      }),
    // Daftar dimuat ulang dari server, BUKAN ditambahi barisnya di sisi klien. ID baru
    // diterbitkan server dari urutan basis data, dan petugas lain dapat menambah baris pada
    // saat yang sama — daftar yang disusun sendiri di peramban akan berbeda dari isi tabel
    // yang sebenarnya.
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: listKey(portal, token) })
    },
  })
}

/** Hook penyuntingan tipe dokumen. */
export function useUpdateDocumentType() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: DocumentTypeInput }) =>
      callAPI<DocumentTypeResponse>(`${ROUTE}/${encodeURIComponent(id)}`, {
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
