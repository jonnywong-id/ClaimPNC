import { useMutation } from '@tanstack/react-query'

import { downloadExport, useQueueList, useScreenMetadata } from '@/api/inboxShared'
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
  return useScreenMetadata<MetadataResponse>(keys.metadata, `${PATH}/tab`)
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
  return useQueueList<ListResponse>(
    (portal, token) => keys.list(portal, token, filter, page),
    buildPath(filter, page),
    enabled,
  )
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
      const params = filterParams(filter).toString()
      const address = params ? `${PATH}/ekspor?${params}` : `${PATH}/ekspor`

      await downloadExport({
        address,
        token,
        portal,
        fallbackName: 'klaim-treaty-non-prop.csv',
      })
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
