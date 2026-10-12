import { useQuery } from '@tanstack/react-query'

import { callAPI } from '@/api/client'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import type { DaftarResponse, KeteranganResponse } from './types'

const PATH = '/api/inbox-analyst-doctor'

/** Banyaknya baris per halaman. Backend menolak permintaan di atas 100. */
export const PAGE_SIZE = 25

/**
 * Kunci cache dikumpulkan di satu tempat supaya invalidasi tidak salah sasaran.
 *
 * PORTAL ikut menjadi bagian kunci, dan itu bukan kerapian: dua portal adalah dua badan
 * hukum dengan basis data berbeda (`ADR-0030`). Menyimpan keduanya di bawah satu kunci akan
 * membuat perpindahan portal menampilkan antrean entitas sebelumnya — kebocoran yang tampil
 * sebagai layar normal, persis `R-20`.
 *
 * TOKEN ikut pula, dan di layar ini alasannya lebih tajam daripada di layar lain: antreannya
 * disaring dengan identitas pemanggil, sehingga cache yang bertahan melewati pergantian
 * pengguna akan menampilkan tugas medis milik orang sebelumnya.
 */
const keys = {
  keterangan: (portal: string | null, token: string | null) =>
    ['inbox-analyst-doctor', 'keterangan', portal, token] as const,

  daftar: (portal: string | null, token: string | null, cari: string, lewati: number) =>
    ['inbox-analyst-doctor', 'daftar', portal, token, cari, lewati] as const,
}

/**
 * Hook keterangan layar — judul kolom, selisih terencana, dan keterbatasan.
 *
 * # Kenapa judul kolom datang dari server
 *
 * Karena kedelapannya adalah HASIL PEMBACAAN rule `pyCaption …` pada
 * `Harness/inboxAnalystDoctor_Harness-Harness.xml`, dan tempat pembacaan itu tercatat adalah
 * backend. Menyalinnya ke layar berarti daftar yang sama hidup di dua tempat, dan yang satu
 * akan tertinggal saat yang lain diperbaiki.
 *
 * Keterbatasan ikut datang dari server supaya ia HILANG DENGAN SENDIRINYA begitu
 * penghalangnya hilang — tanpa menyentuh satu baris pun di layar.
 */
export function useKeteranganAnalystDoctor() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: keys.keterangan(portal, token),
    queryFn: () => callAPI<KeteranganResponse>(`${PATH}/keterangan`, { token, portal }),
    enabled: token !== null && portal !== null,

    // Bentuk layar tidak berubah selama aplikasi berjalan: ia dibaca dari kode, bukan dari
    // data. Mengambilnya ulang tiap kali halaman berpindah hanya menambah perjalanan
    // jaringan tanpa satu pun manfaat.
    staleTime: Infinity,
    gcTime: Infinity,
  })
}

/**
 * Hook antrean penilaian medis milik pengguna yang sedang masuk.
 *
 * Menggantikan `Report Definition/InboxAnalystDoctor_RD-RD.xml` beserta ketiga penyaringnya.
 *
 * # Penyaringan dan paginasi dikerjakan SERVER
 *
 * Tabel klaim berisi puluhan juta baris (`D-10`). Menyaring di peramban berarti hasil
 * pencariannya akan BOHONG, karena hanya menyentuh halaman yang sedang terbuka.
 *
 * # Antreannya milik SATU ORANG, dan itu ditentukan server
 *
 * Layar tidak mengirim identitas apa pun — server membacanya dari sesi. Itu disengaja:
 * identitas yang dikirim layar adalah identitas yang dapat diganti layar, dan antrean ini
 * memuat klaim Personal Accident yang aksesnya dibatasi `FR-R2`.
 *
 * # Tidak dijalankan sebelum portal dipilih
 *
 * Backend memang menolak permintaan tanpa portal (`TKT-F6-002`), tetapi menembaknya lebih
 * dulu hanya untuk menerima penolakan adalah perjalanan jaringan yang sia-sia.
 */
export function useDaftarAnalystDoctor(cari: string, lewati: number) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  const params = new URLSearchParams()
  if (cari.trim()) params.set('cari', cari.trim())
  if (lewati > 0) params.set('lewati', String(lewati))
  params.set('batas', String(PAGE_SIZE))

  return useQuery({
    queryKey: keys.daftar(portal, token, cari.trim(), lewati),
    queryFn: () => callAPI<DaftarResponse>(`${PATH}?${params.toString()}`, { token, portal }),
    enabled: token !== null && portal !== null,

    // Antrean berubah setiap kali sebuah tugas diselesaikan — termasuk oleh Pega, yang masih
    // memiliki penugasannya selama masa paralel. Data dianggap usang seketika, tetapi tidak
    // ditembak ulang sendiri: pengguna yang menekan Muat ulang.
    staleTime: 0,

    // Halaman sebelumnya dipertahankan selama halaman berikutnya diambil, supaya tabel tidak
    // berkedip menjadi kosong lalu terisi lagi setiap kali nomor halaman berpindah.
    placeholderData: (previous) => previous,
  })
}
