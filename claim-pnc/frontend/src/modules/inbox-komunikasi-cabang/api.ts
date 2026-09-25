import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { callAPI, HEADER_PORTAL } from '@/api/client'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import type {
  ActionResponse,
  BranchListResponse,
  ConversationDetailResponse,
  ListResponse,
  MetadataResponse,
  NewMessageRequest,
} from './types'

const PATH = '/api/inbox-komunikasi-cabang'

/**
 * Kunci cache dikumpulkan di satu tempat supaya invalidasi tidak salah sasaran.
 *
 * Portal ikut menjadi bagian kunci, dan itu BUKAN kerapian: percakapan satu badan hukum
 * bukan percakapan badan hukum lain, dan menyimpan keduanya di bawah satu kunci akan membuat
 * perpindahan portal menampilkan percakapan entitas sebelumnya (`R-20`).
 *
 * Token ikut pula, dan di layar ini alasannya lebih tajam daripada di modul inbox lain:
 * batas datanya diturunkan dari LOGIN pemanggil. Pengguna yang berbeda melihat daftar yang
 * berbeda dari alamat yang sama persis — sehingga cache yang tidak memuat identitasnya akan
 * menampilkan percakapan cabang orang sebelumnya kepada orang berikutnya di komputer yang
 * sama.
 *
 * Kode tab ikut karena kedua tab dilayani KUERI yang berbeda di server — bukan satu kueri
 * yang hasilnya disaring — sehingga hasilnya tidak boleh berbagi satu entri cache.
 */
const keys = {
  metadata: (portal: string | null, token: string | null) =>
    ['inbox-komunikasi-cabang', 'tab', portal, token] as const,

  list: (portal: string | null, token: string | null, tab: string, page: number) =>
    ['inbox-komunikasi-cabang', 'daftar', portal, token, tab, page] as const,

  detail: (portal: string | null, token: string | null, id: string) =>
    ['inbox-komunikasi-cabang', 'komunikasi', portal, token, id] as const,

  branches: (portal: string | null, token: string | null) =>
    ['inbox-komunikasi-cabang', 'cabang', portal, token] as const,
}

/**
 * Hook keterangan layar — daftar tab, kolomnya, kolom unduhan, dan selisih terencana.
 *
 * # Kenapa bentuk layar datang dari server
 *
 * Karena kolom tiap tab adalah HASIL PEMBACAAN export Pega, dan tempat pembacaan itu
 * tercatat adalah backend (`internal/inboxkomunikasicabang/tab.go`). Menyalinnya ke layar
 * berarti daftar yang sama hidup di dua tempat, dan yang satu akan tertinggal saat yang lain
 * diperbaiki.
 *
 * Di layar ini hal itu lebih dari sekadar kerapian: kedua tab punya JUMLAH KOLOM yang
 * berbeda — tiga dan lima — dan perbedaan itu bukan pilihan tampilan melainkan akibat
 * langsung dari penyaringnya.
 */
export function useKomunikasiCabangMetadata() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: keys.metadata(portal, token),
    queryFn: () => callAPI<MetadataResponse>(`${PATH}/tab`, { token, portal }),
    enabled: token !== null && portal !== null,

    // Bentuk layar tidak berubah selama aplikasi berjalan: ia dibaca dari kode, bukan dari
    // data. Mengambilnya ulang tiap kali tab berpindah hanya menambah perjalanan jaringan
    // tanpa satu pun manfaat.
    staleTime: Infinity,
    gcTime: Infinity,
  })
}

/**
 * Hook isi satu tab, beserta kedua pencacah dan batas cabang yang berlaku.
 *
 * # Kenapa paginasinya dikerjakan SERVER
 *
 * Karena yang dibaca adalah tabel percakapan yang tumbuh terus (`D-10`). Menyaringnya di
 * peramban berarti menarik seluruh kotak percakapan lebih dulu.
 *
 * Halaman di sini benar-benar dipotong basis data dengan `OFFSET … FETCH NEXT`, sehingga
 * jumlah "total" tetap tepat tanpa menarik seluruh baris.
 *
 * # Kenapa pencacah datang BERSAMA daftar
 *
 * Karena keduanya digambar berdampingan, dan angka yang berasal dari dua saat yang berbeda
 * akan saling bertentangan di mata penggunanya — tabel menampilkan satu baris sementara
 * lencananya menyebut nol.
 */
export function useKomunikasiCabangList(tab: string, page: number, enabled: boolean) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: keys.list(portal, token, tab, page),
    queryFn: () => callAPI<ListResponse>(buildPath(tab, page), { token, portal }),
    enabled: enabled && token !== null && portal !== null,

    // Hasil sebelumnya ditahan selama halaman berikutnya dimuat, alih-alih tabel berkedip
    // menjadi kosong lalu terisi lagi.
    placeholderData: (previous) => previous,

    // Kotak percakapan berubah saat petugas lain membalas, jadi cache-nya pendek — sama
    // dengan modul inbox lain, yang dibuka berulang kali sepanjang hari.
    staleTime: 15 * 1000,
  })
}

