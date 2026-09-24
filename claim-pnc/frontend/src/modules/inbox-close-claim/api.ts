import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { callAPI, HEADER_PORTAL } from '@/api/client'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import type {
  DaftarResponse,
  JenisPermintaan,
  PenyaringKlaimTutup,
  PenyaringResponse,
  PermintaanResponse,
} from './types'

const PATH = '/api/inbox-close-claim'

/** Banyaknya baris per halaman. Backend menolak permintaan di atas 100. */
export const PAGE_SIZE = 25

/**
 * Kunci cache dikumpulkan di satu tempat supaya invalidasi tidak salah sasaran.
 *
 * PORTAL ikut menjadi bagian kunci, dan itu bukan kerapian: dua portal adalah dua badan
 * hukum dengan basis data berbeda (`ADR-0030`). Menyimpan keduanya di bawah satu kunci akan
 * membuat perpindahan portal menampilkan klaim entitas sebelumnya — kebocoran yang tampil
 * sebagai layar normal, persis `R-20`.
 */
const keys = {
  daftar: (portal: string | null, token: string | null, f: PenyaringKlaimTutup) =>
    [
      'inbox-close-claim',
      'daftar',
      portal,
      token,
      f.cari ?? '',
      f.no_polis ?? '',
      f.no_klaim ?? '',
      f.pic ?? '',
      f.lini ?? '',
      f.status_transfer ?? '',
      f.status_bayar ?? '',
      f.lewati ?? 0,
    ] as const,

  penyaring: (portal: string | null, token: string | null) =>
    ['inbox-close-claim', 'penyaring', portal, token] as const,
}

/** Menyusun query string dari penyaring; yang kosong tidak dikirim sama sekali. */
function buildParams(f: PenyaringKlaimTutup): URLSearchParams {
  const params = new URLSearchParams()
  if (f.cari?.trim()) params.set('cari', f.cari.trim())
  if (f.no_polis?.trim()) params.set('no_polis', f.no_polis.trim())
  if (f.no_klaim?.trim()) params.set('no_klaim', f.no_klaim.trim())
  if (f.pic?.trim()) params.set('pic', f.pic.trim())
  if (f.lini?.trim()) params.set('lini', f.lini.trim())
  if (f.status_transfer?.trim()) params.set('status_transfer', f.status_transfer.trim())
  if (f.status_bayar?.trim()) params.set('status_bayar', f.status_bayar.trim())
  return params
}

/**
 * Hook keterangan layar — isi ketiga dropdown penyaring.
 *
 * # Kenapa bentuk penyaring datang dari server
 *
 * Karena kelima pilihan lini bisnis adalah HASIL PEMBACAAN activity Pega
 * (`Activity/GCNMGetManagerReopenCase_Act-Act.xml`), dan tempat pembacaan itu tercatat
 * adalah backend. Menyalinnya ke layar berarti daftar yang sama hidup di dua tempat, dan
 * yang satu akan tertinggal saat yang lain diperbaiki.
 *
 * Di layar ini akibatnya nyata: nilainya dibandingkan server sebagai TEKS, sehingga satu
 * ejaan yang berbeda menghasilkan penyaring yang tidak pernah cocok — daftar kosong tanpa
 * satu pun pesan galat.
 */
