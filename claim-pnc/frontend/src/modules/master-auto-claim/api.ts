import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { callAPI } from '@/api/client'
import type {
  AutoClaimBankListResponse,
  AutoClaimClientListResponse,
  AutoClaimInput,
  AutoClaimListResponse,
  AutoClaimResponse,
  AutoClaimSaveInput,
  BusinessSourceListResponse,
} from '@/api/types'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

const ROUTE = '/api/master/auto-claim'
const ROUTE_SOURCE = '/api/master/auto-claim/sumber-bisnis'
const ROUTE_CLIENT = '/api/master/auto-claim/client'
const ROUTE_BANK = '/api/master/auto-claim/bank'

/**
 * Panjang minimum kata kunci lookup.
 *
 * Harus sama dengan `masterautoclaim.MinLookupKeyword` di backend. Server tetap yang
 * berwenang — ia menjawab daftar kosong untuk kata kunci yang lebih pendek — dan angka
 * di sini hanya mencegah perjalanan jaringan yang sudah pasti kosong.
 */
export const MIN_LOOKUP_KEYWORD = 2

/**
 * listKey menyertakan portal DAN token.
 *
 * Portal ikut karena itulah yang menentukan basis data mana yang menjawab (ADR-0030).
 * Tanpa itu, berpindah entitas akan menampilkan data entitas sebelumnya dari cache —
 * pengguna melihat angka yang masuk akal, dan tidak ada apa pun di layar yang menandakan
 * data itu milik badan hukum lain (R-20).
 *
 * Penyaringnya ikut pula: keempat tab layar adalah empat kombinasi penyaring atas satu
 * endpoint, dan tanpa keduanya di kunci cache, berpindah tab akan menampilkan isi tab
 * sebelumnya sampai permintaan barunya tiba.
 */
function listKey(portal: string | null, token: string | null, status: string, committeeOnly: boolean) {
  return ['master-auto-claim', portal, token, status, committeeOnly] as const
}

function lookupKey(
  kind: 'sumber-bisnis' | 'client',
  portal: string | null,
  token: string | null,
  keyword: string,
) {
  return ['master-auto-claim-lookup', kind, portal, token, keyword] as const
}

function bankKey(portal: string | null, token: string | null) {
  return ['master-auto-claim-bank', portal, token] as const
}

/**
 * Hook daftar Master Auto Claim.
 *
 * Tidak dijalankan sebelum portal dipilih. Backend memang menolak permintaan tanpa
 * portal (TKT-F6-002), tetapi menembaknya lebih dulu hanya untuk menerima penolakan
 * akan menampilkan pesan galat pada layar yang sebenarnya belum siap dibuka.
 */
export function useAutoClaimList(status: string, committeeOnly: boolean) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  const query = new URLSearchParams({ status })
  if (committeeOnly) query.set('komite_saya', 'true')

  return useQuery({
    queryKey: listKey(portal, token, status, committeeOnly),
    queryFn: () => callAPI<AutoClaimListResponse>(`${ROUTE}?${query.toString()}`, { token, portal }),
    enabled: token !== null && portal !== null,
  })
}

/**
 * Hook daftar bank untuk dropdown Bank Penerima.
 *
 * Isinya jarang berubah, tetapi ia TETAP per portal: GENERAL.LST_BANK_GROUP dibaca dari
 * basis data entitas yang sedang dipilih, bukan dari satu tempat bersama.
 */
export function useAutoClaimBankList() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: bankKey(portal, token),
    queryFn: () => callAPI<AutoClaimBankListResponse>(ROUTE_BANK, { token, portal }),
    enabled: token !== null && portal !== null,
  })
}

/**
 * Hook lookup Sumber Bisnis.
 *
 * Tidak dijalankan untuk kata kunci yang terlalu pendek. Server pun menjawabnya dengan
 * daftar kosong, tetapi menembaknya tetap berarti satu perjalanan jaringan untuk setiap
 * huruf yang diketik pada dua huruf pertama.
 */
export function useBusinessSourceLookup(keyword: string) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const clean = keyword.trim()

  return useQuery({
    queryKey: lookupKey('sumber-bisnis', portal, token, clean),
    queryFn: () =>
      callAPI<BusinessSourceListResponse>(
        `${ROUTE_SOURCE}?cari=${encodeURIComponent(clean)}`,
        { token, portal },
      ),
    enabled: token !== null && portal !== null && clean.length >= MIN_LOOKUP_KEYWORD,
  })
}

/** Hook lookup Client. Perilakunya sama dengan lookup Sumber Bisnis. */
export function useClientLookup(keyword: string) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const clean = keyword.trim()

  return useQuery({
    queryKey: lookupKey('client', portal, token, clean),
    queryFn: () =>
      callAPI<AutoClaimClientListResponse>(
        `${ROUTE_CLIENT}?cari=${encodeURIComponent(clean)}`,
        { token, portal },
      ),
    enabled: token !== null && portal !== null && clean.length >= MIN_LOOKUP_KEYWORD,
  })
}

/**
 * Hook penambahan.
 *
 * Seluruh daftar dibuang dari cache setelah berhasil, bukan hanya daftar tab yang
 * sedang dibuka: baris baru lahir berstatus menunggu, sehingga yang berubah adalah tab
 * Waiting Approval dan Komite Approval — dua tab yang justru TIDAK sedang dilihat
 * pengguna saat ia menambah.
 */
export function useCreateAutoClaim() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: (input: AutoClaimInput) =>
      callAPI<AutoClaimResponse>(ROUTE, { metode: 'POST', body: input, token, portal }),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: ['master-auto-claim'] })
    },
  })
}

/**
 * Hook penyimpanan — sekaligus keputusan komite.
 *
 * SATU hook untuk TIGA tombol, karena di sistem lama memang satu operasi:
 * `Activity/UpdateMstAutoClaim_act` dibedakan hanya oleh parameter `stsapprove`.
 *
 * Seluruh daftar dibuang dari cache setelah berhasil. Menyetujui MEMINDAHKAN baris dari
 * satu tab ke tab lain, sehingga daftar yang tidak sedang dilihat pun sudah basi.
 */
export function useSaveAutoClaim() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: ({ inisial, input }: { inisial: string; input: AutoClaimSaveInput }) =>
      callAPI<AutoClaimResponse>(`${ROUTE}/${encodeURIComponent(inisial)}`, {
        metode: 'PUT',
        body: input,
        token,
        portal,
      }),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: ['master-auto-claim'] })
    },
  })
}
