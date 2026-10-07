import { useMutation, useQuery } from '@tanstack/react-query'

import { callAPI, HEADER_PORTAL } from '@/api/client'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

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
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: keys.metadata(portal, token),
    queryFn: () => callAPI<MetadataResponse>(`${PATH}/tab`, { token, portal }),
    enabled: token !== null && portal !== null,

    // Lima menit: cukup lama untuk tidak diambil ulang setiap perpindahan tab, cukup
    // pendek untuk tidak menahan perubahan jabatan sepanjang hari.
    staleTime: 5 * 60 * 1000,
  })
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
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: keys.list(portal, token, tab, page),
    queryFn: () => callAPI<ListResponse>(buildPath(tab, page), { token, portal }),
    enabled: enabled && token !== null && portal !== null,

    // Hasil sebelumnya ditahan selama halaman berikutnya dimuat, alih-alih tabel berkedip
    // menjadi kosong lalu terisi lagi.
    placeholderData: (previous) => previous,

    // Antrean berubah saat petugas lain menyelesaikan pekerjaannya, jadi cache-nya pendek.
    staleTime: 15 * 1000,
  })
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
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useMutation({
    mutationFn: async (tab: string) => {
      const header: Record<string, string> = {}
      if (token) header['Authorization'] = `Bearer ${token}`
      if (portal) header[HEADER_PORTAL] = portal

      const params = tabParams(tab).toString()
      const address = params ? `${PATH}/ekspor?${params}` : `${PATH}/ekspor`

      const response = await fetch(address, { headers: header })
      if (!response.ok) {
        // Galat dijawab sebagai JSON selama header belum terkirim; setelah itu tidak bisa
        // lagi. Yang dibaca di sini adalah kasus pertama.
        const body = (await response.json().catch(() => null)) as { pesan?: string } | null
        throw new Error(body?.pesan ?? 'Berkas ekspor tidak dapat diambil.')
      }

      const blob = await response.blob()
      downloadBlob(blob, filenameOf(response) ?? 'inbox-manager-admin.csv')
    },
  })
}

/**
 * tabParams menyusun satu-satunya isian penyaring layar ini.
 *
 * Ia dipakai daftar DAN ekspor, supaya keduanya tidak dapat membaca tab dengan cara yang
 * berbeda.
 *
 * Tab yang kosong TIDAK dikirim, bukan dikirim sebagai teks kosong: server membedakan
 * "tidak dikirim" dari "dikirim kosong", dan yang pertama berarti tab pertama yang boleh
 * dilihat pemanggil.
 */
function tabParams(tab: string): URLSearchParams {
  const params = new URLSearchParams()
  if (tab) params.set('tab', tab)
  return params
}

/** buildPath menyusun alamat permintaan daftar beserta halamannya. */
function buildPath(tab: string, page: number): string {
  const params = tabParams(tab)
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
 * URL objeknya DICABUT setelah dipakai. Tanpa itu, blob-nya tetap dipegang peramban sampai
 * tab ditutup — dan pada layar yang dipakai sepanjang hari, setiap ekspor menumpuk memori
 * yang tidak pernah dilepas.
 */
function downloadBlob(blob: Blob, filename: string): void {
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = filename
  document.body.appendChild(link)
  link.click()
  link.remove()
  URL.revokeObjectURL(url)
}