/**
 * Hook layar "Detail Komunikasi" — yang di Pega terbuka lewat flow action
 * `DETAILKOMUNIKASICABANG_11`.
 *
 * # Kenapa ia mengambil data sendiri, bukan memakai baris yang sudah di tangan
 *
 * Karena isinya BERBEDA dari baris grid. Grid menampilkan SATU pesan beserta balasan
 * terakhirnya; layar detail menampilkan SELURUH utas percakapan beserta lampirannya, dan
 * tidak satu pun lampiran ada di daftar.
 *
 * # Kenapa cache-nya per NOMOR PERCAKAPAN
 *
 * Karena nomor itulah yang dikirim ke server. Portal dan token ikut menjadi bagian kunci
 * dengan alasan yang sama seperti hook lain — nomor percakapan tidak dijamin unik antar
 * entitas, dan batas cabangnya diturunkan dari login.
 */
export function useKomunikasiCabangDetail(id: string | null) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: keys.detail(portal, token, id ?? ''),
    queryFn: () =>
      callAPI<ConversationDetailResponse>(
        `${PATH}/komunikasi/${encodeURIComponent(id ?? '')}`,
        { token, portal },
      ),
    enabled: id !== null && id !== '' && token !== null && portal !== null,

    // Isi percakapan berubah saat petugas lain membalasnya di Pega — dan selama masa
    // paralel itulah satu-satunya tempat ia berubah. Cache-nya sependek daftar.
    staleTime: 15 * 1000,
  })
}

/**
 * Hook tombol "Balas".
 *
 * # Kenapa ia menyegarkan TIGA hal, bukan satu
 *
 * Karena satu balasan mengubah tiga hal sekaligus di layar: utas percakapannya, daftar yang
 * sedang terbuka (barisnya BERPINDAH tab), dan kedua pencacah di atas grid.
 *
 * Yang paling mudah terlupakan adalah yang ketiga. Pencacah dibaca dari kueri TERSENDIRI di
 * peladen, bukan dihitung dari baris yang tampil — sehingga membiarkannya akan membuat
 * pengguna melihat barisnya berpindah tab sementara angkanya tetap. Angka yang tidak sejalan
 * dengan daftarnya sendiri membuat seluruh layar diragukan.
 *
 * Karena pencacah datang BERSAMA daftar, menyegarkan daftarnya sudah menyegarkan keduanya —
 * dan itulah sebabnya keduanya memang dilayani satu permintaan.
 */
export function useKomunikasiCabangReply() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (input: { komunikasi: string; pesan: string }) =>
      await callAPI<{ komunikasi: string; pesan: string }>(
        `${PATH}/komunikasi/${encodeURIComponent(input.komunikasi)}/balas`,
        { token, portal, metode: 'POST', body: { pesan: input.pesan } },
      ),

    onSuccess: (_result, input) => {
      void queryClient.invalidateQueries({
        queryKey: keys.detail(portal, token, input.komunikasi),
      })

      // SELURUH daftar modul ini, bukan hanya tab dan halaman yang sedang terbuka: barisnya
      // meninggalkan satu tab dan masuk ke tab lain, dan halaman berapa ia mendarat tidak
      // dapat diketahui dari sini.
      void queryClient.invalidateQueries({
        queryKey: ['inbox-komunikasi-cabang', 'daftar'],
      })
    },
  })
}

/**
 * Hook tombol "Selesai Komunikasi".
 *
 * Akibatnya TIDAK DAPAT DIBATALKAN: percakapannya hilang dari kedua tab, dan sistem lama
 * tidak punya satu pun tindakan yang membukanya kembali. Layar karena itu meminta penegasan
 * lebih dulu — lihat KomunikasiCabangPage.
 *
 * Utasnya ikut disegarkan meski barisnya hilang dari daftar: seseorang yang sedang membuka
 * layar detail percakapan itu harus melihat keadaannya yang baru, bukan yang sudah usang.
 */
export function useKomunikasiCabangFinish() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (komunikasi: string) =>
      await callAPI<{ komunikasi: string; pesan: string }>(
        `${PATH}/komunikasi/${encodeURIComponent(komunikasi)}/selesai`,
        { token, portal, metode: 'POST' },
      ),

    onSuccess: (_result, komunikasi) => {
      void queryClient.invalidateQueries({
        queryKey: keys.detail(portal, token, komunikasi),
      })
      void queryClient.invalidateQueries({
        queryKey: ['inbox-komunikasi-cabang', 'daftar'],
      })
    },
  })
}

