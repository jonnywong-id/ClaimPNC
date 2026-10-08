import { usePersonalTaskList, useScreenMetadata } from '@/api/inboxShared'

import type { DaftarResponse, KeteranganResponse } from './types'

const PATH = '/api/inbox-rcl'

/** Banyaknya baris per halaman. Backend menolak permintaan di atas 100. */
export const PAGE_SIZE = 25

/**
 * Kunci cache. PORTAL dan TOKEN ikut menjadi bagian kunci: dua portal adalah dua badan
 * hukum (`ADR-0030`, `R-20`), dan antreannya disaring dengan identitas pemanggil — cache
 * yang bertahan melewati pergantian pengguna akan menampilkan tugas dokter sebelumnya.
 */
const keys = {
  keterangan: (portal: string | null, token: string | null) =>
    ['inbox-rcl', 'keterangan', portal, token] as const,

  daftar: (portal: string | null, token: string | null, cari: string, lewati: number) =>
    ['inbox-rcl', 'daftar', portal, token, cari, lewati] as const,
}

/**
 * Hook keterangan layar. Judul kolom datang dari server karena ia hasil pembacaan
 * `Harness/RCL_Harness-Harness.xml` yang tercatat di backend.
 */
export function useKeteranganRCL() {
  return useScreenMetadata<KeteranganResponse>(keys.keterangan, `${PATH}/keterangan`)
}

/**
 * Hook antrean RCL Dokter milik pengguna yang sedang masuk.
 *
 * Menggantikan `Report Definition/InboxRCLDokter_RD-RD.xml`. Layar tidak mengirim identitas
 * apa pun — server membacanya dari sesi lalu mencari identitas LAMA-nya sendiri.
 */
export function useDaftarRCL(cari: string, lewati: number) {
  return usePersonalTaskList<DaftarResponse>(
    (portal, token) => keys.daftar(portal, token, cari.trim(), lewati),
    PATH,
    cari,
    lewati,
    PAGE_SIZE,
  )
}
