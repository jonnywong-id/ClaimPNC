import { useQuery } from '@tanstack/react-query'

import { callAPI, HEADER_PORTAL } from '@/api/client'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import type {
  OutstandingFilter,
  OutstandingListResponse,
  OutstandingSummaryResponse,
} from './types'

const PATH = '/api/inbox-outstanding'

/** Banyaknya baris per halaman. Backend menolak permintaan di atas 100. */
export const PAGE_SIZE = 25

/**
 * Kunci cache dikumpulkan di satu tempat supaya invalidasi tidak salah sasaran.
 *
 * PORTAL ikut menjadi bagian kunci, dan itu bukan kerapian: dua portal adalah dua badan
 * hukum dengan basis data berbeda (`ADR-0030`). Menyimpan keduanya di bawah satu kunci
 * akan membuat perpindahan portal menampilkan klaim entitas sebelumnya — kebocoran yang
 * tampil sebagai layar normal, persis `R-20`.
 */
const keys = {
  list: (portal: string | null, token: string | null, f: OutstandingFilter) =>
    [
      'inbox-outstanding',
      portal,
      token,
      f.search ?? '',
      f.stage ?? '',
      f.branch ?? '',
      f.documentStatus ?? '',
      f.offset ?? 0,
    ] as const,

  // Ringkasan sengaja TIDAK memuat status maupun offset pada kuncinya.
  //
  // Ia tidak berubah oleh keduanya — donut harus tetap menampilkan seluruh status supaya
  // irisan yang sedang dipilih terlihat dan dapat dibatalkan. Memasukkannya ke kunci akan
  // menembak ulang ringkasan setiap kali pengguna mengeklik irisan atau berpindah halaman,
  // dan hasilnya selalu sama.
  summary: (portal: string | null, token: string | null, f: OutstandingFilter) =>
    ['inbox-outstanding', 'ringkasan', portal, token, f.search ?? '', f.stage ?? '', f.branch ?? ''] as const,
}

function filterParams(f: OutstandingFilter): URLSearchParams {
  const params = new URLSearchParams()
  if (f.search?.trim()) params.set('cari', f.search.trim())
  if (f.stage?.trim()) params.set('tahap', f.stage.trim())
  if (f.branch?.trim()) params.set('cabang', f.branch.trim())
  return params
}

function buildPath(f: OutstandingFilter): string {
  const params = filterParams(f)
  if (f.documentStatus) params.set('status_dokumen', f.documentStatus)
  if (f.offset) params.set('lewati', String(f.offset))
  params.set('batas', String(PAGE_SIZE))

  return `${PATH}?${params.toString()}`
}

/**
 * Hook daftar klaim yang masih berjalan.
 *
 * Menggantikan `RDB List/BrowseInboxOutstanding1-SQL.xml` beserta activity yang
 * menyisipkan potongan WHERE per jabatan ke dalamnya.
 *
 * # Penyaringan dikerjakan SERVER
 *
 * Tabel klaim berisi puluhan juta baris (`D-10`). Menyaringnya di peramban berarti
 * mengirim seluruh riwayat klaim ke setiap layar yang dibuka — dan hasil pencariannya
 * akan BOHONG, karena hanya menyentuh halaman yang sedang terbuka.
 *
 * # Tidak dijalankan sebelum portal dipilih
 *
 * Backend memang menolak permintaan tanpa portal (`TKT-F6-002`), tetapi menembaknya lebih
 * dulu hanya untuk menerima penolakan adalah perjalanan jaringan yang sia-sia. Layar yang
 * menuntun pengguna memilih portal lebih berguna daripada pesan galat.
 */
export function useOutstandingList(filter: OutstandingFilter) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: keys.list(portal, token, filter),
    queryFn: () => callAPI<OutstandingListResponse>(buildPath(filter), { token, portal }),
    enabled: token !== null && portal !== null,

    // Daftar ini berubah setiap kali ada yang mengerjakan klaim, dan layar pemantauan
    // yang menampilkan keadaan basi menyesatkan. Data dianggap usang seketika, tetapi
    // tidak ditembak ulang sendiri — pengguna yang menekan Muat ulang.
    staleTime: 0,
  })
}

