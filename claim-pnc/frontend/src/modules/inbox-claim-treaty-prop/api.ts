import { useQueueList, useScreenMetadata } from '@/api/inboxShared'

import type { FilterForm, ListResponse, MetadataResponse } from './types'

const PATH = '/api/inbox-claim-treaty-prop'

/**
 * Kunci cache dikumpulkan di satu tempat supaya invalidasi tidak salah sasaran.
 *
 * Portal ikut menjadi bagian kunci, dan itu BUKAN kerapian: antrean kerja satu badan hukum
 * bukan antrean badan hukum lain, dan menyimpan keduanya di bawah satu kunci akan membuat
 * perpindahan portal menampilkan pekerjaan entitas sebelumnya (`R-20`).
 */
const keys = {
  metadata: (portal: string | null, token: string | null) =>
    ['inbox-claim-treaty-prop', 'tab', portal, token] as const,

  list: (portal: string | null, token: string | null, filter: FilterForm, page: number) =>
    [
      'inbox-claim-treaty-prop',
      'daftar',
      portal,
      token,
      filter.tab,
      filter.lihatSemua,
      page,
    ] as const,
}

/**
 * Hook keterangan layar — daftar tab, kolomnya, dan selisih terencana yang berlaku.
 *
 * # Kenapa bentuk layar datang dari server
 *
 * Karena kolom tiap tab adalah HASIL PEMBACAAN export Pega, dan tempat pembacaan itu
 * tercatat adalah backend (`internal/inboxclaimtreatyprop/tab.go`). Menyalinnya ke layar
 * berarti daftar yang sama hidup di dua tempat, dan yang satu akan tertinggal saat yang
 * lain diperbaiki.
 *
 * Keadaan "tab terhalang" ikut datang dari sana dengan alasan yang sama — begitu DBA
 * mengirim DDL yang dibutuhkan tab Komite, penghalangnya hilang tanpa menyunting frontend.
 */
export function useClaimTreatyPropMetadata() {
  return useScreenMetadata<MetadataResponse>(keys.metadata, `${PATH}/tab`)
}

/**
 * Hook isi satu tab.
 *
 * # Kenapa penyaringan dan paginasi dikerjakan di SERVER
 *
 * Karena yang dibaca adalah tabel penugasan Pega yang berisi puluhan juta baris (`D-10`),
 * digabungkan ke `POOLDATA.JSON_KLAIM`. Menyaringnya di peramban berarti menarik seluruh
 * antrean lebih dulu.
 *
 * Berbeda dengan Inbox Admin, halaman di sini benar-benar dipotong basis data dengan
 * `OFFSET … FETCH NEXT`, sehingga jumlah "total" tetap tepat tanpa menarik seluruh baris.
 */
export function useClaimTreatyPropList(filter: FilterForm, page: number, enabled: boolean) {
  return useQueueList<ListResponse>(
    (portal, token) => keys.list(portal, token, filter, page),
    buildPath(filter, page),
    enabled,
  )
}

/**
 * buildPath menyusun parameter permintaan.
 *
 * Isian yang kosong TIDAK dikirim, bukan dikirim sebagai teks kosong: server membedakan
 * "tidak dikirim" dari "dikirim kosong", dan pada `tab` yang kedua akan menjadi tab bawaan
 * sementara yang pertama memang itu yang diinginkan.
 *
 * `lihat_semua` hanya dikirim saat BENAR. Mengirim `lihat_semua=0` tidak salah, tetapi ia
 * membuat dua permintaan yang hasilnya sama punya URL berbeda — dan URL berbeda berarti
 * cache terpisah untuk hasil yang sama.
 */
function buildPath(filter: FilterForm, page: number): string {
  const params = new URLSearchParams()

  if (filter.tab) params.set('tab', filter.tab)
  if (filter.lihatSemua) params.set('lihat_semua', '1')
  if (page > 1) params.set('halaman', String(page))

  const query = params.toString()
  return query ? `${PATH}?${query}` : PATH
}
