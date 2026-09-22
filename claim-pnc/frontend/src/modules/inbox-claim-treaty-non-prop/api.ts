import { useMutation, useQuery } from '@tanstack/react-query'

import { callAPI, HEADER_PORTAL } from '@/api/client'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import type { FilterForm, ListResponse, MetadataResponse } from './types'

const PATH = '/api/inbox-claim-treaty-non-prop'

/**
 * Kunci cache dikumpulkan di satu tempat supaya invalidasi tidak salah sasaran.
 *
 * Portal ikut menjadi bagian kunci, dan itu BUKAN kerapian: antrean kerja satu badan hukum
 * bukan antrean badan hukum lain, dan menyimpan keduanya di bawah satu kunci akan membuat
 * perpindahan portal menampilkan pekerjaan entitas sebelumnya (`R-20`).
 *
 * Kedua checkbox ikut pula. Keduanya mengubah KUERI yang dijalankan server — bukan sekadar
 * menyaring hasil yang sama — sehingga dua keadaan checkbox yang berbeda tidak boleh
 * berbagi satu entri cache.
 */
const keys = {
  metadata: (portal: string | null, token: string | null) =>
    ['inbox-claim-treaty-non-prop', 'tab', portal, token] as const,

  list: (portal: string | null, token: string | null, filter: FilterForm, page: number) =>
    [
      'inbox-claim-treaty-non-prop',
      'daftar',
      portal,
      token,
      filter.tab,
      filter.lihatSemua,
      filter.lihatTBA,
      page,
    ] as const,
}

/**
 * Hook keterangan layar — daftar tab, kolomnya, dan selisih terencana yang berlaku.
 *
 * # Kenapa bentuk layar datang dari server
 *
 * Karena kolom tiap tab adalah HASIL PEMBACAAN export Pega, dan tempat pembacaan itu
 * tercatat adalah backend (`internal/inboxclaimtreatynonprop/tab.go`). Menyalinnya ke layar
 * berarti daftar yang sama hidup di dua tempat, dan yang satu akan tertinggal saat yang
 * lain diperbaiki.
 *
 * Keadaan "tab terhalang" ikut datang dari sana dengan alasan yang sama — begitu Tim Pega
 * mengirim kueri komite yang hilang, penghalangnya hilang tanpa menyunting frontend.
 */
export function useClaimTreatyNonPropMetadata() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: keys.metadata(portal, token),
    queryFn: () => callAPI<MetadataResponse>(`${PATH}/tab`, { token, portal }),
    enabled: token !== null && portal !== null,

    // Bentuk layar tidak berubah selama aplikasi berjalan: ia dibaca dari kode, bukan
    // dari data. Mengambilnya ulang tiap kali tab berpindah hanya menambah perjalanan
    // jaringan tanpa satu pun manfaat.
    staleTime: Infinity,
    gcTime: Infinity,
  })
}

/**
 * Hook isi satu tab.
 *
 * # Kenapa penyaringan dan paginasi dikerjakan di SERVER
 *
 * Karena yang dibaca adalah tabel penugasan Pega yang berisi puluhan juta baris (`D-10`),
 * digabungkan ke dua tabel lain. Menyaringnya di peramban berarti menarik seluruh antrean
 * lebih dulu.
 *
 * Halaman di sini benar-benar dipotong basis data dengan `OFFSET … FETCH NEXT`, sehingga
 * jumlah "total" tetap tepat tanpa menarik seluruh baris.
 */
export function useClaimTreatyNonPropList(
  filter: FilterForm,
  page: number,
  enabled: boolean,
) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: keys.list(portal, token, filter, page),
    queryFn: () => callAPI<ListResponse>(buildPath(filter, page), { token, portal }),
    enabled: enabled && token !== null && portal !== null,

    // Hasil sebelumnya ditahan selama halaman berikutnya dimuat, alih-alih tabel berkedip
    // menjadi kosong lalu terisi lagi.
    placeholderData: (previous) => previous,

    // Antrean kerja berubah saat petugas lain mengambil pekerjaannya, jadi cache-nya
    // pendek — sama dengan layar Treaty Prop, yang dibuka berulang kali sepanjang hari.
    staleTime: 15 * 1000,
  })
}

