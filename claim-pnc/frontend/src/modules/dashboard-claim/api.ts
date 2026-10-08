import { useQuery } from '@tanstack/react-query'

import { callAPI } from '@/api/client'
import { useScreenMetadata } from '@/api/inboxShared'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import type {
  PenyaringDashboard,
  PenyaringResponse,
  RingkasanResponse,
  TelusurResponse,
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
  return useScreenMetadata<PenyaringResponse>(keys.penyaring, `${PATH}/penyaring`)
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
