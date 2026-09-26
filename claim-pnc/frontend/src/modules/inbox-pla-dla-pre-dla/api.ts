import { useMutation, useQuery } from '@tanstack/react-query'

import { callAPI, downloadAPI } from '@/api/client'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import type { DokumenResponse, ListResponse, MetadataResponse } from './types'

const PATH = '/api/inbox-pla-dla-pre-dla'

/**
 * Kunci cache dikumpulkan di satu tempat supaya invalidasi tidak salah sasaran.
 *
 * Portal ikut menjadi bagian kunci, dan itu BUKAN kerapian: pemberitahuan reasuransi satu
 * badan hukum bukan milik badan hukum lain, dan menyimpan keduanya di bawah satu kunci
 * akan membuat perpindahan portal menampilkan antrean entitas sebelumnya (`R-20`).
 *
 * Seluruh penyaring ikut pula — daftar, halaman, kata kunci, dan kedua batas tanggal.
 * Semuanya menyaring DI SERVER, sehingga hasil dengan penyaring berbeda tidak boleh
 * berbagi satu entri cache.
 */
const keys = {
  metadata: (portal: string | null, token: string | null) =>
    ['inbox-pla-dla-pre-dla', 'daftar', portal, token] as const,

  list: (
    portal: string | null,
    token: string | null,
    tab: string,
    page: number,
    search: string,
    from: string,
    to: string,
  ) =>
    [
      'inbox-pla-dla-pre-dla',
      'baris',
      portal,
      token,
      tab,
      page,
      search,
      from,
      to,
    ] as const,

  documents: (
    portal: string | null,
    token: string | null,
    tab: string,
    claimKey: string,
  ) =>
    ['inbox-pla-dla-pre-dla', 'rincian', portal, token, tab, claimKey] as const,
}

/**
 * Hook keterangan layar — daftar tab, kolomnya, dan selisih terencana.
 *
 * # Kenapa bentuk layar datang dari server
 *
 * Karena kolom tiap daftar adalah HASIL PEMBACAAN export Pega, dan tempat pembacaan itu
 * tercatat adalah backend (`internal/inboxpladlapredla/tab.go`). Menyalinnya ke layar
 * berarti daftar yang sama hidup di dua tempat, dan yang satu akan tertinggal saat yang
 * lain diperbaiki.
 *
 * Di layar ini ada satu hal lagi yang harus datang dari sana: `punya_rincian`. Tab Pre DLA
 * tidak menggambar grid rincian, dan itu temuan dari membaca section-nya — bukan pilihan
 * tampilan yang boleh diputuskan layar.
 */
export function usePLADLAMetadata() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: keys.metadata(portal, token),
    queryFn: () => callAPI<MetadataResponse>(`${PATH}/daftar`, { token, portal }),
    enabled: token !== null && portal !== null,

    // Bentuk layar tidak berubah selama aplikasi berjalan: ia dibaca dari kode, bukan
    // dari data.
    staleTime: Infinity,
    gcTime: Infinity,
  })
}

/** Isian penyaring yang dikirim ke server. */
export type ParameterDaftar = {
  tab: string
  page: number
  cari: string
  dari: string
  sampai: string
}

/**
 * Hook isi satu antrean.
 *
 * # Kenapa paginasi DAN penyaring dikerjakan SERVER
 *
 * Karena yang dibaca adalah tabel klaim berisi puluhan juta baris (`D-10`). Menyaringnya
 * di peramban berarti menarik seluruh antrean lebih dulu.
 *
 * Dan ada alasan kedua yang khas layar ini: rentang tanggal menyaring tabel DOKUMEN, bukan
 * baris klaim yang sudah jadi. Baris yang sudah sampai ke peramban tidak lagi membawa
 * tanggal dokumen mana pun selain yang terbaru, sehingga penyaringan di sisi peramban
 * tidak mungkin menghasilkan jawaban yang sama.
 */
