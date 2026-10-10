import { useQuery } from '@tanstack/react-query'

import { HEADER_PORTAL, callAPI } from '@/api/client'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import type { HasilPencarian, Keterangan, Penyaring } from './types'

const PATH = '/api/monitoring-slink-ojk'

/**
 * Kunci cache dikumpulkan di satu tempat supaya invalidasi tidak salah sasaran.
 *
 * Portal ikut menjadi bagian SETIAP kunci, dan itu bukan kerapian: kewajiban lapor SLIK
 * adalah kewajiban satu badan hukum kepada OJK, dan menyimpan dua entitas di bawah satu
 * kunci akan membuat perpindahan portal menampilkan kewajiban entitas sebelumnya
 * (`R-20`).
 */
const keys = {
  keterangan: (portal: string | null, token: string | null) =>
    ['monitoring-slink-ojk', 'keterangan', portal, token] as const,

  data: (portal: string | null, token: string | null, filter: Penyaring, halaman: number) =>
    [
      'monitoring-slink-ojk',
      'data',
      portal,
      token,
      filter.segmen,
      filter.business_name,
      filter.date_of_loss,
      filter.date_of_request_document,
      halaman,
    ] as const,
}

/**
 * useSessionPortal menyatukan dua nilai yang SELALU dibutuhkan bersama.
 *
 * Keduanya dibaca di setiap hook di bawah, dan memisahkannya hanya akan mengulang dua
 * baris yang sama — beserta risiko salah satu yang tertinggal saat hook baru ditambahkan.
 */
function useSessionPortal() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  return { token, portal, ready: token !== null && portal !== null }
}

/**
 * Hook keterangan layar: kedua segmen beserta katalog kolomnya.
 *
 * Katalognya jarang berubah — ia diturunkan dari export Pega, bukan dari data — sehingga
 * `staleTime` lima menit mencegah permintaan berulang setiap kali segmen berpindah.
 */
export function useKeterangan() {
  const { token, portal, ready } = useSessionPortal()

  return useQuery({
    queryKey: keys.keterangan(portal, token),
    queryFn: () => callAPI<Keterangan>(`${PATH}/keterangan`, { token, portal }),
    enabled: ready,
    staleTime: 5 * 60 * 1000,
  })
}

/**
 * queryString menyusun parameter penyaring.
 *
 * Isian KOSONG tidak ikut dikirim, bukan dikirim sebagai teks kosong. Alasannya bukan
 * kerapian URL: backend membedakan "tidak menyaring" dari "menyaring dengan nilai
 * kosong", dan mengirim keduanya sebagai hal yang sama akan membuat layar tanpa penyaring
 * menampilkan nol baris — yang di layar ini terbaca sebagai "tidak ada yang perlu
 * dilaporkan ke OJK".
 */
function queryString(filter: Penyaring, halaman: number, ukuran: number): string {
  const params = new URLSearchParams()
  params.set('segmen', filter.segmen)

  const optional: Array<[string, string]> = [
    ['business_name', filter.business_name],
    ['date_of_loss', filter.date_of_loss],
    ['date_of_request_document', filter.date_of_request_document],
  ]
  for (const [key, value] of optional) {
    if (value.trim() !== '') params.set(key, value.trim())
  }

  params.set('halaman', String(halaman))
  params.set('ukuran', String(ukuran))
  return params.toString()
}

/** Banyaknya baris per halaman. */
export const UKURAN_HALAMAN = 50

/**
 * Hook tombol "Cari Data".
 *
 * `enabled` dikendalikan pemanggil, bukan dijalankan otomatis saat penyaring berubah:
 * layar lama menuntut tombol ditekan, dan menembak permintaan setiap kali satu huruf
 * diketik akan menjalankan kueri atas puluhan juta baris berkali-kali (`D-10`).
 */
export function useData(filter: Penyaring, halaman: number, aktif: boolean) {
  const { token, portal, ready } = useSessionPortal()

  return useQuery({
    queryKey: keys.data(portal, token, filter, halaman),
    queryFn: () =>
      callAPI<HasilPencarian>(
        `${PATH}/data?${queryString(filter, halaman, UKURAN_HALAMAN)}`,
        { token, portal },
      ),
    enabled: ready && aktif,

    // Halaman sebelumnya DIPERTAHANKAN saat halaman berpindah, supaya tabel tidak
    // berkedip menjadi kosong lalu terisi lagi. Pada tabel 38 kolom, kedipan itu
    // membuat layar terasa rusak.
    placeholderData: (previous) => previous,
  })
}

/**
 * unduh mengambil berkas CSV lalu menyerahkannya ke peramban.
 *
 * # Kenapa lewat `fetch`, bukan `window.open` atau tautan biasa
 *
 * Karena kedua berkasnya menuntut DUA header yang tidak dapat dipasang pada navigasi
 * biasa: `Authorization` dan `X-Portal`. Menaruh keduanya di URL akan membuat token dan
 * entitas tercatat di log peramban, log proxy, dan header `Referer` — dan berkas ini
 * memuat nomor CIF debitur beserta tanggal lahirnya.
 *
 * Galatnya dibiarkan naik ke pemanggil apa adanya supaya layar dapat menampilkannya
 * dengan komponen galat yang sama seperti bagian lain — bukan sebagai jendela kosong
 * yang tidak menjelaskan apa pun.
 */
