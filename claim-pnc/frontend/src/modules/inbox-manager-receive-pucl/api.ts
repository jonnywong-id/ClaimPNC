import { useMutation, useQuery } from '@tanstack/react-query'

import { callAPI } from '@/api/client'
import { queryPath, useQueueList, useScreenMetadata, useTabExport } from '@/api/inboxShared'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import type { DocumentResponse, ListResponse, MetadataResponse } from './types'

const PATH = '/api/inbox-manager-receive-pucl'

/**
 * Kunci cache dikumpulkan di satu tempat supaya invalidasi tidak salah sasaran.
 *
 * Portal ikut menjadi bagian kunci, dan itu BUKAN kerapian: antrean kerja satu badan hukum
 * bukan antrean badan hukum lain, dan menyimpan keduanya di bawah satu kunci akan membuat
 * perpindahan portal menampilkan pekerjaan entitas sebelumnya (`R-20`).
 *
 * Kode tab ikut pula. Ketiga tab dilayani KUERI yang berbeda di server — bukan satu kueri
 * yang hasilnya disaring — sehingga hasilnya tidak boleh berbagi satu entri cache.
 */
const keys = {
  metadata: (portal: string | null, token: string | null) =>
    ['inbox-manager-receive-pucl', 'tab', portal, token] as const,

  list: (portal: string | null, token: string | null, tab: string, page: number) =>
    ['inbox-manager-receive-pucl', 'daftar', portal, token, tab, page] as const,

  document: (portal: string | null, token: string | null, reference: string) =>
    ['inbox-manager-receive-pucl', 'dokumen', portal, token, reference] as const,
}

/**
 * Hook keterangan layar — daftar tab, kolomnya, dan selisih terencana yang berlaku.
 *
 * # Kenapa bentuk layar datang dari server
 *
 * Karena kolom tiap tab adalah HASIL PEMBACAAN export Pega, dan tempat pembacaan itu
 * tercatat adalah backend (`internal/inboxmanagerreceivepucl/tab.go`). Menyalinnya ke layar
 * berarti daftar yang sama hidup di dua tempat, dan yang satu akan tertinggal saat yang lain
 * diperbaiki.
 *
 * Di layar ini hal itu lebih dari sekadar kerapian: kedua tab Receive berbagi SEMBILAN kolom
 * yang identik, dan satu perbedaan kecil di antara keduanya akan berarti salah satunya tidak
 * lagi mencerminkan grid aslinya.
 */
export function useManagerReceivePUCLMetadata() {
  return useScreenMetadata<MetadataResponse>(keys.metadata, `${PATH}/tab`)
}

/**
 * Hook isi satu tab.
 *
 * # Kenapa paginasinya dikerjakan SERVER
 *
 * Karena yang dibaca adalah tabel objek kerja dan tabel penugasan Pega yang berisi puluhan
 * juta baris (`D-10`). Menyaringnya di peramban berarti menarik seluruh antrean lebih dulu.
 *
 * Halaman di sini benar-benar dipotong basis data dengan `OFFSET … FETCH NEXT`, sehingga
 * jumlah "total" tetap tepat tanpa menarik seluruh baris.
 */
export function useManagerReceivePUCLList(tab: string, page: number, enabled: boolean) {
  return useQueueList<ListResponse>(
    (portal, token) => keys.list(portal, token, tab, page),
    buildPath(tab, page),
    enabled,
  )
}

/**
 * Hook layar kerja penerimaan dokumen — flow action `InputReceiveDocument`.
 *
 * # Kenapa kuncinya dikodekan
 *
 * Karena `pzInsKey` memuat SPASI — `ASM-FW-GCNMFW-WORK RCV-900001` — dan spasi mentah di
 * dalam alamat bukan alamat yang sah. Ia dikodekan di sini, bukan diserahkan ke pemanggil,
 * supaya tidak ada satu pun tempat yang lupa melakukannya.
 *
 * # Kenapa cache-nya pendek
 *
 * Karena berkas ini SEDANG DIKERJAKAN — di Pega, oleh petugas lain, saat layar ini terbuka.
 * Cache panjang akan membuat petugas membaca kronologi kejadian yang sudah berubah, lalu
 * mengerjakan berkasnya di Pega berdasarkan isi yang lama.
 */
export function useReceiveDocument(reference: string, enabled = true) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: keys.document(portal, token, reference),
    queryFn: () =>
      callAPI<DocumentResponse>(
        `${PATH}/dokumen/${encodeURIComponent(reference)}`,
        { token, portal },
      ),
    enabled: enabled && reference !== '' && token !== null && portal !== null,
    staleTime: 15 * 1000,
  })
}

/**
 * Hook kedelapan tombol layar kerja yang di Pega MENGUBAH data.
 *
 * # Kenapa tombolnya DITEKAN ke server, bukan dijawab layar sendiri
 *
 * Karena alasan mengapa sebuah tombol belum dapat dijalankan adalah keadaan SISTEM, bukan
 * keadaan layar — dan ia berubah begitu kemampuannya dibangun. Menuliskan alasannya di layar
 * berarti alasan yang sama hidup di dua tempat, dan yang di layar akan tertinggal pada hari
 * tombolnya mulai bekerja.
 *
 * Ada alasan kedua yang sama pentingnya: penekanannya DICATAT di sisi peladen. Selama masa
 * paralel, jejak itulah satu-satunya tanda seberapa sering tombol-tombol ini benar-benar
 * dibutuhkan — dan itu yang menjadi dasar memutuskan mana yang dibangun lebih dulu.
 */
export function useReceiveDocumentAction() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useMutation({
    mutationFn: (input: { tindakan: string; berkas: string }) => {
      const query = new URLSearchParams({ tindakan: input.tindakan })
      if (input.berkas) query.set('berkas', input.berkas)

      return callAPI<void>(`${PATH}/tindakan?${query.toString()}`, {
        token,
        portal,
        metode: 'POST',
      })
    },
  })
}

/**
 * Hook tombol ekspor.
 *
 * # Kenapa berkasnya diambil dengan fetch, bukan dengan tautan unduh biasa
 *
 * Karena `<a href>` dan `window.open` TIDAK membawa header — dan endpoint ini menuntut dua:
 * `Authorization` dan `X-Portal`. Satu-satunya cara memakai tautan biasa adalah menaruh
 * token di dalam alamat, dan itu ditolak dengan alasan yang sudah dicatat di
 * `api/client.ts`: nilai di URL ikut tercatat di riwayat peramban, log proxy, dan header
 * Referer. Berkas ini memuat nomor polis dan nama tertanggung; jejaknya tidak boleh
 * tertinggal di sana.
 *
 * # Isinya mengikuti TAB yang sedang terbuka
 *
 * Ketiga tab punya kolom yang berbeda, sehingga berkasnya pun berbeda susunannya. Tab yang
 * sedang dilihat dikirim ke endpoint ekspor persis seperti ke endpoint daftar — berkas yang
 * isinya tidak dapat dicocokkan dengan layar adalah berkas yang menyesatkan.
 *
 * # Biaya yang disadari
 *
 * Peladen MENGALIRKAN berkasnya potong demi potong, tetapi peramban menampungnya utuh
 * sebagai Blob sebelum menyimpannya. Manfaat pengaliran karena itu tinggal di sisi peladen —
 * memorinya tetap datar — sedangkan memori peramban tumbuh sebesar berkasnya. Pada batas
 * 50.000 baris itu beberapa megabita, dan dapat diterima.
 */
export function useExportManagerReceivePUCL() {
  return useTabExport(PATH, 'inbox-manager-receive-pucl.csv')
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