export function usePLADLAList(parameter: ParameterDaftar, enabled: boolean) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: keys.list(
      portal,
      token,
      parameter.tab,
      parameter.page,
      parameter.cari,
      parameter.dari,
      parameter.sampai,
    ),
    queryFn: () =>
      callAPI<ListResponse>(alamatDaftar(parameter), { token, portal }),
    enabled: enabled && token !== null && portal !== null,

    // Hasil sebelumnya ditahan selama halaman berikutnya dimuat, alih-alih tabel berkedip
    // menjadi kosong lalu terisi lagi.
    placeholderData: (previous) => previous,

    // Antrean berubah saat petugas lain mengirim dokumennya, jadi cache-nya pendek —
    // sama dengan modul inbox lain, yang dibuka berulang kali sepanjang hari.
    staleTime: 15 * 1000,
  })
}

/**
 * Hook grid rincian satu klaim.
 *
 * # Kenapa baru ditembak saat panelnya dibuka
 *
 * `kunci_klaim` kosong berarti panelnya tertutup, dan permintaan tidak dijalankan.
 * Mengambil rinciannya untuk setiap baris yang tampil akan menjalankan satu kueri per
 * baris untuk panel yang mungkin tidak pernah dibuka.
 *
 * # Kenapa TIDAK disimpan lama di cache
 *
 * Isinya berubah karena perbuatan orang lain — petugas lain mengirim dokumennya — dan
 * panel yang menampilkan keadaan lama akan membuat petugas mengirim ulang sesuatu yang
 * sudah terkirim.
 */
export function usePLADLADokumen(tab: string, claimKey: string) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  // Kunci klaim memuat SPASI (`ASM-FW-GCNMFW-WORK PNC-xxxx`). Tanpa pengodean, alamatnya
  // terpotong di spasi itu dan yang sampai ke server hanya awalannya — kunci yang tidak
  // pernah cocok dengan klaim mana pun.
  const path =
    `${PATH}/klaim/${encodeURIComponent(claimKey)}` +
    `?daftar=${encodeURIComponent(tab)}`

  return useQuery({
    queryKey: keys.documents(portal, token, tab, claimKey),
    queryFn: () => callAPI<DokumenResponse>(path, { token, portal }),
    enabled: token !== null && portal !== null && claimKey !== '',
    staleTime: 0,
  })
}

/**
 * Hook tombol "Export To Excel".
 *
 * Berkasnya diambil lewat `downloadAPI`, bukan lewat tautan biasa: endpoint ini menuntut
 * header `Authorization` dan `X-Portal`, dan `<a href>` tidak membawa keduanya.
 * Menaruh token di alamat sebagai gantinya ditolak — nilai di URL ikut tercatat di riwayat
 * peramban, log proxy, dan header Referer, sementara berkas ini memuat nama tertanggung
 * dan nomor polis.
 */
export function useEksporPLADLA() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useMutation({
    mutationFn: (parameter: ParameterDaftar) =>
      downloadAPI(
        alamatEkspor(parameter),
        `inbox-${parameter.tab || 'pla'}.csv`,
        { token, portal },
      ),
  })
}

/** alamatDaftar menyusun alamat permintaan daftar beserta seluruh penyaringnya. */
function alamatDaftar(parameter: ParameterDaftar): string {
  const query = parameterPenyaring(parameter)
  if (parameter.page > 1) query.set('halaman', String(parameter.page))

  const teks = query.toString()
  return teks ? `${PATH}?${teks}` : PATH
}

/**
 * alamatEkspor menyusun alamat unduhan.
 *
 * Halaman sengaja TIDAK ikut: ekspor menyalin SELURUH daftar yang cocok, bukan halaman
 * yang sedang tampil. Menyertakan halaman akan menghasilkan berkas berisi sepuluh baris
 * dari antrean yang berisi ribuan — dan pengguna tidak akan menyadarinya sampai ia
 * mencocokkannya.
 */
function alamatEkspor(parameter: ParameterDaftar): string {
  const teks = parameterPenyaring(parameter).toString()
  return teks ? `${PATH}/ekspor?${teks}` : `${PATH}/ekspor`
}

/** parameterPenyaring menyusun bagian alamat yang sama bagi daftar dan ekspor. */
function parameterPenyaring(parameter: ParameterDaftar): URLSearchParams {
  const query = new URLSearchParams()
  if (parameter.tab) query.set('daftar', parameter.tab)
  if (parameter.cari) query.set('cari', parameter.cari)
  if (parameter.dari) query.set('dari', parameter.dari)
  if (parameter.sampai) query.set('sampai', parameter.sampai)
  return query
}
