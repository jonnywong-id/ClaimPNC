import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { callAPI } from '@/api/client'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import type {
  ProtectionDetail,
  ProtectionFields,
  ProtectionFilter,
  ProtectionListResponse,
  ProtectionTypeListResponse,
  ClaimLookup,
} from './types'

const PATH = '/api/input-req-protection'

/**
 * Banyaknya baris per halaman.
 *
 * Mengikuti `pyPageSize` layar lama, yang bernilai 50 pada
 * `Report Definition/InboxReqOpenProtection_RD-RD.xml`. Backend menolak permintaan di atas
 * 100.
 */
export const PAGE_SIZE = 50

/**
 * Kunci cache dikumpulkan di satu tempat supaya invalidasi tidak salah sasaran.
 *
 * PORTAL ikut menjadi bagian kunci, dan itu bukan kerapian: dua portal adalah dua badan
 * hukum dengan basis data berbeda (`ADR-0030`). Menyimpan keduanya di bawah satu kunci akan
 * membuat perpindahan portal menampilkan proteksi entitas sebelumnya — kebocoran yang
 * tampil sebagai layar normal, persis `R-20`.
 */
const keys = {
  all: ['input-req-protection'] as const,
  list: (portal: string | null, token: string | null, f: ProtectionFilter) =>
    ['input-req-protection', 'daftar', portal, token, f.search ?? '', f.offset ?? 0] as const,
  detail: (portal: string | null, token: string | null, number: string) =>
    ['input-req-protection', 'detail', portal, token, number] as const,
}

/**
 * Kunci master tipe sengaja BERADA DI LUAR `keys.all`.
 *
 * Menyimpan maupun menyunting proteksi membatalkan seluruh cache di bawah `keys.all`. Bila
 * master ikut di dalamnya, setiap penyimpanan akan menembak ulang master yang sama sekali
 * tidak berubah — membatalkan cache satu jam yang justru menjadi alasan hook-nya terpisah.
 */
const claimKeys = {
  lookup: (portal: string | null, token: string | null, nomor: string) =>
    ["input-req-protection-klaim", portal, token, nomor] as const,
}

const typeKeys = {
  list: (portal: string | null, token: string | null) =>
    ['input-req-protection-tipe', portal, token] as const,
}

function buildPath(f: ProtectionFilter): string {
  const params = new URLSearchParams()
  if (f.search?.trim()) params.set('cari', f.search.trim())
  if (f.offset) params.set('lewati', String(f.offset))
  params.set('batas', String(PAGE_SIZE))

  return `${PATH}?${params.toString()}`
}

/**
 * Hook daftar permintaan proteksi yang belum diakseptasi.
 *
 * Menggantikan `Report Definition/InboxReqOpenProtection_RD-RD.xml`, yang menyaring
 * `.AcceptStatus IS NULL` dan mengurutkan menurun.
 *
 * # Penyaringan dikerjakan SERVER
 *
 * Daftarnya hanya dibatasi "belum diakseptasi", sehingga ia tumbuh tanpa batas seiring
 * waktu. Menyaringnya di peramban berarti hasil pencarian hanya menyentuh halaman yang
 * sedang terbuka — dan itu BOHONG: pengguna diberi tahu sesuatu tidak ada padahal ia ada di
 * halaman lain.
 *
 * # Tidak dijalankan sebelum portal dipilih
 *
 * Backend memang menolak permintaan tanpa portal (`TKT-F6-002`), tetapi menembaknya lebih
 * dulu hanya untuk menerima penolakan adalah perjalanan jaringan yang sia-sia.
 */
export function useProtectionList(filter: ProtectionFilter) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: keys.list(portal, token, filter),
    queryFn: () => callAPI<ProtectionListResponse>(buildPath(filter), { token, portal }),
    enabled: token !== null && portal !== null,

    // Daftar ini berubah setiap kali ada yang membuat atau mengakseptasi proteksi. Data
    // dianggap usang seketika, tetapi tidak ditembak ulang sendiri — pengguna yang menekan
    // Muat ulang.
    staleTime: 0,
  })
}

/** Hook pembacaan satu permintaan proteksi beserta isian formnya. */
export function useProtectionDetail(number: string | null) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: keys.detail(portal, token, number ?? ''),
    queryFn: () =>
      callAPI<ProtectionDetail>(`${PATH}/${encodeURIComponent(number ?? '')}`, {
        token,
        portal,
      }),
    enabled: token !== null && portal !== null && number !== null && number !== '',
  })
}

