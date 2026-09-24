import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { callAPI } from '@/api/client'
import type {
  DetailDocumentTypeInput,
  DetailDocumentTypeListResponse,
  DetailDocumentTypeReferenceResponse,
  DetailDocumentTypeResponse,
} from '@/api/types'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

const ROUTE = '/api/master/detail-tipe-dokumen'
const ROUTE_REFERENCE = `${ROUTE}/pilihan`

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
  return ['daftar-detail-tipe-dokumen', portal, token] as const
}

function detailKey(id: string, portal: string | null, token: string | null) {
  return ['daftar-detail-tipe-dokumen', 'detail', id, portal, token] as const
}

function referenceKey(portal: string | null, token: string | null) {
  return ['daftar-detail-tipe-dokumen', 'pilihan', portal, token] as const
}

/**
 * Hook daftar Daftar Detail Tipe Dokumen.
 *
 * Menggantikan Report Definition `BrowseVLstDetTypeDoc_RD` yang mengisi grid layar
 * `ListDetTypeDocument-Harness`.
 *
 * Tidak dijalankan sebelum portal dipilih. Backend memang menolak permintaan tanpa portal
 * (TKT-F6-002), tetapi menembaknya lebih dulu hanya untuk menerima penolakan akan
 * menampilkan pesan galat pada layar yang sebenarnya belum siap dibuka — layar yang
 * menuntun pengguna memilih portal lebih berguna daripada pesan galat.
 *
 * Seluruh baris dimuat sekaligus, tanpa paginasi server. Isinya daftar rincian dokumen —
 * ratusan baris, bukan puluhan ribu. Layar yang datanya besar, inbox dan laporan, tidak
 * boleh mengikuti pola ini.
 */
export function useDetailDocumentTypeList() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: listKey(portal, token),
    queryFn: () => callAPI<DetailDocumentTypeListResponse>(ROUTE, { token, portal }),
    enabled: token !== null && portal !== null,
  })
}

/**
 * Hook satu baris LENGKAP dengan daftar bisnisnya.
 *
 * Dimuat terpisah, bukan diambil dari hasil daftar, karena daftar memang tidak
 * membawanya: grid hanya menampilkan tiga kolom dan tidak satu pun menyebut lini bisnis.
 * Memakai baris dari daftar akan membuat form tampak seolah seluruh lini bisnisnya sudah
 * dihapus — dan menyimpannya benar-benar menghapusnya.
 *
 * `id` null berarti tidak ada baris yang sedang dibuka — hook-nya diam.
 */
export function useDetailDocumentType(id: string | null) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: detailKey(id ?? '', portal, token),
    queryFn: () =>
      callAPI<DetailDocumentTypeResponse>(`${ROUTE}/${encodeURIComponent(id ?? '')}`, {
        token,
        portal,
      }),
    enabled: token !== null && portal !== null && id !== null,
  })
}

/**
 * Hook keempat daftar pilihan form.
 *
 * Menggantikan keempat autocomplete pada form Pega sekaligus. SATU permintaan, bukan
 * empat: keempatnya selalu dibutuhkan bersamaan, dan memisahkannya hanya menambah tiga
 * keadaan setengah-siap yang harus dijaga layar.
 *
 * MENUNTUT portal: keempat masternya hidup di basis data setiap entitas.
 *
 * Kegagalannya TIDAK menghalangi apa pun: keempat kode boleh diketik sendiri, sehingga
 * yang hilang hanya kenyamanan memilih. Yang gagal disebutkan server lewat
 * `tidak_tersedia`, sehingga layar dapat mengatakan apa yang hilang alih-alih menampilkan
 * daftar kosong yang terbaca sebagai "masternya memang kosong".
 *
 * Daftarnya jarang berubah, sehingga tidak dimuat ulang setiap kali form dibuka.
 */
export function useDetailDocumentTypeReferences() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: referenceKey(portal, token),
    queryFn: () => callAPI<DetailDocumentTypeReferenceResponse>(ROUTE_REFERENCE, { token, portal }),
    enabled: token !== null && portal !== null,
    staleTime: 60 * 60 * 1000,
  })
}

/** Hook penambahan detail tipe dokumen. */
export function useCreateDetailDocumentType() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: (input: DetailDocumentTypeInput) =>
      callAPI<DetailDocumentTypeResponse>(ROUTE, {
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
      void client.invalidateQueries({ queryKey: listKey(portal, token) })
    },
  })
}

/** Hook penyuntingan detail tipe dokumen. */
export function useUpdateDetailDocumentType() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: DetailDocumentTypeInput }) =>
      callAPI<DetailDocumentTypeResponse>(`${ROUTE}/${encodeURIComponent(id)}`, {
        metode: 'PUT',
        body: input,
        token,
        portal,
      }),
    // Daftar DAN baris yang disunting keduanya dibuang dari cache. Tanpa yang kedua,
    // membuka kembali baris yang sama akan menampilkan daftar bisnis sebelum perubahan —
    // dan pengguna akan mengira penyimpanannya gagal.
    onSuccess: (_result, variables) => {
      void client.invalidateQueries({ queryKey: listKey(portal, token) })
      void client.invalidateQueries({ queryKey: detailKey(variables.id, portal, token) })
    },
  })
}
