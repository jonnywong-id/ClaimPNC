import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { callAPI } from '@/api/client'
import type {
  BusinessListResponse,
  DocumentObjectInput,
  DocumentObjectListResponse,
  DocumentObjectResponse,
} from '@/api/types'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

const ROUTE = '/api/master/objek-dokumen'

/**
 * Daftar bisnis dibaca dari rute yang SAMA dengan modul Master COL Simas Online.
 *
 * Bukan duplikasi yang terlewat, melainkan yang dikehendaki: POOLDATA.BUSINESS adalah data
 * acuan bersama milik GISFW, bukan milik satu layar. Backend pun hanya mendaftarkannya
 * sekali — chi akan panik saat start bila dua modul mendaftarkan jalur yang sama.
 *
 * Yang TIDAK dilakukan: mengimpor hook milik modul tetangga. Aturan struktur frontend
 * melarang satu fitur mengimpor dari fitur lain, dan melanggarnya akan membuat kedua layar
 * gagal bersamaan bila salah satunya dipindahkan.
 */
const ROUTE_BUSINESS = '/api/master/bisnis'

/**
 * listKey menyertakan portal DAN token.
 *
 * Portal ikut di dalam kunci karena itulah yang menentukan basis data mana yang menjawab
 * (`ADR-0030`). Tanpa itu, berpindah entitas akan menampilkan data entitas sebelumnya dari
 * cache — pengguna melihat angka yang masuk akal, dan tidak ada apa pun di layar yang
 * menandakan data itu milik badan hukum lain (`R-20`).
 *
 * Token ikut supaya cache pengguna sebelumnya tidak terwarisi pengguna berikutnya di
 * peramban yang sama, mengikuti pola usePortalList.
 */
function listKey(portal: string | null, token: string | null) {
  return ['daftar-objek-dokumen', portal, token] as const
}

function detailKey(id: string, portal: string | null, token: string | null) {
  return ['daftar-objek-dokumen', 'detail', id, portal, token] as const
}

/**
 * businessKey sengaja SAMA PERSIS dengan kunci milik modul Master COL Simas Online.
 *
 * Dengan begitu kedua layar berbagi satu entri cache: membuka layar kedua tidak menembak
 * ulang daftar yang sama, dan keduanya tidak mungkin menampilkan master bisnis yang
 * berbeda. Kunci yang berbeda untuk data yang sama justru yang akan menimbulkan selisih itu.
 */
function businessKey(portal: string | null, token: string | null) {
  return ['master-bisnis', portal, token] as const
}

/**
 * Hook daftar objek dokumen.
 *
 * Tidak dijalankan sebelum portal dipilih. Backend memang menolak permintaan tanpa portal
 * (TKT-F6-002), tetapi menembaknya lebih dulu hanya untuk menerima penolakan akan
 * menampilkan pesan galat pada layar yang sebenarnya belum siap dibuka — layar yang
 * menuntun pengguna memilih portal lebih berguna daripada pesan galat.
 */
export function useDocumentObjectList() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: listKey(portal, token),
    queryFn: () => callAPI<DocumentObjectListResponse>(ROUTE, { token, portal }),
    enabled: token !== null && portal !== null,
  })
}

/**
 * Hook satu baris LENGKAP dengan pemetaan bisnisnya.
 *
 * Dimuat terpisah, bukan diambil dari hasil daftar, karena daftar memang tidak membawanya:
 * grid hanya menampilkan ID dan keterangannya, dan menarik pemetaan seluruh baris berarti
 * satu kueri yang hasilnya tidak pernah dilihat siapa pun.
 *
 * `id` null berarti tidak ada baris yang sedang dibuka — hook-nya diam.
 */
export function useDocumentObject(id: string | null) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: detailKey(id ?? '', portal, token),
    queryFn: () =>
      callAPI<DocumentObjectResponse>(`${ROUTE}/${encodeURIComponent(id ?? '')}`, { token, portal }),
    enabled: token !== null && portal !== null && id !== null,
  })
}

/**
 * Hook daftar bisnis untuk saran isian Bisnis.
 *
 * MENUNTUT portal: POOLDATA.BUSINESS hidup di basis data setiap entitas, sehingga "bisnis
 * milik siapa" ditentukan portal yang aktif.
 *
 * Daftarnya jarang berubah, sehingga tidak dimuat ulang setiap kali form dibuka.
 */
export function useBusinessList() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: businessKey(portal, token),
    queryFn: () => callAPI<BusinessListResponse>(ROUTE_BUSINESS, { token, portal }),
    enabled: token !== null && portal !== null,
    staleTime: 60 * 60 * 1000,
  })
}

/** Hook penambahan objek dokumen. */
export function useCreateDocumentObject() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: (input: DocumentObjectInput) =>
      callAPI<DocumentObjectResponse>(ROUTE, {
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

/** Hook penyuntingan objek dokumen. */
export function useUpdateDocumentObject() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: DocumentObjectInput }) =>
      callAPI<DocumentObjectResponse>(`${ROUTE}/${encodeURIComponent(id)}`, {
        metode: 'PUT',
        body: input,
        token,
        portal,
      }),
    // Daftar DAN baris yang disunting keduanya dibuang dari cache. Tanpa yang kedua,
    // membuka kembali baris yang sama akan menampilkan pemetaan bisnis sebelum perubahan —
    // dan pengguna akan mengira penyimpanannya gagal.
    onSuccess: (_result, variables) => {
      void client.invalidateQueries({ queryKey: listKey(portal, token) })
      void client.invalidateQueries({ queryKey: detailKey(variables.id, portal, token) })
    },
  })
}