/**
 * Hook daftar cabang untuk pemilih tujuan pada form "Kirim Pesan".
 *
 * # Kenapa TERPISAH dari metadata layar
 *
 * Karena masa hidupnya berbeda. Bentuk grid dibaca dari kode dan tidak pernah berubah selama
 * aplikasi hidup, sehingga cache-nya `Infinity`. Daftar cabang datang dari basis data dan
 * dapat bertambah — menyatukannya akan memaksa keduanya punya masa cache yang sama.
 *
 * # Kenapa hanya diambil saat formnya terbuka
 *
 * Karena mayoritas pembukaan layar ini tidak berakhir dengan pengiriman pesan. Mengambilnya
 * bersama daftar berarti satu perjalanan jaringan tambahan pada setiap pembukaan, untuk
 * daftar yang hampir selalu tidak terpakai.
 */
export function useKomunikasiCabangBranches(enabled: boolean) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: keys.branches(portal, token),
    queryFn: () => callAPI<BranchListResponse>(`${PATH}/cabang`, { token, portal }),
    enabled: enabled && token !== null && portal !== null,

    // Daftar cabang berubah sangat jarang — ia master, bukan data transaksi. Lima menit
    // menghindari pengambilan ulang setiap kali form dibuka-tutup dalam satu sesi kerja.
    staleTime: 5 * 60 * 1000,
  })
}

/**
 * Hook tombol "Kirim Pesan" — pembuatan percakapan BARU.
 *
 * # Kenapa ia menyegarkan SELURUH daftar
 *
 * Karena percakapan baru mendarat di tab "Belum Dijawab", dan halaman berapa ia mendarat
 * tidak dapat diketahui dari sini — urutannya menurut tanggal, dan tanggalnya diisi basis
 * data.
 *
 * Kedua pencacah ikut tersegarkan karena keduanya datang BERSAMA daftar.
 */
export function useKomunikasiCabangSendMessage() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (input: NewMessageRequest) =>
      await callAPI<ActionResponse>(`${PATH}/pesan`, {
        token,
        portal,
        metode: 'POST',
        body: input,
      }),

    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: ['inbox-komunikasi-cabang', 'daftar'],
      })
    },
  })
}

/**
 * Hook tombol unduh.
 *
 * # Kenapa berkasnya diambil dengan fetch, bukan dengan tautan unduh biasa
 *
 * Karena `<a href>` dan `window.open` TIDAK membawa header — dan endpoint ini menuntut dua:
 * `Authorization` dan `X-Portal`. Satu-satunya cara memakai tautan biasa adalah menaruh
 * token di dalam alamat, dan itu ditolak dengan alasan yang sudah dicatat di
 * `api/client.ts`: nilai di URL ikut tercatat di riwayat peramban, log proxy, dan header
 * Referer.
 *
 * Di layar ini alasannya lebih tajam lagi: yang menentukan ISI berkas bukan hanya alamatnya
 * melainkan siapa yang memintanya — batas cabang diturunkan dari token itu sendiri.
 *
 * # Biaya yang disadari
 *
 * Peladen MENGALIRKAN berkasnya potong demi potong, tetapi peramban menampungnya utuh
 * sebagai Blob sebelum menyimpannya. Manfaat pengaliran karena itu tinggal di sisi peladen —
 * memorinya tetap datar — sedangkan memori peramban tumbuh sebesar berkasnya.
 */
export function useExportKomunikasiCabang() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useMutation({
    mutationFn: async (tab: string) => {
      const header: Record<string, string> = {}
      if (token) header['Authorization'] = `Bearer ${token}`
      if (portal) header[HEADER_PORTAL] = portal

      const params = new URLSearchParams()
      if (tab) params.set('tab', tab)

      const query = params.toString()
      const address = query ? `${PATH}/ekspor?${query}` : `${PATH}/ekspor`

      const response = await fetch(address, { headers: header })
      if (!response.ok) {
        // Galat dijawab sebagai JSON selama header belum terkirim; setelah itu tidak bisa
        // lagi. Yang dibaca di sini adalah kasus pertama — termasuk sumber cabang yang
        // sedang mati, yang dijawab 503 beserta alasannya.
        const body = (await response.json().catch(() => null)) as {
          pesan?: string
          detail?: { pesan?: string }[]
        } | null

        const detail = body?.detail?.map((item) => item.pesan).filter(Boolean)
        if (detail && detail.length > 0) throw new Error(detail.join(' '))

        throw new Error(body?.pesan ?? 'Berkas unduhan tidak dapat diambil.')
      }

      const blob = await response.blob()
      downloadBlob(blob, filenameOf(response) ?? 'komunikasi-cabang.csv')
    },
  })
}

/** buildPath menyusun alamat permintaan daftar beserta halamannya. */
function buildPath(tab: string, page: number): string {
  const params = new URLSearchParams()
  if (tab) params.set('tab', tab)
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
 * tab ditutup — dan pada layar yang dipakai sepanjang hari, setiap unduhan menumpuk memori
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