/**
 * Hook ringkasan inbox per status kelengkapan dokumen — sumber donut.
 *
 * # Ia permintaan TERSENDIRI, bukan bagian respons daftar
 *
 * Keduanya berubah pada irama yang berbeda: daftar ditembak ulang setiap kali pengguna
 * berpindah halaman atau mengetik pencarian, sedangkan ringkasan hanya perlu berubah saat
 * penyaring di luar status berubah. Menyatukannya berarti menghitung ulang donut pada
 * setiap penekanan tombol paginasi.
 */
export function useOutstandingSummary(filter: OutstandingFilter) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: keys.summary(portal, token, filter),
    queryFn: () => {
      const params = filterParams(filter)
      const query = params.toString()
      return callAPI<OutstandingSummaryResponse>(
        query ? `${PATH}/ringkasan?${query}` : `${PATH}/ringkasan`,
        { token, portal },
      )
    },
    enabled: token !== null && portal !== null,
    staleTime: 0,
  })
}

/**
 * Mengunduh CSV.
 *
 * # Unduhan TIDAK mengikuti penyaring layar, dan itu disengaja
 *
 * Fungsi ini sempat mengirim `cari`, `tahap`, dan `cabang` — penyaring yang sedang berlaku
 * di layar. Backend sekarang mengabaikannya, dan pengirimannya dihentikan supaya kode di
 * sini tidak menjanjikan hal yang tidak terjadi.
 *
 * Sebabnya: unduhan dan daftar adalah dua hal yang berbeda di sistem lama.
 * `RDB List/ExportDataDetailKlaim-SQL.xml` tidak menyaring pemilik pekerjaan maupun
 * penyaring layar; yang membatasinya hanyalah LINI BISNIS petugas dan rentang tanggal.
 * Berkasnya memang berkas pemantauan satu lini, bukan salinan layar.
 *
 * Layar menyatakan perbedaan ini kepada pengguna — lihat OutstandingPage.
 *
 * # Kenapa bukan hook, dan kenapa tidak lewat callAPI
 *
 * Unduhan bukan data yang disimpan di cache; ia berkas yang diserahkan ke peramban. Tetapi
 * ia TIDAK dapat dijadikan `<a href>` biasa, karena permintaannya menuntut dua header —
 * `Authorization` dan `X-Portal` — dan tautan biasa tidak dapat mengirim header.
 *
 * Yang dipakai karena itu: permintaan biasa, lalu hasilnya diserahkan ke peramban sebagai
 * objek URL. Konsekuensinya berkas dimuat seluruhnya ke memori peramban lebih dulu, dan
 * itu diterima — backend sudah membatasi jumlah barisnya.
 */
export async function downloadOutstandingCSV(token: string, portal: string): Promise<void> {
  const url = `${PATH}/unduh`

  const response = await fetch(url, {
    headers: {
      Authorization: `Bearer ${token}`,
      [HEADER_PORTAL]: portal,
    },
  })
  if (!response.ok) {
    throw new Error('Berkas tidak dapat diunduh.')
  }

  const blob = await response.blob()
  const objectURL = URL.createObjectURL(blob)

  try {
    const link = document.createElement('a')
    link.href = objectURL
    link.download = fileNameFrom(response) ?? 'inbox-outstanding.csv'
    document.body.appendChild(link)
    link.click()
    link.remove()
  } finally {
    // Objek URL menahan blob-nya di memori sampai dicabut. Tanpa ini, setiap unduhan
    // meninggalkan satu salinan berkas yang tidak pernah dilepas.
    URL.revokeObjectURL(objectURL)
  }
}

/** Membaca nama berkas dari header Content-Disposition; null bila tidak ada. */
function fileNameFrom(response: Response): string | null {
  const disposition = response.headers.get('Content-Disposition')
  if (!disposition) return null

  const match = /filename="([^"]+)"/.exec(disposition)
  return match?.[1] ?? null
}
