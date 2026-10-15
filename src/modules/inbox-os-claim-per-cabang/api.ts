import { useMutation, useQuery } from '@tanstack/react-query'

import { callAPI, HEADER_PORTAL } from '@/api/client'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import type { DetailResponse, ListResponse, SummaryResponse } from './types'

const PATH = '/api/inbox-os-claim-per-cabang'

/**
 * Kunci cache dikumpulkan di satu tempat supaya invalidasi tidak salah sasaran.
 *
 * Portal ikut menjadi bagian kunci, dan itu BUKAN kerapian: klaim satu badan hukum bukan
 * klaim badan hukum lain, dan menyimpan keduanya di bawah satu kunci akan membuat
 * perpindahan portal menampilkan klaim entitas sebelumnya (`R-20`).
 *
 * Token ikut pula, dan di layar ini alasannya lebih tajam daripada biasanya: batas datanya
 * adalah CABANG PEMANGGIL. Dua pengguna dari dua cabang yang memakai peramban yang sama akan
 * melihat daftar yang berbeda, dan tanpa token di dalam kunci yang kedua akan melihat
 * sisa daftar milik yang pertama.
 */
const keys = {
  list: (portal: string | null, token: string | null, page: number, search: string) =>
    ['inbox-os-claim-per-cabang', 'daftar', portal, token, page, search] as const,
  detail: (portal: string | null, token: string | null, nomor: string | null) =>
    ['inbox-os-claim-per-cabang', 'detail', portal, token, nomor] as const,
  summary: (portal: string | null, token: string | null) =>
    ['inbox-os-claim-per-cabang', 'ringkasan', portal, token] as const,
}

/**
 * Hook isi layar.
 *
 * # Tidak ada satu pun penyaring, dan itu disengaja
 *
 * Layar lamanya memang tidak punya. `GetDataOutstandingperCabang` menyaring dua hal saja —
 * `registerdate IS NOT NULL` yang tetap, dan kode cabang pemanggil — dan sectionnya tidak
 * memuat satu pun kotak cari maupun dropdown. Menambahkan penyaring di sini berarti
 * menambah kemampuan yang tidak pernah ada, dan pada layar yang sedang diuji
 * kesetaraannya, kemampuan tambahan adalah selisih yang harus dipertanggungjawabkan.
 *
 * # Paginasi dikerjakan SERVER
 *
 * Layar lama menarik seluruh baris cabang sekaligus ke klipboard Pega. Dengan data historis
 * puluhan juta baris (`D-10`) itu bukan pola yang dibawa; halaman di sini benar-benar
 * dipotong basis data dengan `OFFSET … FETCH NEXT`.
 */
export function useOSClaimPerCabangList(page: number, search = '') {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  const cari = search.trim()

  return useQuery({
    queryKey: keys.list(portal, token, page, cari),
    queryFn: () => callAPI<ListResponse>(buildPath(page, cari), { token, portal }),
    enabled: token !== null && portal !== null,

    // Hasil sebelumnya ditahan selama halaman berikutnya dimuat, alih-alih tabel berkedip
    // menjadi kosong lalu terisi lagi.
    placeholderData: (previous) => previous,

    // Daftar berubah saat petugas lain memperbarui progres klaim, jadi cache-nya pendek —
    // layar ini dibuka berulang kali sepanjang hari.
    staleTime: 15 * 1000,
  })
}

/**
 * Hook tombol "Export To Excel".
 *
 * # Kenapa berkasnya diambil dengan fetch, bukan dengan tautan unduh biasa
 *
 * Karena `<a href>` dan `window.open` TIDAK membawa header — dan endpoint ini menuntut dua:
 * `Authorization` dan `X-Portal`. Satu-satunya cara memakai tautan biasa adalah menaruh
 * token di dalam alamat, dan nilai di URL ikut tercatat di riwayat peramban, log proxy, dan
 * header Referer. Berkas ini memuat nomor polis, nama tertanggung, dan nilai uang; jejaknya
 * tidak boleh tertinggal di sana.
 *
 * # Biaya yang disadari
 *
 * Peladen MENGALIRKAN berkasnya potong demi potong, tetapi peramban menampungnya utuh
 * sebagai Blob sebelum menyimpannya. Manfaat pengaliran karena itu tinggal di sisi peladen —
 * memorinya tetap datar — sedangkan memori peramban tumbuh sebesar berkasnya. Pada batas
 * 50.000 baris dengan 48 kolom itu beberapa megabita, dan dapat diterima.
 */