/**
 * Hook pembacaan master tipe proteksi, untuk pilihan pada form dan penyaring pada daftar.
 *
 * # Kenapa cache-nya panjang, berbeda dari daftar proteksi
 *
 * Masternya sembilan baris yang berubah sangat jarang, sementara daftar proteksi berubah
 * setiap kali ada yang membuat atau mengakseptasi. Menyamakan keduanya berarti menembak
 * master di setiap pemuatan halaman tanpa alasan.
 *
 * Satu jam dipilih, bukan tak terbatas: master yang baru disunting harus terlihat tanpa
 * menunggu pengguna keluar-masuk aplikasi.
 *
 * # Portal ikut menjadi kunci
 *
 * Master pun tinggal di basis data tiap entitas (`ADR-0030`). Tanpa portal di dalam kunci,
 * berpindah portal akan menampilkan tipe milik entitas sebelumnya — kebocoran yang tampil
 * sebagai layar normal (`R-20`).
 */
export function useProtectionTypes() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: typeKeys.list(portal, token),
    queryFn: () => callAPI<ProtectionTypeListResponse>(`${PATH}/tipe`, { token, portal }),
    enabled: token !== null && portal !== null,
    staleTime: 60 * 60 * 1000,
  })
}

/**
 * Hook pembuatan permintaan proteksi baru.
 *
 * Nomor proteksi TIDAK dikirim: ia diterbitkan server dari sequence Oracle, dan dikembalikan
 * lewat respons. Mengirimnya dari sini akan membuat siapa pun dapat menentukan nomornya
 * sendiri.
 */
export function useCreateProtection() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: (values: ProtectionFields) =>
      callAPI<ProtectionDetail>(PATH, { metode: 'POST', body: values, token, portal }),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: keys.all })
    },
  })
}

/**
 * Hook penyuntingan permintaan proteksi yang belum tertaut klaim.
 *
 * Backend menolak penyuntingan proteksi yang sudah tertaut maupun sudah diakseptasi dengan
 * `409`. Layar memeriksanya lebih dulu lewat `dapat_disunting`, tetapi pemeriksaan itu hanya
 * menjaga pengguna dari kesalahan — yang menahan dua permintaan bersamaan adalah backend.
 */
export function useUpdateProtection() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: ({ number, values }: { number: string; values: ProtectionFields }) =>
      callAPI<ProtectionDetail>(`${PATH}/${encodeURIComponent(number)}`, {
        metode: 'PUT',
        body: values,
        token,
        portal,
      }),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: keys.all })
    },
  })
}

/**
 * Hook pencarian klaim, untuk mengisi field TURUNAN pada form.
 *
 * # Kenapa layar mencarinya, padahal server mencarinya lagi saat menyimpan
 *
 * Keduanya melayani hal berbeda. Yang ini membuat pengguna MELIHAT apa yang akan tersimpan
 * sebelum menekan Simpan — termasuk Current Date Of Loss dan Cause Of Loss Dipilih, yang
 * tidak dapat ia ketik. Pencarian saat menyimpan yang MENENTUKAN apa yang benar-benar
 * tersimpan.
 *
 * Menghapus salah satunya menghilangkan hal yang berbeda: tanpa yang ini pengguna menyimpan
 * sesuatu yang belum pernah ia lihat; tanpa yang itu nilai "sebelum" dapat dipalsukan lewat
 * permintaan yang dirakit tangan.
 *
 * # Tidak dijalankan sebelum nomornya benar-benar diketik
 *
 * `enabled` menahan pemanggilan selama nomor klaim kosong. Tanpa itu, setiap pembukaan form
 * baru menembak server dengan nomor kosong hanya untuk menerima penolakan.
 */
export function useClaimLookup(claimNumber: string) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const nomor = claimNumber.trim()

  return useQuery({
    queryKey: claimKeys.lookup(portal, token, nomor),
    queryFn: () =>
      callAPI<ClaimLookup>(`${PATH}/klaim/${encodeURIComponent(nomor)}`, { token, portal }),
    enabled: token !== null && portal !== null && nomor !== '',

    // Klaim jarang berubah selama satu sesi pengisian form, dan pengguna dapat berpindah
    // antar nomor klaim beberapa kali. Lima menit cukup untuk itu tanpa menahan perubahan
    // yang benar-benar terjadi di sisi Pega.
    staleTime: 5 * 60 * 1000,

    // Nomor klaim yang salah ketik menghasilkan 404, dan mencobanya ulang tidak akan
    // mengubah jawabannya — ia hanya menunda pesan yang perlu segera dilihat pengguna.
    retry: false,
  })
}