async function unduh(
  path: string,
  namaBawaan: string,
  token: string | null,
  portal: string | null,
): Promise<void> {
  const header: Record<string, string> = {}
  if (token) header['Authorization'] = `Bearer ${token}`
  if (portal) header[HEADER_PORTAL] = portal

  const response = await fetch(path, { headers: header })
  if (!response.ok) {
    // Backend menjawab galat penyaring sebagai JSON — sebelum satu byte CSV pun
    // terkirim. Pesannya dibaca dari sana supaya pengguna tahu isian mana yang salah.
    let pesan = 'Berkas tidak dapat diunduh.'
    try {
      const body = (await response.json()) as { pesan?: string }
      if (body.pesan) pesan = body.pesan
    } catch {
      // Badan bukan JSON — pesan bawaan dipakai.
    }
    throw new Error(pesan)
  }

  const blob = await response.blob()
  const url = URL.createObjectURL(blob)
  const anchor = document.createElement('a')
  anchor.href = url
  anchor.download = namaDariHeader(response) ?? namaBawaan
  document.body.appendChild(anchor)
  anchor.click()
  anchor.remove()

  // Dibebaskan setelah unduhan dimulai; tanpa ini, berkas puluhan MB tetap menempel di
  // memori tab sampai halamannya ditutup.
  URL.revokeObjectURL(url)
}

/** namaDariHeader membaca nama berkas yang ditentukan server. */
function namaDariHeader(response: Response): string | null {
  const disposition = response.headers.get('Content-Disposition')
  if (!disposition) return null
  const match = /filename="([^"]+)"/.exec(disposition)
  // `?? null` bukan basa-basi: `noUncheckedIndexedAccess` menyatakan `match[1]` dapat
  // `undefined` meski pola ini punya satu grup, dan pemanggil membedakan "server tidak
  // menyebut nama" dari "namanya kosong".
  return match?.[1] ?? null
}

/** Mengunduh berkas ekspor satu segmen — tombol "Export Data". */
export function useEkspor() {
  const { token, portal } = useSessionPortal()

  return (filter: Penyaring) =>
    unduh(
      `${PATH}/ekspor?${queryString(filter, 1, UKURAN_HALAMAN)}`,
      `Laporan SLIK OJK ${filter.segmen}.csv`,
      token,
      portal,
    )
}

/** Mengunduh berkas contoh unggahan — tombol "Format File". */
export function useFormatFile() {
  const { token, portal } = useSessionPortal()

  return () =>
    unduh(
      `${PATH}/format`,
      'Format Auto Klaim Slik OJK ASM.csv',
      token,
      portal,
    )
}

/**
 * Hasil satu aksi tulis.
 *
 * `baru` dan `diperbarui` dipisah karena artinya berbeda bagi pelapor: `diperbarui`
 * adalah klaim yang SUDAH pernah dilaporkan dan kini dilaporkan lagi — dan itulah yang
 * patut diperiksa sebelum berkasnya dikirim ke OJK.
 */
export type HasilTulis = {
  baru: number
  diperbarui: number
  total: number
  ditolak: Array<{ baris?: number; no_klaim?: string; alasan: string }>
}

/**
 * Menyusun laporan dari data klaim sumber — tombol "Proses Data Klaim".
 *
 * Penyaringnya dikirim SAMA seperti "Cari Data", supaya yang tersusun persis yang
 * terlihat di tabel.
 *
 * Bukan `useMutation`: layar ini memanggilnya dari satu tombol dan menangani hasilnya
 * sendiri, dan membungkusnya sebagai mutation hanya menambah lapisan tanpa cache yang
 * perlu dibatalkan.
 */
export function useProses() {
  const { token, portal } = useSessionPortal()

  return (filter: Penyaring): Promise<HasilTulis> =>
    callAPI<HasilTulis>(
      `${PATH}/proses?${queryString(filter, 1, UKURAN_HALAMAN)}`,
      { metode: 'POST', token, portal },
    )
}

/**
 * Mengunggah berkas CSV — tombol "Upload Data Klaim".
 *
 * Berkasnya dikirim sebagai `FormData`; `callAPI` mengenalinya dan TIDAK memasang
 * `Content-Type` sendiri, supaya peramban yang menuliskan `boundary`-nya.
 */
export function useUnggah() {
  const { token, portal } = useSessionPortal()

  return (berkas: File): Promise<HasilTulis> => {
    const form = new FormData()
    form.append('berkas', berkas)
    return callAPI<HasilTulis>(`${PATH}/unggah`, {
      metode: 'POST',
      body: form,
      token,
      portal,
    })
  }
}

/**
 * Mengirim data debitur klaim yang sedang tampil ke sistem SLIK — tombol "SLIK OJK".
 *
 * Sasarannya SAMA dengan "Proses Data Klaim": himpunan yang sedang disaring. Badan
 * permintaan sengaja dibiarkan KOSONG — backend membaca penyaringnya dari query string,
 * dan badan berisi `no_klaim` dipakai pemanggil yang ingin mengirim satu klaim saja.
 *
 * Tanpa alamat layanan yang dikonfigurasi, backend menjawab `503` beserta sebabnya —
 * dan TIDAK meninggalkan baris pengiriman palsu.
 */
export function useKirimSlik() {
  const { token, portal } = useSessionPortal()

  return (filter: Penyaring): Promise<HasilTulis> =>
    callAPI<HasilTulis>(
      `${PATH}/kirim?${queryString(filter, 1, UKURAN_HALAMAN)}`,
      { metode: 'POST', token, portal },
    )
}
