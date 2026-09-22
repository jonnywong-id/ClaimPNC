import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { callAPI } from '@/api/client'
import type {
  SupplierBankListResponse,
  SupplierBranchListResponse,
  SupplierCityListResponse,
  SupplierCodeListResponse,
  SupplierCountryListResponse,
  SupplierInput,
  SupplierListResponse,
  SupplierResponse,
} from '@/api/types'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

const ROUTE = '/api/master/supplier'
const ROUTE_BRANCH = '/api/master/supplier/cabang'
const ROUTE_CITY = '/api/master/supplier/kota'
const ROUTE_COUNTRY = '/api/master/supplier/negara'
const ROUTE_BANK = '/api/master/supplier/bank'
const ROUTE_CODE = '/api/master/supplier/sandi'

/**
 * Panjang minimum kata kunci lookup Kota.
 *
 * Harus sama dengan `mastersupplier.MinLookupKeyword` di backend. Server tetap yang
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
  return ['master-supplier', portal, token] as const
}

function branchKey(portal: string | null, token: string | null) {
  return ['master-supplier-cabang', portal, token] as const
}

function cityKey(portal: string | null, token: string | null, keyword: string) {
  return ['master-supplier-kota', portal, token, keyword] as const
}

function countryKey(portal: string | null, token: string | null) {
  return ['master-supplier-negara', portal, token] as const
}

function bankKey(portal: string | null, token: string | null) {
  return ['master-supplier-bank', portal, token] as const
}

function codeKey(portal: string | null, token: string | null) {
  return ['master-supplier-sandi', portal, token] as const
}

/**
 * Hook daftar Master Supplier.
 *
 * Tidak dijalankan sebelum portal dipilih. Backend memang menolak permintaan tanpa portal
 * (TKT-F6-002), tetapi menembaknya lebih dulu hanya untuk menerima penolakan akan
 * menampilkan pesan galat pada layar yang sebenarnya belum siap dibuka.
 *
 * # Penyaring `cari` milik endpoint ini TIDAK dipakai layar
 *
 * Endpoint-nya menerimanya, dan ia memang ditambahkan karena layar lama tidak punya
 * penyaring apa pun. Yang dipakai layar sekarang adalah pencarian bawaan `DataTable`, sama
 * seperti seluruh layar master lain — pencarian kedua di kepala halaman hanya akan
 * membingungkan.
 *
 * Perpindahan ke penyaringan sisi server dilakukan bersama paginasi sisi server
 * (`TKT-U2-001`), bukan sendirian: keduanya mengubah hal yang sama dan memecahnya berarti
 * DataTable menyaring sebagian data yang sudah tersaring server.
 */
export function useSupplierList() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: listKey(portal, token),
    queryFn: () => callAPI<SupplierListResponse>(ROUTE, { token, portal }),
    enabled: token !== null && portal !== null,
  })
}

/**
 * Hook daftar cabang untuk dropdown Cabang.
 *
 * Isinya jarang berubah, tetapi ia TETAP per portal: daftar cabang dibaca dari basis data
 * entitas yang sedang dipilih, bukan dari satu tempat bersama.
 */
export function useSupplierBranchList() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: branchKey(portal, token),
    queryFn: () => callAPI<SupplierBranchListResponse>(ROUTE_BRANCH, { token, portal }),
    enabled: token !== null && portal !== null,
  })
}

/** Hook daftar negara untuk isian Negara. */
export function useSupplierCountryList() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: countryKey(portal, token),
    queryFn: () => callAPI<SupplierCountryListResponse>(ROUTE_COUNTRY, { token, portal }),
    enabled: token !== null && portal !== null,
  })
}

/** Hook daftar bank untuk dropdown Bank. */
export function useSupplierBankList() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: bankKey(portal, token),
    queryFn: () => callAPI<SupplierBankListResponse>(ROUTE_BANK, { token, portal }),
    enabled: token !== null && portal !== null,
  })
}

/**
 * Hook kelima daftar dropdown bersandi.
 *
 * SATU permintaan untuk lima daftar, bukan lima. Kelimanya dibaca server dari tabel yang
 * sama dalam satu kali pemindaian, dan memecahnya berarti lima perjalanan jaringan untuk
 * mengisi satu form.
 */
export function useSupplierCodeList() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: codeKey(portal, token),
    queryFn: () => callAPI<SupplierCodeListResponse>(ROUTE_CODE, { token, portal }),
    enabled: token !== null && portal !== null,
  })
}

/**
 * Hook lookup Kota.
 *
 * Berbeda dari Cabang, Negara, dan Bank yang dimuat penuh, Kota MEMAKAI kata kunci:
 * tabelnya berbaris ribuan, dan memuatnya penuh ke peramban untuk daftar yang hanya akan
 * dipilih satu adalah biaya yang tidak perlu dibayar berulang kali.
 *
 * Tidak dijalankan untuk kata kunci yang terlalu pendek. Server pun menjawabnya dengan
 * daftar kosong, tetapi menembaknya tetap berarti satu perjalanan jaringan untuk setiap
 * huruf yang diketik pada dua huruf pertama.
 */
export function useSupplierCityLookup(keyword: string) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const clean = keyword.trim()

  return useQuery({
    queryKey: cityKey(portal, token, clean),
    queryFn: () =>
      callAPI<SupplierCityListResponse>(`${ROUTE_CITY}?cari=${encodeURIComponent(clean)}`, {
        token,
        portal,
      }),
    enabled: token !== null && portal !== null && clean.length >= MIN_LOOKUP_KEYWORD,
  })
}

/**
 * Hook penambahan.
 *
 * Daftar sandi ikut dibuang dari cache, bukan hanya daftar supplier: sandi yang baru saja
 * dipakai pertama kalinya harus muncul sebagai pilihan pada form berikutnya. Tanpa itu,
 * petugas yang baru memasukkan jenis supplier baru tidak akan menemukannya lagi.
 */
export function useCreateSupplier() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: (input: SupplierInput) =>
      callAPI<SupplierResponse>(ROUTE, { metode: 'POST', body: input, token, portal }),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: ['master-supplier'] })
      void client.invalidateQueries({ queryKey: ['master-supplier-sandi'] })
    },
  })
}

/**
 * Hook penyimpanan.
 *
 * Daftar dibuang dari cache setelah berhasil. Menyimpan dapat mengubah status aktif yang
 * BERLAKU — `EditMasterSupplier_post` step 7 menyalin STS_AKTIF_PROMLIST ke STS_AKTIF —
 * sehingga baris di daftar berubah, bukan hanya isi formnya.
 */
export function useSaveSupplier() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: SupplierInput }) =>
      callAPI<SupplierResponse>(`${ROUTE}/${encodeURIComponent(id)}`, {
        metode: 'PUT',
        body: input,
        token,
        portal,
      }),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: ['master-supplier'] })
      void client.invalidateQueries({ queryKey: ['master-supplier-sandi'] })
    },
  })
}
