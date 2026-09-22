import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { callAPI } from '@/api/client'
import type {
  Clause,
  ClauseBusinessListResponse,
  ClauseCategoryListResponse,
  ClauseDeleteResponse,
  ClauseInput,
  ClauseListResponse,
  ClauseResponse,
} from '@/api/types'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

const ROUTE = '/api/master/pasal-kerugian'
const ROUTE_CATEGORY = '/api/master/pasal-kerugian/kategori'
const ROUTE_BUSINESS = '/api/master/pasal-kerugian/bisnis'

/**
 * Panjang minimum kata kunci pencarian lini bisnis.
 *
 * Harus sama dengan `masterpasal.MinLookupKeyword` di backend. Server tetap yang
 * berwenang — ia menjawab daftar kosong untuk kata kunci yang lebih pendek — dan angka di
 * sini hanya mencegah perjalanan jaringan yang sudah pasti kosong.
 */
export const MIN_LOOKUP_KEYWORD = 2

/**
 * listKey menyertakan portal DAN token.
 *
 * Portal ikut karena itulah yang menentukan basis data mana yang menjawab (ADR-0030).
 * Tanpa itu, berpindah entitas akan menampilkan data entitas sebelumnya dari cache —
 * pengguna melihat angka yang masuk akal, dan tidak ada apa pun di layar yang menandakan
 * data itu milik badan hukum lain (R-20).
 */
function listKey(portal: string | null, token: string | null) {
  return ['master-pasal-kerugian', portal, token] as const
}

function detailKey(portal: string | null, token: string | null, id: string) {
  return ['master-pasal-kerugian-detail', portal, token, id] as const
}

function businessKey(portal: string | null, token: string | null, keyword: string) {
  return ['master-pasal-kerugian-bisnis', portal, token, keyword] as const
}

/**
 * Hook daftar pasal kerugian.
 *
 * Tidak dijalankan sebelum portal dipilih. Backend memang menolak permintaan tanpa portal
 * (TKT-F6-002), tetapi menembaknya lebih dulu hanya untuk menerima penolakan akan
 * menampilkan pesan galat pada layar yang sebenarnya belum siap dibuka.
 */
export function useClauseList() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: listKey(portal, token),
    queryFn: () => callAPI<ClauseListResponse>(ROUTE, { token, portal }),
    enabled: token !== null && portal !== null,
  })
}

/**
 * Hook satu pasal, dipakai saat form penyuntingan dibuka.
 *
 * Ia perlu ada — daftar tidak cukup — karena daftar TIDAK memuat lini bisnis, dan nama
 * lini bisnisnya disegarkan server dari master yang berlaku sekarang. Itu persis yang
 * dilakukan tombol "Ubah" pada layar lama, yang memanggil
 * `PNCGetListPasalDataCOL_Act(idstatusp=<IDDATA>)` alih-alih memakai baris grid yang sudah
 * ada di layar.
 *
 * `id` kosong berarti form sedang dalam mode tambah; permintaannya tidak dijalankan.
 */
export function useClause(id: string) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: detailKey(portal, token, id),
    queryFn: () => callAPI<ClauseResponse>(`${ROUTE}/${encodeURIComponent(id)}`, { token, portal }),
    enabled: token !== null && portal !== null && id !== '',
  })
}

/**
 * Hook daftar pilihan Kategori.
 *
 * Ia TIDAK menyertakan portal di kunci cache-nya, karena isinya memang tidak bergantung
 * pada entitas — ketiganya milik aplikasi. Token tetap ikut: daftarnya berada di balik
 * sesi, sehingga pengguna berikutnya tidak memakai jawaban milik sesi sebelumnya.
 *
 * `staleTime` panjang karena isinya tidak berubah selama aplikasi berjalan.
 */
export function useClauseCategoryList() {
  const token = useSession((state) => state.token)

  return useQuery({
    queryKey: ['master-pasal-kerugian-kategori', token] as const,
    queryFn: () => callAPI<ClauseCategoryListResponse>(ROUTE_CATEGORY, { token }),
    enabled: token !== null,
    staleTime: Infinity,
  })
}

/**
 * Hook pencarian lini bisnis.
 *
 * Tidak dijalankan untuk kata kunci yang terlalu pendek. Server pun menjawabnya dengan
 * daftar kosong, tetapi menembaknya tetap berarti satu perjalanan jaringan untuk setiap
 * huruf yang diketik pada dua huruf pertama.
 */
export function useClauseBusinessLookup(keyword: string) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const clean = keyword.trim()

  return useQuery({
    queryKey: businessKey(portal, token, clean),
    queryFn: () =>
      callAPI<ClauseBusinessListResponse>(
        `${ROUTE_BUSINESS}?cari=${encodeURIComponent(clean)}`,
        { token, portal },
      ),
    enabled: token !== null && portal !== null && clean.length >= MIN_LOOKUP_KEYWORD,
  })
}

/** Hook penambahan. */
export function useCreateClause() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: (input: ClauseInput) =>
      callAPI<ClauseResponse>(ROUTE, { metode: 'POST', body: input, token, portal }),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: ['master-pasal-kerugian'] })
    },
  })
}

/**
 * Hook penyuntingan.
 *
 * Kedua kunci cache dibuang — daftar DAN detail baris itu — karena keduanya sudah basi
 * setelah penyimpanan. Tanpa membuang detailnya, membuka kembali baris yang sama akan
 * menampilkan isian sebelum perubahan sampai cache-nya kedaluwarsa sendiri.
 */
export function useUpdateClause() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: ClauseInput }) =>
      callAPI<ClauseResponse>(`${ROUTE}/${encodeURIComponent(id)}`, {
        metode: 'PUT',
        body: input,
        token,
        portal,
      }),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: ['master-pasal-kerugian'] })
      void client.invalidateQueries({ queryKey: ['master-pasal-kerugian-detail'] })
    },
  })
}

/**
 * Hook penghapusan.
 *
 * # Ia PERMANEN
 *
 * `D-66` menetapkan soft delete menyeluruh, tetapi POOLDATA.V_M_DATA_PASAL tidak punya
 * kolom penanda terhapus dan Work Owner memilih "jalankan as is" pada 2026-09-19 — yaitu
 * `DELETE` fisik seperti Pega. Barisnya tidak dapat dipulihkan, dan tabelnya juga tidak
 * punya kolom pencatat siapa dan kapan.
 *
 * Konfirmasinya ada di LAYAR, bukan di hook ini: hook tidak tahu apa yang sedang dilihat
 * pengguna, dan pertanyaan "yakin?" hanya bermakna di depan barisnya.
 */
export function useDeleteClause() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: (id: string) =>
      callAPI<ClauseDeleteResponse>(`${ROUTE}/${encodeURIComponent(id)}`, {
        metode: 'DELETE',
        token,
        portal,
      }),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: ['master-pasal-kerugian'] })
      void client.invalidateQueries({ queryKey: ['master-pasal-kerugian-detail'] })
    },
  })
}

/** Bentuk isian awal form dari sebuah baris; dipakai ClauseForm dan uji layar. */
export function inputFrom(clause: Clause): ClauseInput {
  return {
    no_pasal: clause.no_pasal,
    isi_pasal: clause.isi_pasal,
    deskripsi: clause.deskripsi,
    kategori: clause.kategori,
    bisnis: clause.bisnis,
  }
}
