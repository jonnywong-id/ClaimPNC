import { useQuery } from '@tanstack/react-query'

import { callAPI } from '@/api/client'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import type { OpenResponse, SearchForm, SearchResponse } from './types'

const PATH = '/api/riwayat-klaim'

/**
 * Kunci cache dikumpulkan di satu tempat supaya invalidasi tidak salah sasaran.
 *
 * Portal ikut menjadi bagian kunci, dan itu BUKAN kerapian: riwayat klaim satu badan
 * hukum bukan riwayat badan hukum lain, dan menyimpan keduanya di bawah satu kunci akan
 * membuat perpindahan portal menampilkan data entitas sebelumnya (`R-20`).
 */
const keys = {
  open: (portal: string | null, token: string | null) =>
    ['riwayat-klaim', 'buka', portal, token] as const,

  search: (portal: string | null, token: string | null, form: SearchForm, page: number) =>
    [
      'riwayat-klaim',
      'cari',
      portal,
      token,
      form.tipe,
      form.nilai,
      form.tanggal_pencarian,
      form.tanggal_lahir,
      page,
    ] as const,
}

/**
 * Hook pembukaan layar — menjalankan gerbang proteksi data.
 *
 * # Kenapa useQuery untuk sebuah POST
 *
 * Karena yang dibutuhkan adalah "kerjakan SEKALI per kunjungan, lalu simpan hasilnya",
 * dan itu semantik cache, bukan semantik aksi.
 *
 * Membuka layar MEMAKAI satu jatah pencarian — perilaku sistem lama yang dibangun penuh
 * atas keputusan Work Owner 2026-09-20. Memakai useMutation di dalam useEffect akan
 * memanggilnya DUA KALI di pengembangan, karena `StrictMode` sengaja menjalankan efek dua
 * kali; dua kali berarti dua jatah terpakai untuk satu kunjungan.
 *
 * `staleTime: Infinity` membuat TanStack Query tidak pernah mengulanginya selama layar
 * terbuka, dan pengulangan permintaan yang gagal dimatikan: percobaan ulang otomatis atas
 * permintaan yang memakai jatah akan menghabiskannya tanpa satu pun tindakan pengguna.
 */
export function useOpenClaimHistory() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: keys.open(portal, token),
    queryFn: () =>
      callAPI<OpenResponse>(`${PATH}/buka`, { metode: 'POST', token, portal }),

    // Tidak ditembak sebelum portal dipilih. Backend memang menolak permintaan tanpa
    // portal (`TKT-F6-002`), tetapi menembaknya lebih dulu hanya untuk menerima
    // penolakan akan memakai jatah pada permintaan yang sudah pasti gagal.
    enabled: token !== null && portal !== null,

    staleTime: Infinity,
    gcTime: Infinity,
    refetchOnMount: false,
    refetchOnReconnect: false,
    retry: false,
  })
}

/**
 * Hook pencarian riwayat klaim.
 *
 * Tidak memakai jatah — hanya pembukaan layar yang memakainya. Ia tetap menempuh gerbang
 * di server, sehingga pengguna yang jatahnya habis di tengah pekerjaan tetap ditolak.
 *
 * # Kenapa penyaringan dikerjakan di SERVER
 *
 * Karena yang dicari adalah `T_CLAIM_PNC` yang berisi puluhan juta baris (`D-10`).
 * Menyaringnya di peramban berarti mengirim seluruh isi tabel ke setiap pencarian —
 * bukan lambat, melainkan mustahil.
 */
export function useClaimHistorySearch(form: SearchForm, page: number, enabled: boolean) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: keys.search(portal, token, form, page),
    queryFn: () => callAPI<SearchResponse>(buildPath(form, page), { token, portal }),
    enabled: enabled && token !== null && portal !== null,

    // Hasil sebelumnya ditahan selama halaman berikutnya dimuat, alih-alih tabel
    // berkedip menjadi kosong lalu terisi lagi.
    placeholderData: (previous) => previous,

    // Riwayat klaim berubah saat klaim diproses petugas lain, jadi cache-nya pendek.
    staleTime: 30 * 1000,
  })
}

/**
 * buildPath menyusun parameter pencarian.
 *
 * Isian yang kosong TIDAK dikirim, bukan dikirim sebagai teks kosong: server membedakan
 * "tidak dikirim" dari "dikirim kosong", dan yang kedua akan terbaca sebagai tanggal yang
 * tidak dapat dibaca.
 */
function buildPath(form: SearchForm, page: number): string {
  const params = new URLSearchParams()
  params.set('tipe', form.tipe)

  if (form.nilai.trim()) params.set('nilai', form.nilai.trim())
  if (form.tanggal_pencarian) params.set('tanggal_pencarian', form.tanggal_pencarian)
  if (form.tanggal_lahir) params.set('tanggal_lahir', form.tanggal_lahir)
  if (page > 1) params.set('halaman', String(page))

  return `${PATH}?${params.toString()}`
}