/**
 * Hook tombol ekspor.
 *
 * # Kenapa berkasnya diambil dengan fetch, bukan dengan tautan unduh biasa
 *
 * Karena `<a href>` dan `window.open` TIDAK membawa header — dan endpoint ini menuntut
 * dua: `Authorization` dan `X-Portal`. Satu-satunya cara memakai tautan biasa adalah
 * menaruh token di dalam alamat, dan itu ditolak dengan alasan yang sudah dicatat di
 * `api/client.ts`: nilai di URL ikut tercatat di riwayat peramban, log proxy, dan header
 * Referer. Berkas ini memuat nama tertanggung dan nama Ceding Co; jejaknya tidak boleh
 * tertinggal di sana.
 *
 * # Penyaringnya SAMA dengan yang sedang tampil
 *
 * Isian yang sama dikirim ke kedua endpoint. Ekspor yang mengabaikan penyaring akan
 * mengeluarkan berkas yang isinya tidak dapat dicocokkan dengan apa pun di layar — dan
 * pada layar berisi dua checkbox, itu bukan kekeliruan yang mudah disadari.
 *
 * # Biaya yang disadari
 *
 * Peladen MENGALIRKAN berkasnya potong demi potong, tetapi peramban menampungnya utuh
 * sebagai Blob sebelum menyimpannya. Manfaat pengaliran karena itu tinggal di sisi
 * peladen — memorinya tetap datar — sedangkan memori peramban tumbuh sebesar berkasnya.
 * Pada batas 50.000 baris itu beberapa megabita, dan dapat diterima.
 */
export function useExportClaimTreatyNonProp() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useMutation({
    mutationFn: async (filter: FilterForm) => {
      const header: Record<string, string> = {}
      if (token) header['Authorization'] = `Bearer ${token}`
      if (portal) header[HEADER_PORTAL] = portal

      const params = filterParams(filter).toString()
      const address = params ? `${PATH}/ekspor?${params}` : `${PATH}/ekspor`

      const response = await fetch(address, { headers: header })
      if (!response.ok) {
        // Galat dijawab sebagai JSON selama header belum terkirim; setelah itu tidak
        // bisa lagi. Yang dibaca di sini adalah kasus pertama.
        const body = (await response.json().catch(() => null)) as { pesan?: string } | null
        throw new Error(body?.pesan ?? 'Berkas ekspor tidak dapat diambil.')
      }

      const blob = await response.blob()
      downloadBlob(blob, filenameOf(response) ?? 'klaim-treaty-non-prop.csv')
    },
  })
}

/**
 * filterParams menyusun ketiga isian penyaring.
 *
 * Ia dipakai daftar DAN ekspor, supaya keduanya tidak dapat membaca penyaring dengan cara
 * yang berbeda.
 *
 * Isian yang kosong TIDAK dikirim, bukan dikirim sebagai teks kosong: server membedakan
 * "tidak dikirim" dari "dikirim kosong", dan pada `tab` yang kedua akan menjadi tab bawaan
 * sementara yang pertama memang itu yang diinginkan.
 *
 * Kedua checkbox hanya dikirim saat BENAR. Mengirim `lihat_semua=0` tidak salah, tetapi ia
 * membuat dua permintaan yang hasilnya sama punya URL berbeda — dan URL berbeda berarti
 * cache terpisah untuk hasil yang sama.
 */
function filterParams(filter: FilterForm): URLSearchParams {
  const params = new URLSearchParams()

  if (filter.tab) params.set('tab', filter.tab)
  if (filter.lihatSemua) params.set('lihat_semua', '1')
  if (filter.lihatTBA) params.set('lihat_tba', '1')

  return params
}

/** buildPath menyusun alamat permintaan daftar beserta halamannya. */
function buildPath(filter: FilterForm, page: number): string {
  const params = filterParams(filter)
  if (page > 1) params.set('halaman', String(page))

  const query = params.toString()
  return query ? `${PATH}?${query}` : PATH
}

/** filenameOf membaca nama berkas dari header Content-Disposition. */
function filenameOf(response: Response): string | null {
  const disposition = response.headers.get('Content-Disposition')
  if (!disposition) return null
  const found = /filename="([^"]+)"/.exec(disposition)
  return found?.[1] ?? null
}

/**
 * downloadBlob menyimpan berkas lewat tautan sementara.
 *
 * URL objeknya DICABUT setelah dipakai. Tanpa itu, blob-nya tetap dipegang peramban
 * sampai tab ditutup — dan pada layar yang dipakai sepanjang hari, setiap ekspor
 * menumpuk memori yang tidak pernah dilepas.
 */
function downloadBlob(blob: Blob, filename: string): void {
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = filename
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  URL.revokeObjectURL(url)
}
