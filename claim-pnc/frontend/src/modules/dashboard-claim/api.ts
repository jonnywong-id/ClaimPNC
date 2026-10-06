import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { APIError, callAPI, simpanBerkas, unduhBerkas } from '@/api/client'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import type {
  HasilPermintaanKlaim,
  IzinPermintaanKlaim,
  JenisPermintaanKlaim,
  PenyaringDashboard,
  PermintaanTransfer,
  PenyaringResponse,
  RingkasanResponse,
  TelusurResponse,
  TampunganResponse,
  TransferResponse,
  Tile,
} from './types'

const PATH = '/api/dashboard-claim'

/** Banyaknya baris per halaman. Backend menolak permintaan di atas 100. */
export const PAGE_SIZE = 25

/**
 * Kunci cache dikumpulkan di satu tempat supaya invalidasi tidak salah sasaran.
 *
 * PORTAL ikut menjadi bagian kunci, dan itu bukan kerapian: dua portal adalah dua badan
 * hukum dengan basis data berbeda (`ADR-0030`). Menyimpan keduanya di bawah satu kunci akan
 * membuat perpindahan portal menampilkan angka entitas sebelumnya — kebocoran yang tampil
 * sebagai layar normal, persis `R-20`.
 *
 * Pada layar INI akibatnya khas: yang salah bukan satu baris melainkan seluruh ringkasannya,
 * dan angka yang salah entitas tidak terlihat keliru dengan cara lain apa pun.
 */
const keys = {
  penyaring: (portal: string | null, token: string | null) =>
    ['dashboard-claim', 'penyaring', portal, token] as const,

  ringkasan: (portal: string | null, token: string | null, f: PenyaringDashboard) =>
    ['dashboard-claim', 'ringkasan', portal, token, f.lini_bisnis ?? '', f.cari ?? ''] as const,

  telusur: (portal: string | null, token: string | null, tile: Tile, f: PenyaringDashboard) =>
    [
      'dashboard-claim',
      'telusur',
      portal,
      token,
      tile,
      f.lini_bisnis ?? '',
      f.cari ?? '',
      f.halaman ?? 1,
    ] as const,
}

/** Menyusun query string dari penyaring; yang kosong tidak dikirim sama sekali. */
function buildParams(f: PenyaringDashboard): URLSearchParams {
  const params = new URLSearchParams()
  if (f.lini_bisnis?.trim()) params.set('lini_bisnis', f.lini_bisnis.trim())
  if (f.cari?.trim()) params.set('cari', f.cari.trim())
  return params
}

/**
 * Hook bentuk layar — isi dropdown lini bisnis dan daftar kartu.
 *
 * # Kenapa bentuk penyaring datang dari server
 *
 * Karena kelima pilihan lini bisnis adalah HASIL PEMBACAAN activity Pega
 * (`Activity/SetDashboardClaim-Act.xml`), dan tempat pembacaan itu tercatat adalah backend.
 * Menyalinnya ke layar berarti daftar yang sama hidup di dua tempat, dan yang satu akan
 * tertinggal saat yang lain diperbaiki.
 *
 * Di layar ini akibatnya nyata: nilainya dibandingkan server sebagai TEKS, sehingga satu
 * ejaan yang berbeda menghasilkan penyaring yang tidak pernah cocok — dan backend
 * menolaknya dengan 422, bukan diam-diam menampilkan semuanya.
 */
export function usePenyaringDashboard() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: keys.penyaring(portal, token),
    queryFn: () => callAPI<PenyaringResponse>(`${PATH}/penyaring`, { token, portal }),
    enabled: token !== null && portal !== null,

    // Bentuk layar tidak berubah selama aplikasi berjalan: ia dibaca dari kode, bukan dari
    // data. Mengambilnya ulang tiap kali penyaring berubah hanya menambah perjalanan
    // jaringan tanpa satu pun manfaat.
    staleTime: Infinity,
    gcTime: Infinity,
  })
}

/**
 * Hook keempat angka kartu.
 *
 * Menggantikan keempat kueri hitung sistem lama, yang di sana dijalankan satu per satu oleh
 * `SetDashboardClaim` menurut `param.tipe`.
 *
 * # Satu permintaan untuk keempatnya
 *
 * Layar membaca keempat angka sekaligus, bukan satu per kartu. Memecahnya menjadi empat
 * permintaan akan membuat keempat angka datang pada saat yang berbeda — dan pada layar
 * manajerial, empat angka yang tidak sezaman lebih buruk daripada empat angka yang datang
 * sedikit lebih lambat.
 *
 * # Tidak dijalankan sebelum portal dipilih
 *
 * Backend memang menolak permintaan tanpa portal (`TKT-F6-002`), tetapi menembaknya lebih
 * dulu hanya untuk menerima penolakan adalah perjalanan jaringan yang sia-sia.
 */
