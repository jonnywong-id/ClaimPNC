import { queryPath, usePortalQuery, useQueueList, useTabExport } from '@/api/inboxShared'

import type { ListResponse, MetadataResponse } from './types'

const PATH = '/api/inbox-manager-admin'

/**
 * Kunci cache dikumpulkan di satu tempat supaya invalidasi tidak salah sasaran.
 *
 * Portal ikut menjadi bagian kunci, dan itu BUKAN kerapian: antrean satu badan hukum bukan
 * antrean badan hukum lain, dan menyimpan keduanya di bawah satu kunci akan membuat
 * perpindahan portal menampilkan pekerjaan entitas sebelumnya (`R-20`).
 *
 * Kunci keterangan layar ikut memuat token, dan di modul ini alasannya lebih kuat daripada
 * sekadar kebiasaan: jawabannya BERBEDA menurut pengguna — yang dikirim hanyalah tab yang
 * boleh ia lihat. Berbagi satu entri antar pengguna berarti menampilkan tab yang bukan
 * haknya, lalu ditolak server saat ia mengkliknya.
 */
const keys = {
  metadata: (portal: string | null, token: string | null) =>
    ['inbox-manager-admin', 'tab', portal, token] as const,

  list: (portal: string | null, token: string | null, tab: string, page: number) =>
    ['inbox-manager-admin', 'daftar', portal, token, tab, page] as const,
}

/**
 * Hook keterangan layar — tab yang boleh dilihat, kolomnya, dan selisih terencana.
 *
 * # Kenapa bentuk layar datang dari server
 *
 * Karena kolom tiap tab adalah HASIL PEMBACAAN export Pega, dan tempat pembacaan itu
 * tercatat adalah backend (`internal/inboxmanageradmin/tab.go`). Menyalinnya ke layar
 * berarti daftar yang sama hidup di dua tempat, dan yang satu akan tertinggal saat yang lain
 * diperbaiki.
 *
 * Di layar ini ada alasan kedua yang lebih menentukan: kewenangan tab pun datang dari sana.
 * Menyusunnya di layar berarti penyembunyian tab menjadi satu-satunya kendali — persis cara
 * Pega bekerja (`pyContainerVisibleWhen`), dan persis yang `11-SECURITY.md` §3.1 sebut
 * sebagai kenyamanan tampilan, bukan kendali.
 *
 * # Kenapa TIDAK di-cache selamanya, berbeda dari modul inbox lain
 *
 * Karena isinya bergantung pada jabatan pengguna, dan jabatan itu datang dari sistem
 * kepegawaian yang dapat berubah di luar aplikasi ini. Menyimpannya selamanya berarti
 * perubahan jabatan baru terlihat setelah pengguna menutup peramban.
 */
export function useInboxManagerAdminMetadata() {
  // Lima menit: cukup lama untuk tidak diambil ulang setiap perpindahan tab, cukup
  // pendek untuk tidak menahan perubahan jabatan sepanjang hari.
  return usePortalQuery<MetadataResponse>(keys.metadata, `${PATH}/tab`, true, 5 * 60 * 1000)
}

/**
 * Hook isi satu tab.
 *
 * # Kenapa paginasi dikerjakan di SERVER, meski server menariknya seluruhnya
 *
 * Keputusan Work Owner 2026-09-26: paginasi layar ini direplikasi apa adanya dari Pega,
 * sehingga server menarik seluruh baris yang cocok lalu memotong halamannya. Angka
 * totalnya karena itu selalu tepat.
 *
 * Yang TIDAK boleh disimpulkan dari situ: bahwa layar boleh menyaring sendiri. Yang dibaca
 * adalah `DATAPEGA.PC_ASM_FW_GCNMFW_WORK` yang berisi puluhan juta baris (`D-10`) — yang
 * sampai ke peramban hanyalah satu halaman, dan memang harus begitu.
 */
export function useInboxManagerAdminList(tab: string, page: number, enabled: boolean) {
  return useQueueList<ListResponse>(
    (portal, token) => keys.list(portal, token, tab, page),
    buildPath(tab, page),
    enabled,
  )
}

/**
 * Hook tombol "Export To Excel".
 *
 * # Berkasnya CSV, meski tombolnya berbunyi Excel
 *
 * Karena begitu pula sistem lama: `ExportExcelManagerAdminPA` memanggil
 * `pxConvertResultsToCSV`, bukan penulis Excel mana pun. Judul tombolnya dipertahankan apa
 * adanya (`D-13`).
 *
 * # Isinya TIDAK sama dengan yang di layar, dan itu disengaja
 *
 * Berkas memuat delapan kolom apa adanya dari `HeadersMap` activity lama — termasuk
 * **Status Klaim** yang tidak ada di grid, dan tanpa **Lama Waktu Klaim** yang justru ada di
 * grid. Keputusan Work Owner 2026-09-26. Ia dinyatakan ke pengguna lewat panel selisih
 * terencana di bawah tabel, supaya tidak dilaporkan sebagai kerusakan.
 *
 * # Biaya yang disadari
 *
 * Peladen MENGALIRKAN berkasnya potong demi potong, tetapi peramban menampungnya utuh
 * sebagai Blob sebelum menyimpannya. Manfaat pengaliran karena itu tinggal di sisi peladen —
 * memorinya tetap datar — sedangkan memori peramban tumbuh sebesar berkasnya. Pada batas
 * 50.000 baris itu beberapa megabita, dan dapat diterima.
 */
export function useExportInboxManagerAdmin() {
  return useTabExport(PATH, 'inbox-manager-admin.csv')
}

/**
 * buildPath menyusun alamat permintaan daftar beserta halamannya.
 *
 * Tab yang kosong TIDAK dikirim, bukan dikirim sebagai teks kosong: server membedakan
 * "tidak dikirim" dari "dikirim kosong". Ekspor membaca tab dengan aturan yang sama
 * (`useTabExport`).
 */
function buildPath(tab: string, page: number): string {
  return queryPath(PATH, [
    ['tab', tab],
    ['halaman', page > 1 && String(page)],
  ])
}