export function usePenyaringKlaimTutup() {
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
 * Hook daftar klaim yang sudah tutup.
 *
 * Menggantikan `RDB List/GcnmBrowseReopenCase_SQL-SQL.xml` beserta
 * `RDB List/GCNMCountCloseClaim-SQL.xml` dan activity yang menyisipkan enam potongan WHERE
 * ke dalam keduanya.
 *
 * # Penyaringan dikerjakan SERVER
 *
 * Tabel klaim berisi puluhan juta baris (`D-10`), dan layar ini membaca bagian yang PALING
 * BESAR darinya — klaim yang sudah tutup bertambah terus dan tidak pernah berkurang.
 * Menyaringnya di peramban berarti hasil pencariannya akan BOHONG, karena hanya menyentuh
 * halaman yang sedang terbuka.
 *
 * # Tidak dijalankan sebelum portal dipilih
 *
 * Backend memang menolak permintaan tanpa portal (`TKT-F6-002`), tetapi menembaknya lebih
 * dulu hanya untuk menerima penolakan adalah perjalanan jaringan yang sia-sia.
 */
export function useDaftarKlaimTutup(filter: PenyaringKlaimTutup) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  const params = buildParams(filter)
  if (filter.lewati) params.set('lewati', String(filter.lewati))
  params.set('batas', String(PAGE_SIZE))

  return useQuery({
    queryKey: keys.daftar(portal, token, filter),
    queryFn: () =>
      callAPI<DaftarResponse>(`${PATH}?${params.toString()}`, { token, portal }),
    enabled: token !== null && portal !== null,

    // Daftar ini berubah setiap kali sebuah klaim ditutup atau dibuka kembali. Data
    // dianggap usang seketika, tetapi tidak ditembak ulang sendiri — pengguna yang menekan
    // Muat ulang.
    staleTime: 0,
  })
}

/**
 * Hook pengajuan permintaan ReOpen atau Copy Klaim.
 *
 * # Ia mengajukan PERMINTAAN, bukan menjalankan tindakannya
 *
 * `P-1` menetapkan klaim masih ditulis Pega selama masa paralel, sehingga yang tercatat di
 * sini adalah permintaan beserta pelakunya; eksekusinya tetap di Pega. Klaimnya TIDAK
 * berubah, dan barisnya tetap ada di layar.
 *
 * Karena itu daftarnya diperbarui setelah berhasil: yang berubah bukan klaimnya melainkan
 * PENANDA permintaan tertunda pada barisnya. Tanpa pembaruan itu, pengguna tidak melihat
 * apa pun berubah dan akan menekan tombolnya lagi — yang lalu ditolak 409, dan penolakan
 * itu terbaca sebagai kegagalan.
 */
export function useAjukanPermintaan() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: (input: { jenis: JenisPermintaan; klaim_id: string; alasan: string }) =>
      callAPI<PermintaanResponse>(`${PATH}/permintaan`, {
        token,
        portal,
        metode: 'POST',
        body: input,
      }),

    onSuccess: () => {
      // Seluruh halaman daftar diperbarui, bukan hanya yang sedang terbuka: baris yang sama
      // dapat terlihat pada halaman lain dengan penyaring yang berbeda.
      void client.invalidateQueries({ queryKey: ['inbox-close-claim', 'daftar'] })
    },
  })
}

/**
 * Menyusun unduhan CSV beserta penyaring yang sedang berlaku.
 *
 * # Kenapa bukan hook, dan kenapa tidak lewat callAPI
 *
 * Unduhan bukan data yang disimpan di cache; ia berkas yang diserahkan ke peramban. Tetapi
 * ia TIDAK dapat dijadikan `<a href>` biasa, karena permintaannya menuntut dua header —
 * `Authorization` dan `X-Portal` — dan tautan biasa tidak dapat mengirim header.
 *
 * Yang dipakai karena itu: permintaan biasa, lalu hasilnya diserahkan ke peramban sebagai
 * objek URL. Konsekuensinya berkas dimuat seluruhnya ke memori peramban lebih dulu, dan itu
 * diterima — backend sudah membatasi jumlah barisnya.
 */
export async function unduhKlaimTutupCSV(
  filter: PenyaringKlaimTutup,
  token: string,
  portal: string,
): Promise<void> {
  const query = buildParams(filter).toString()
  const url = query ? `${PATH}/unduh?${query}` : `${PATH}/unduh`

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
    link.download = fileNameFrom(response) ?? 'inbox-close-claim.csv'
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