export function useRingkasanDashboard(filter: PenyaringDashboard) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  const params = buildParams(filter)

  return useQuery({
    queryKey: keys.ringkasan(portal, token, filter),
    queryFn: () => {
      const query = params.toString()
      return callAPI<RingkasanResponse>(query ? `${PATH}/ringkasan?${query}` : `${PATH}/ringkasan`, {
        token,
        portal,
      })
    },
    enabled: token !== null && portal !== null,

    // Angkanya berubah setiap kali sebuah klaim berpindah tahap. Dianggap usang seketika,
    // tetapi tidak ditembak ulang sendiri — pengguna yang menekan Muat ulang.
    staleTime: 0,
  })
}

/**
 * Hook telusur satu tile.
 *
 * `enabled` ikut menunggu `tile` terisi: layar baru menembaknya SETELAH sebuah kartu
 * dipilih, bukan saat dibuka. Tanpa itu, membuka layar akan menarik daftar yang belum tentu
 * dilihat siapa pun.
 */
export function useTelusurDashboard(tile: Tile | null, filter: PenyaringDashboard) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  const params = buildParams(filter)
  params.set('halaman', String(filter.halaman ?? 1))
  params.set('ukuran', String(PAGE_SIZE))

  return useQuery({
    queryKey: keys.telusur(portal, token, tile ?? 'outstanding', filter),
    queryFn: () =>
      callAPI<TelusurResponse>(`${PATH}/${tile}?${params.toString()}`, { token, portal }),
    enabled: token !== null && portal !== null && tile !== null,

    // Halaman sebelumnya dipertahankan selama halaman berikutnya dimuat, sehingga tabel
    // tidak berkedip kosong setiap kali pengguna berpindah halaman.
    placeholderData: (previous) => previous,
    staleTime: 0,
  })
}

/**
 * Hook tab **Inbox Tampungan PIC**.
 *
 * Ia TIDAK mengirim penyaring lini bisnis, dan itu bukan kelalaian: kueri lamanya tidak
 * punya penandanya, dan layar lamanya tidak menggambar dropdown Bisnis pada tab ini.
 *
 * `enabled` ikut menunggu tab itu benar-benar dibuka, supaya membuka layar tidak menarik
 * daftar yang belum tentu dilihat siapa pun.
 */
export function useTampunganPIC(aktif: boolean, filter: PenyaringDashboard) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  const params = new URLSearchParams()
  if (filter.cari?.trim()) params.set('cari', filter.cari.trim())
  params.set('halaman', String(filter.halaman ?? 1))
  params.set('ukuran', String(PAGE_SIZE))

  return useQuery({
    queryKey: [
      'dashboard-claim',
      'tampungan',
      portal,
      token,
      filter.cari ?? '',
      filter.halaman ?? 1,
    ] as const,
    queryFn: () =>
      callAPI<TampunganResponse>(`${PATH}/tampungan?${params.toString()}`, { token, portal }),
    enabled: token !== null && portal !== null && aktif,
    placeholderData: (previous) => previous,
    staleTime: 0,
  })
}

/**
 * Mengunduh satu tile sebagai CSV.
 *
 * Menggantikan tombol "Export to Excel" pada grid layar lama. Bentuknya CSV, bukan XLSX:
 * `D-11` menetapkan pembuatan berkas dikerjakan sendiri, dan CSV terbaca Excel tanpa pustaka
 * tambahan sekaligus dapat dialirkan baris demi baris tanpa menahan memori.
 *
 * Bukan `<a download>` biasa karena permintaannya butuh header `Authorization` dan
 * `X-Portal` — tanpa keduanya unduhan dijawab "sesi tidak sah", atau lebih buruk: dilayani
 * entitas yang salah.
 */
export async function unduhTile(
  tile: Tile,
  filter: PenyaringDashboard,
  token: string | null,
  portal: string | null,
): Promise<void> {
  const params = buildParams(filter)
  const query = params.toString()
  const path = `${PATH}/${tile}/unduh${query ? `?${query}` : ''}`

  simpanBerkas(await unduhBerkas(path, { token, portal }))
}

/** Mengunduh tab Inbox Tampungan PIC sebagai CSV. */
export async function unduhTampungan(
  cari: string,
  token: string | null,
  portal: string | null,
): Promise<void> {
  const params = new URLSearchParams()
  if (cari.trim()) params.set('cari', cari.trim())
  const query = params.toString()

  simpanBerkas(await unduhBerkas(`${PATH}/tampungan/unduh${query ? `?${query}` : ''}`, { token, portal }))
}

/**
 * Hook pengajuan permintaan transfer.
 *
 * # Ia mengajukan PERMINTAAN, bukan memindahkan penugasan
 *
 * `P-1` menetapkan `PC_ASSIGN_WORKLIST` masih ditulis Pega selama masa paralel, sehingga
 * yang tercatat adalah permintaan beserta pemohonnya; pelaksanaannya tetap di Pega.
 * Penugasannya TIDAK berpindah, dan barisnya tetap ada di layar — itulah sebabnya layar
 * wajib menyatakannya, bukan membiarkan pengguna menduga.
 */