export function useExportOSClaimPerCabang() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useMutation({
    mutationFn: async () => {
      const header: Record<string, string> = {}
      if (token) header['Authorization'] = `Bearer ${token}`
      if (portal) header[HEADER_PORTAL] = portal

      const response = await fetch(`${PATH}/ekspor`, { headers: header })
      if (!response.ok) {
        // Galat dijawab sebagai JSON selama header belum terkirim; setelah itu tidak bisa
        // lagi. Yang dibaca di sini adalah kasus pertama, dan cabang yang tidak diketahui
        // termasuk di dalamnya — pesannya diteruskan apa adanya ke pengguna.
        const body = (await response.json().catch(() => null)) as { pesan?: string } | null
        throw new Error(body?.pesan ?? 'Berkas ekspor tidak dapat diambil.')
      }

      const blob = await response.blob()
      downloadBlob(blob, filenameOf(response) ?? 'SummaryOS.csv')
    },
  })
}

/**
 * buildPath menyusun alamat permintaan daftar beserta halaman dan kotak carinya.
 *
 * Keduanya ditulis lewat `URLSearchParams`, bukan dirangkai sendiri: nomor polis dapat memuat
 * `&`, `#`, atau spasi, dan perangkaian tangan akan memotong pencariannya di karakter itu
 * tanpa satu pun galat.
 *
 * Parameter yang kosong TIDAK dikirim, supaya alamat tanpa penyaring tetap sama persis
 * seperti sebelumnya — cache peladen dan pembacaan log tidak berubah karenanya.
 */
function buildPath(page: number, search: string): string {
  const params = new URLSearchParams()
  if (page > 1) params.set('halaman', String(page))
  if (search) params.set('cari', search)

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
  document.body.removeChild(link)
  URL.revokeObjectURL(url)
}

/**
 * Hook isi popup Detail.
 *
 * # Ia hanya mengirim NOMOR klaim
 *
 * Tombol Detail di Pega mengirim lima parameter dari baris yang diklik — nomor, nilai
 * cadangan, umur, lini bisnis, dan catatan PIC. Di sini hanya nomornya; empat sisanya dibaca
 * ulang peladen.
 *
 * Perbedaannya bukan kerapian. Mengirim nilai cadangan dari sini berarti angka uang yang
 * tampil di popup ditentukan peramban, dan itu dapat diubah lewat alat pengembang biasa.
 *
 * # `enabled` menunggu nomornya ada
 *
 * Popup baru diminta setelah barisnya diklik. Tanpa penjaga itu, layar akan menembak endpoint
 * dengan nomor kosong setiap kali dibuka — dan dijawab "klaim tidak ditemukan" yang tidak
 * pernah dilihat siapa pun tetapi tetap mengisi log peladen.
 */
export function useOSClaimPerCabangDetail(nomorKlaim: string | null) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: keys.detail(portal, token, nomorKlaim),
    queryFn: () =>
      callAPI<DetailResponse>(`${PATH}/${encodeURIComponent(nomorKlaim ?? '')}`, {
        token,
        portal,
      }),
    enabled: token !== null && portal !== null && nomorKlaim !== null,

    // Isi popup jarang berubah selama satu sesi membaca, dan popup yang sama sering dibuka
    // ulang bolak-balik dari daftar.
    staleTime: 30 * 1000,
  })
}

/**
 * Hook panel ringkasan.
 *
 * Permintaan TERPISAH dari daftar, dan itu disengaja: panel tidak berubah saat pengguna
 * berpindah halaman, sehingga menyatukannya berarti menghitung ulang seluruh ringkasan
 * cabang pada setiap klik halaman. Terpisah pula kegagalannya — panel yang gagal dimuat
 * tidak mengosongkan grid di bawahnya.
 */
export function useOSClaimPerCabangSummary() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: keys.summary(portal, token),
    queryFn: () => callAPI<SummaryResponse>(`${PATH}/ringkasan`, { token, portal }),
    enabled: token !== null && portal !== null,
    staleTime: 15 * 1000,
  })
}