export function useAjukanTransfer() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: (body: PermintaanTransfer) =>
      callAPI<TransferResponse>(`${PATH}/transfer`, { metode: 'POST', body, token, portal }),

    // Daftar disegarkan supaya penanda "permintaan terkirim" muncul tanpa muat ulang manual.
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: ['dashboard-claim'] })
    },
  })
}

/**
 * Jalur modul Inbox Close Claim.
 *
 * # Kenapa menembak modul lain, dan kenapa lewat HTTP
 *
 * Tile Close Claim pada dashboard menampilkan populasi yang **sama persis** dengan Inbox
 * Close Claim, dan kedua tombolnya — ReOpen dan Copy Klaim — sudah punya endpoint yang
 * berjalan beserta aturan otorisasi dan pencatatannya.
 *
 * Membangunnya ulang di sini berarti dua tempat yang memutuskan hal yang sama, dan cepat
 * atau lambat keduanya berbeda. Yang dipakai karena itu endpoint yang sudah ada.
 *
 * Yang TIDAK dilakukan: mengimpor dari `src/modules/inbox-close-claim`. Aturan frontend
 * melarang satu fitur mengimpor fitur lain — kebutuhan bersama naik ke `shared/`. Yang
 * dipakai di sini adalah **kontrak HTTP**-nya, bukan kodenya, sehingga modul itu sama
 * sekali tidak disentuh.
 */
const PATH_CLOSE_CLAIM = '/api/inbox-close-claim'

/**
 * Izin mengajukan ReOpen dan Copy Klaim.
 *
 * Jawabannya datang dari DOMAIN modul Inbox Close Claim, bukan ditebak di layar ini. Di
 * sistem lama penjaganya adalah When rule `IsManagerPNC_CLOSE`; menyalin aturannya ke sini
 * berarti dua salinan yang dapat berbeda.
 */
export function useIzinPermintaanKlaim(aktif: boolean) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: ['dashboard-claim', 'izin-permintaan', portal, token],
    enabled: aktif && portal !== null,
    queryFn: () =>
      callAPI<IzinPermintaanKlaim>(`${PATH_CLOSE_CLAIM}/penyaring`, { token, portal }),

    // Izin berubah hanya saat peran pengguna berubah, dan itu menuntut masuk ulang.
    staleTime: 5 * 60 * 1000,
  })
}

/**
 * Pengajuan ReOpen atau Copy Klaim atas baris-baris yang dicentang.
 *
 * # Satu permintaan per klaim, bukan satu permintaan berisi banyak klaim
 *
 * Endpoint `POST /permintaan` menerima **satu** `klaim_id`. Mencentang lima baris karena itu
 * mengirim lima permintaan — persis seperti pengguna mengklik barisnya satu per satu di
 * layar lama, yang juga tidak punya pengajuan massal.
 *
 * Akibatnya sebagiannya dapat berhasil dan sebagiannya gagal, dan itu TIDAK disembunyikan:
 * hasilnya dikembalikan per baris. Menyatukannya menjadi satu "gagal" akan membuat pengguna
 * mengirim ulang seluruhnya, dan yang sudah tercatat akan ditolak 409 — penolakan yang lalu
 * terbaca sebagai kegagalan baru.
 *
 * Dikirim BERURUTAN, bukan serentak. Lima permintaan tulis serentak ke tabel yang sama
 * tidak mempercepat apa pun yang berarti, dan urutan tercatatnya menjadi tidak dapat
 * ditebak saat dibaca kembali di jejak audit.
 */
export function useAjukanPermintaanKlaim() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: async (input: {
      jenis: JenisPermintaanKlaim
      alasan: string
      baris: { klaim_id: string; nomor_klaim: string }[]
    }): Promise<HasilPermintaanKlaim[]> => {
      const hasil: HasilPermintaanKlaim[] = []

      for (const baris of input.baris) {
        try {
          await callAPI(`${PATH_CLOSE_CLAIM}/permintaan`, {
            token,
            portal,
            metode: 'POST',
            body: { jenis: input.jenis, klaim_id: baris.klaim_id, alasan: input.alasan },
          })
          hasil.push({ ...baris, berhasil: true })
        } catch (failure) {
          hasil.push({ ...baris, berhasil: false, pesan: pesanGalat(failure) })
        }
      }

      return hasil
    },

    onSuccess: () => {
      // Daftar tile diperbarui karena yang berubah bukan klaimnya melainkan penanda
      // permintaan tertunda pada barisnya. Daftar modul Inbox Close Claim ikut diperbarui:
      // baris yang sama tampil di sana, dan membiarkannya basi membuat pengguna mengajukan
      // dua kali dari dua layar.
      client.invalidateQueries({ queryKey: ['dashboard-claim', 'telusur'] })
      client.invalidateQueries({ queryKey: ['inbox-close-claim', 'daftar'] })
    },
  })
}

/** Membaca pesan galat yang layak dibaca pengguna. */
function pesanGalat(failure: unknown): string {
  if (failure instanceof APIError) return failure.message
  if (failure instanceof Error) return failure.message
  return 'Terjadi kesalahan pada sistem.'
}
